package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/evalphobia/logrus_sentry"
	"github.com/prometheus/common/log"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/gocardless/draupnir/auth"
	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/routes"
	"github.com/gocardless/draupnir/routes/chain"
	"github.com/gocardless/draupnir/store"
	"github.com/gocardless/draupnir/version"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/oklog/run"
)

type Config struct {
	Port                   int    `required:"true"`
	InsecurePort           int    `required:"false" default:"8080"`
	DatabaseUrl            string `required:"true" split_words:"true"`
	DataPath               string `required:"true" split_words:"true"`
	Environment            string `required:"false"`
	SharedSecret           string `required:"true" split_words:"true"`
	OauthRedirectUrl       string `required:"true" split_words:"true"`
	OauthClientId          string `required:"true" split_words:"true"`
	OauthClientSecret      string `required:"true" split_words:"true"`
	TlsCertificatePath     string `required:"true" split_words:"true"`
	TlsPrivateKeyPath      string `required:"true" split_words:"true"`
	TrustedUserEmailDomain string `required:"true" split_words:"true"`
	SentryDsn              string `required:"false" split_words:"true"`
}

func main() {
	var c Config
	err := envconfig.Process("draupnir", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := sql.Open("postgres", c.DatabaseUrl)
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err.Error())
	}

	logger := log.With("app", "draupnir")

	if c.SentryDsn != "" {
		hook, err := logrus_sentry.NewSentryHook(c.SentryDsn, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})

		if err != nil {
			logger.With("error", err.Error()).Fatal("Could not initialise sentry-raven client")
		}

		hook.StacktraceConfiguration.Enable = true
		log.AddHook(hook)
	}

	oauthConfig := oauth2.Config{
		ClientID:     c.OauthClientId,
		ClientSecret: c.OauthClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://www.googleapis.com/oauth2/v4/token",
		},
		RedirectURL: c.OauthRedirectUrl,
	}

	executor := exec.OSExecutor{DataPath: c.DataPath}

	authenticator := auth.GoogleAuthenticator{
		OAuthClient:            auth.GoogleOAuthClient{Config: &oauthConfig},
		SharedSecret:           c.SharedSecret,
		TrustedUserEmailDomain: c.TrustedUserEmailDomain,
	}
	if c.Environment == "test" {
		authenticator.OAuthClient = auth.FakeOAuthClient{}
	}

	imageStore := store.DBImageStore{DB: db}
	instanceStore := store.DBInstanceStore{DB: db}

	imageRouteSet := routes.Images{
		ImageStore:    imageStore,
		InstanceStore: instanceStore,
		Executor:      executor,
		Authenticator: authenticator,
		Logger:        logger.With("resource", "images"),
	}

	instanceRouteSet := routes.Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
		Authenticator: authenticator,
		Logger:        logger.With("resource", "instances"),
	}

	accessTokenRouteSet := routes.AccessTokens{
		Callbacks: make(map[string]chan routes.OAuthCallback),
		Client:    &oauthConfig,
		Logger:    logger.With("resource", "access_tokens"),
	}

	router := mux.NewRouter()

	asJSON := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			next(w, r)
		}
	}
	withVersion := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Draupnir-Version", version.Version)
			next(w, r)
		}
	}

	defaultChain := chain.
		New().
		Add(routes.LogRequest).
		Add(withVersion).
		Add(asJSON).
		Add(routes.CheckAPIVersion).
		ToMiddleware()

	chain.
		FromRoute(router.Methods("GET").Path("/health_check")).
		Add(routes.LogRequest).
		Add(withVersion).
		Add(asJSON).
		ToRoute(routes.HealthCheck)

	chain.
		FromRoute(router.Methods("GET").Path("/authenticate")).
		ToRoute(accessTokenRouteSet.Authenticate)

	chain.
		FromRoute(router.Methods("GET").Path("/oauth_callback")).
		ToRoute(accessTokenRouteSet.Callback)

	chain.
		FromRoute(router.Methods("POST").Path("/access_tokens")).
		Add(defaultChain).
		ToRoute(accessTokenRouteSet.Create)

	chain.
		FromRoute(router.Methods("GET").Path("/images")).
		Add(defaultChain).
		ToRoute(imageRouteSet.List)

	chain.
		FromRoute(router.Methods("POST").Path("/images")).
		Add(defaultChain).
		ToRoute(imageRouteSet.Create)

	chain.
		FromRoute(router.Methods("GET").Path("/images/{id}")).
		Add(defaultChain).
		ToRoute(imageRouteSet.Get)

	chain.
		FromRoute(router.Methods("POST").Path("/images/{id}/done")).
		Add(defaultChain).
		ToRoute(imageRouteSet.Done)

	chain.
		FromRoute(router.Methods("DELETE").Path("/images/{id}")).
		Add(defaultChain).
		ToRoute(imageRouteSet.Destroy)

	chain.
		FromRoute(router.Methods("GET").Path("/instances")).
		Add(defaultChain).
		ToRoute(instanceRouteSet.List)

	chain.
		FromRoute(router.Methods("POST").Path("/instances")).
		Add(defaultChain).
		ToRoute(instanceRouteSet.Create)

	chain.
		FromRoute(router.Methods("GET").Path("/instances/{id}")).
		Add(defaultChain).
		ToRoute(instanceRouteSet.Get)

	chain.
		FromRoute(router.Methods("DELETE").Path("/instances/{id}")).
		Add(defaultChain).
		ToRoute(instanceRouteSet.Destroy)

	var g run.Group

	// The default server for draupnir which will listen on TLS
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", c.Port),
		Handler: router,
	}

	g.Add(
		func() error { return server.ListenAndServeTLS(c.TlsCertificatePath, c.TlsPrivateKeyPath) },
		func(error) { server.Shutdown(context.Background()) },
	)

	// We then listen for insecure connections on localhost, allowing connections from
	// within the host without requiring the user to explicitly ignore certificates.
	serverInsecure := http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", c.InsecurePort),
		Handler: router,
	}

	g.Add(
		func() error { return serverInsecure.ListenAndServe() },
		func(error) { serverInsecure.Shutdown(context.Background()) },
	)

	if err := g.Run(); err != nil {
		logger.Fatal(err.Error())
	}
}
