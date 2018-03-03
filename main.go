package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"reflect"

	"github.com/burntsushi/toml"
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
	"github.com/oklog/run"
	"github.com/pkg/errors"
)

type Config struct {
	Port                   int    `toml:"port"`
	InsecurePort           int    `toml:"insecure_port"`
	DatabaseUrl            string `toml:"database_url"`
	DataPath               string `toml:"data_path"`
	Environment            string `toml:"environment"`
	SharedSecret           string `toml:"shared_secret"`
	OauthRedirectUrl       string `toml:"oauth_redirect_url"`
	OauthClientId          string `toml:"oauth_client_id"`
	OauthClientSecret      string `toml:"oauth_client_secret"`
	TlsCertificatePath     string `toml:"tls_certificate_path"`
	TlsPrivateKeyPath      string `toml:"tls_private_key_path"`
	TrustedUserEmailDomain string `toml:"trusted_user_email_domain"`
	SentryDsn              string `toml:"sentry_dsn" required:"false"`
}

const ConfigFilePath = "/etc/draupnir/config.toml"

func loadConfig(path string) (Config, error) {
	var config Config
	file, err := os.Open(path)
	if err != nil {
		return config, errors.Wrap(err, fmt.Sprintf("No configuration file found at %s", ConfigFilePath))
	}

	_, err = toml.DecodeReader(file, &config)
	if err != nil {
		return config, errors.Wrap(err, "Could not parse configuration file")
	}

	return config, validateConfig(config)
}

func validateConfig(cfg Config) error {
	cfgValue := reflect.ValueOf(&cfg).Elem()
	cfgType := reflect.TypeOf(cfg)
	emptyFields := []string{}

	for i := 0; i < cfgValue.NumField(); i++ {
		field := cfgValue.Field(i)
		tag := cfgType.Field(i).Tag
		empty := reflect.Zero(field.Type())
		if tag.Get("required") == "false" {
			continue
		}
		if reflect.DeepEqual(field.Interface(), empty.Interface()) {
			emptyFields = append(emptyFields, tag.Get("toml"))
		}
	}

	if len(emptyFields) == 0 {
		return nil
	}

	return fmt.Errorf("Missing required fields: %v", emptyFields)
}

func main() {
	logger := log.With("app", "draupnir")

	c, err := loadConfig(ConfigFilePath)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not load config")
	}

	logger = log.With("environment", c.Environment)

	oauthConfig := createOauthConfig(c)
	authenticator := createAuthenticator(c, oauthConfig)

	db := connectToDatabase(c, logger)
	imageStore := createImageStore(db)
	instanceStore := createInstanceStore(db)

	executor := createExecutor(c)

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

	// Every request will be logged, and any error raised in serving the request
	// will also be logged.
	rootHandler := chain.
		New(routes.NewErrorHandler(logger)).
		Add(routes.NewRequestLogger(logger))

	// If Sentry is available, attach the Sentry middleware
	// This will report all errors to Sentry
	if c.SentryDsn != "" {
		sentryClient, err := raven.New(c.SentryDsn)
		if err != nil {
			logger.With("error", err.Error()).Fatal("Could not initialise sentry-raven client")
		}

		rootHandler = rootHandler.
			Add(routes.NewSentryReporter(sentryClient))
	}

	router := mux.NewRouter()

	// Healthcheck
	// We don't enforce a particulate API version on this route, because it should
	// be easy to hit to monitor the health of the system.
	router.Methods("GET").Path("/health_check").HandlerFunc(
		rootHandler.
			Add(routes.WithVersion).
			Add(routes.AsJSON).
			Resolve(routes.HealthCheck),
	)

	// OAuth
	// These routes are a bit special, because they don't accept or return JSON.
	// They're intended to be used through a web browser.
	router.Methods("GET").Path("/authenticate").HandlerFunc(
		rootHandler.
			Resolve(accessTokenRouteSet.Authenticate),
	)

	router.Methods("GET").Path("/oauth_callback").HandlerFunc(
		rootHandler.
			Add(routes.OauthErrorRenderer).
			Resolve(accessTokenRouteSet.Callback),
	)

	// Core API routes
	// These routes all accept and return JSON, and will enforce that the client
	// sends a compatible API version header.
	defaultChain := rootHandler.
		Add(routes.DefaultErrorRenderer).
		Add(routes.WithVersion).
		Add(routes.AsJSON).
		Add(routes.CheckAPIVersion(version.Version))

	// Access Tokens
	router.Methods("POST").Path("/access_tokens").HandlerFunc(
		defaultChain.Resolve(accessTokenRouteSet.Create),
	)

	// Images
	router.Methods("GET").Path("/images").HandlerFunc(
		defaultChain.Resolve(imageRouteSet.List),
	)

	router.Methods("POST").Path("/images").HandlerFunc(
		defaultChain.Resolve(imageRouteSet.Create),
	)

	router.Methods("GET").Path("/images/{id}").HandlerFunc(
		defaultChain.Resolve(imageRouteSet.Get),
	)

	router.Methods("POST").Path("/images/{id}/done").HandlerFunc(
		defaultChain.Resolve(imageRouteSet.Done),
	)

	router.Methods("DELETE").Path("/images/{id}").HandlerFunc(
		defaultChain.Resolve(imageRouteSet.Destroy),
	)

	// Instances
	router.Methods("GET").Path("/instances").HandlerFunc(
		defaultChain.Resolve(instanceRouteSet.List),
	)

	router.Methods("POST").Path("/instances").HandlerFunc(
		defaultChain.Resolve(instanceRouteSet.Create),
	)

	router.Methods("GET").Path("/instances/{id}").HandlerFunc(
		defaultChain.Resolve(instanceRouteSet.Get),
	)

	router.Methods("DELETE").Path("/instances/{id}").HandlerFunc(
		defaultChain.Resolve(instanceRouteSet.Destroy),
	)

	router.Methods("DELETE").Path("/instances/{id}").HandlerFunc(
		defaultChain.Resolve(instanceRouteSet.Destroy),
	)

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
		logger.Fatal(errors.Wrap(err, "could not start HTTP servers").Error())
	}
}

func connectToDatabase(c Config, logger log.Logger) *sql.DB {
	db, err := sql.Open("postgres", c.DatabaseUrl)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not connect to database")
	}
	return db
}

func createOauthConfig(c Config) oauth2.Config {
	return oauth2.Config{
		ClientID:     c.OauthClientId,
		ClientSecret: c.OauthClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://www.googleapis.com/oauth2/v4/token",
		},
		RedirectURL: c.OauthRedirectUrl,
	}
}

func createAuthenticator(c Config, oauthConfig oauth2.Config) auth.Authenticator {
	authenticator := auth.GoogleAuthenticator{
		OAuthClient:            auth.GoogleOAuthClient{Config: &oauthConfig},
		SharedSecret:           c.SharedSecret,
		TrustedUserEmailDomain: c.TrustedUserEmailDomain,
	}
	if c.Environment == "test" {
		authenticator.OAuthClient = auth.FakeOAuthClient{}
	}
	return authenticator
}

func createImageStore(db *sql.DB) store.ImageStore {
	return store.DBImageStore{DB: db}
}

func createInstanceStore(db *sql.DB) store.InstanceStore {
	return store.DBInstanceStore{DB: db}
}

func createExecutor(c Config) exec.Executor {
	return exec.OSExecutor{DataPath: c.DataPath}
}
