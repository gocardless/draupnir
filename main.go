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

// HTTPConfig holds Draupnir's HTTP configuration
type HTTPConfig struct {
	Port               int    `toml:"port"`
	InsecurePort       int    `toml:"insecure_port"`
	TLSCertificatePath string `toml:"tls_certificate"`
	TLSPrivateKeyPath  string `toml:"tls_private_key"`
}

// OAuthConfig holds Draupnir's OAuth configuration
type OAuthConfig struct {
	RedirectURL  string `toml:"redirect_url"`
	ClientID     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

// Config holds all Draupnir configuration
type Config struct {
	DatabaseURL            string      `toml:"database_url"`
	DataPath               string      `toml:"data_path"`
	Environment            string      `toml:"environment"`
	SharedSecret           string      `toml:"shared_secret"`
	TrustedUserEmailDomain string      `toml:"trusted_user_email_domain"`
	SentryDsn              string      `toml:"sentry_dsn" required:"false"`
	HTTPConfig             HTTPConfig  `toml:"http"`
	OAuthConfig            OAuthConfig `toml:"oauth"`
}

// ConfigFilePath is the expected path of the Draupnir configuration file
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

	err = validateConfig(config)
	if err != nil {
		return config, errors.Wrap(err, "Invalid configuration")
	}

	return config, nil
}

func validateConfig(cfg Config) error {
	cfgValue := reflect.ValueOf(&cfg).Elem()
	cfgType := reflect.TypeOf(cfg)
	emptyFields := emptyConfigFields(cfgValue, cfgType)
	if len(emptyFields) > 0 {
		return fmt.Errorf("Missing required fields: %v", emptyFields)
	}
	return nil
}

func emptyConfigFields(val reflect.Value, ty reflect.Type) []string {
	emptyFields := []string{}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := ty.Field(i).Tag
		empty := reflect.Zero(field.Type())
		if tag.Get("required") == "false" {
			continue
		}
		if reflect.DeepEqual(field.Interface(), empty.Interface()) {
			emptyFields = append(emptyFields, tag.Get("toml"))
		}
		if field.Type().Kind() == reflect.Struct {
			emptySubFields := emptyConfigFields(field, ty.Field(i).Type)
			for i := 0; i < len(emptySubFields); i++ {
				emptyFields = append(emptyFields, fmt.Sprintf("%s.%s", tag.Get("toml"), emptySubFields[i]))
			}
		}
	}

	return emptyFields
}

func main() {
	logger := log.With("app", "draupnir")

	logger.Info("Loading config file ", ConfigFilePath)
	cfg, err := loadConfig(ConfigFilePath)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not load configuration")
	}
	logger.Info("Configuration successfully loaded")

	logger = log.With("environment", cfg.Environment)

	oauthConfig := createOauthConfig(cfg.OAuthConfig)
	authenticator := createAuthenticator(cfg, oauthConfig)

	db := connectToDatabase(cfg, logger)
	imageStore := createImageStore(db)
	instanceStore := createInstanceStore(db)

	executor := createExecutor(cfg)

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
	if cfg.SentryDsn != "" {
		sentryClient, err := raven.New(cfg.SentryDsn)
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
		Addr:    fmt.Sprintf(":%d", cfg.HTTPConfig.Port),
		Handler: router,
	}

	g.Add(
		func() error {
			return server.ListenAndServeTLS(cfg.HTTPConfig.TLSCertificatePath, cfg.HTTPConfig.TLSPrivateKeyPath)
		},
		func(error) { server.Shutdown(context.Background()) },
	)

	// We then listen for insecure connections on localhost, allowing connections from
	// within the host without requiring the user to explicitly ignore certificates.
	serverInsecure := http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", cfg.HTTPConfig.InsecurePort),
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
	db, err := sql.Open("postgres", c.DatabaseURL)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not connect to database")
	}
	return db
}

func createOauthConfig(c OAuthConfig) oauth2.Config {
	return oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://www.googleapis.com/oauth2/v4/token",
		},
		RedirectURL: c.RedirectURL,
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
