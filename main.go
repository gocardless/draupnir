package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	raven "github.com/getsentry/raven-go"
	"github.com/prometheus/common/log"
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
	logger := log.With("app", "draupnir")

	var c Config
	err := envconfig.Process("draupnir", &c)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not read config")
	}

	logger = log.With("environment", c.Environment)

	db, err := sql.Open("postgres", c.DatabaseUrl)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not connect to database")
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
	}

	instanceRouteSet := routes.Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
		Authenticator: authenticator,
	}

	accessTokenRouteSet := routes.AccessTokens{
		Callbacks: make(map[string]chan routes.OAuthCallback),
		Client:    &oauthConfig,
	}

	errorHandler := routes.NewErrorHandler(logger)

	if c.SentryDsn != "" {
		sentryClient, err := raven.New(c.SentryDsn)
		if err != nil {
			logger.With("error", err.Error()).Fatal("Could not initialise sentry-raven client")
		}

		errorHandler = routes.NewSentryErrorHandler(logger, sentryClient)
	}

	asJSON := func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Content-Type", "application/json")
			next(w, r)
			return nil
		}
	}
	withVersion := func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("Draupnir-Version", version.Version)
			next(w, r)
			return nil
		}
	}

	logRequest := routes.NewRequestLogger(logger)
	withErrorHandler := chain.New(errorHandler)

	defaultChain := withErrorHandler.
		Add(logRequest).
		Add(withVersion).
		Add(asJSON).
		Add(routes.CheckAPIVersion(version.Version))

	router := mux.NewRouter()

	withErrorHandler.
		Route(router.Methods("GET").Path("/health_check")).
		Add(logRequest).
		Add(withVersion).
		Add(asJSON).
		Resolve(routes.HealthCheck)

	withErrorHandler.
		Route(router.Methods("GET").Path("/authenticate")).
		Add(logRequest).
		Resolve(accessTokenRouteSet.Authenticate)

	chain.New(routes.HandleOAuthError).
		Route(router.Methods("GET").Path("/oauth_callback")).
		Add(logRequest).
		Resolve(accessTokenRouteSet.Callback)

	defaultChain.
		Route(router.Methods("POST").Path("/access_tokens")).
		Resolve(accessTokenRouteSet.Create)

	defaultChain.
		Route(router.Methods("GET").Path("/images")).
		Resolve(imageRouteSet.List)

	defaultChain.
		Route(router.Methods("POST").Path("/images")).
		Resolve(imageRouteSet.Create)

	defaultChain.
		Route(router.Methods("GET").Path("/images/{id}")).
		Resolve(imageRouteSet.Get)

	defaultChain.
		Route(router.Methods("POST").Path("/images/{id}/done")).
		Resolve(imageRouteSet.Done)

	defaultChain.
		Route(router.Methods("DELETE").Path("/images/{id}")).
		Resolve(imageRouteSet.Destroy)

	defaultChain.
		Route(router.Methods("GET").Path("/instances")).
		Resolve(instanceRouteSet.List)

	defaultChain.
		Route(router.Methods("POST").Path("/instances")).
		Resolve(instanceRouteSet.Create)

	defaultChain.
		Route(router.Methods("GET").Path("/instances/{id}")).
		Resolve(instanceRouteSet.Get)

	defaultChain.
		Route(router.Methods("DELETE").Path("/instances/{id}")).
		Resolve(instanceRouteSet.Destroy)

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
