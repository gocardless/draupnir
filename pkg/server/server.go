package server

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/gocardless/draupnir/pkg/exec"
	"github.com/gocardless/draupnir/pkg/server/api/auth"
	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/gocardless/draupnir/pkg/server/api/middleware"
	"github.com/gocardless/draupnir/pkg/server/api/routes"
	"github.com/gocardless/draupnir/pkg/server/config"
	"github.com/gocardless/draupnir/pkg/store"
	"github.com/gocardless/draupnir/pkg/version"
	"github.com/gorilla/mux"
	rungroup "github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"golang.org/x/oauth2"
)

// ConfigFilePath is the expected path of the server configuration file
const ConfigFilePath = "/etc/draupnir/config.toml"

// Run starts the draupnir server
// Any error returned is fatal
func Run(logger log.Logger) error {
	logger.With("config", ConfigFilePath).Info("Loading config file")
	cfg, err := config.Load(ConfigFilePath)
	if err != nil {
		return errors.Wrap(err, "Could not load configuration")
	}

	trustedProxies, err := parseTrustedProxies(cfg.TrustedProxyCIDRs)
	if err != nil {
		return errors.Wrap(err, "failed to parse trusted proxes")
	}

	logger.Info("Configuration successfully loaded")

	logger = log.With("environment", cfg.Environment)

	oauthConfig := createOauthConfig(cfg.OAuthConfig)
	authenticator := createAuthenticator(cfg, oauthConfig)
	executor := createExecutor(cfg)

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return errors.Wrap(err, "Could not connect to database")
	}
	imageStore := createImageStore(db)
	instanceStore := createInstanceStore(db, cfg)
	whitelistedAddressStore := createWhitelistedAddressStore(db)

	sentryClient, err := raven.New(cfg.SentryDsn)
	if err != nil {
		return errors.Wrap(err, "Could not initialise sentry-raven client")
	}

	// Setup the IP address whitelisting component.
	// This is optional, it's useful to be able to disable this in environments
	// where iptables is not available (e.g. integration tests).
	var whitelister *IPAddressWhitelister
	var whitelisterTriggerFunc func(string)

	if cfg.EnableWhitelisting {
		whitelister = NewIPAddressWhitelister(logger.With("component", "whitelister"), sentryClient, whitelistedAddressStore)
		whitelisterTriggerFunc = whitelister.TriggerReconcile
	} else {
		whitelisterTriggerFunc = func(s string) {
			logger.Debugf("IP whitelisting disabled, skipping trigger: %s", s)
		}
	}

	imageRouteSet := routes.Images{
		ImageStore:    imageStore,
		InstanceStore: instanceStore,
		Executor:      executor,
	}

	instanceRouteSet := routes.Instances{
		InstanceStore:           instanceStore,
		ImageStore:              imageStore,
		WhitelistedAddressStore: whitelistedAddressStore,
		ApplyWhitelist:          whitelisterTriggerFunc,
		Executor:                executor,
		MinInstancePort:         cfg.MinInstancePort,
		MaxInstancePort:         cfg.MaxInstancePort,
	}

	accessTokenRouteSet := routes.AccessTokens{
		Callbacks: make(map[string]chan routes.OAuthCallback),
		Client:    &oauthConfig,
	}

	router := mux.NewRouter()

	// Every request will be logged, and any error raised in serving the request
	// will also be logged.
	rootHandler := chain.
		New(middleware.NewErrorHandler(logger)).
		Add(middleware.RecordUserIPAddress(logger, trustedProxies, cfg.UseXForwardedFor)).
		Add(middleware.NewRequestLogger(logger))

	rootHandler = rootHandler.
		Add(middleware.NewSentryReporter(sentryClient))

	// Healthcheck
	// We don't enforce a particular API version on this route, because it should
	// be easy to hit to monitor the health of the system.
	router.Methods("GET").Path("/health_check").HandlerFunc(
		rootHandler.
			Add(middleware.WithVersion).
			Add(middleware.AsJSON).
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
		Add(middleware.DefaultErrorRenderer).
		Add(middleware.WithVersion).
		Add(middleware.AsJSON).
		Add(middleware.CheckAPIVersion(version.Version)).
		Add(middleware.Authenticate(authenticator))

	// Access Tokens
	// This route is hit before the user is authenticated, so we don't use the
	// Authenticate middleware
	router.Methods("POST").Path("/access_tokens").HandlerFunc(
		rootHandler.
			Add(middleware.DefaultErrorRenderer).
			Add(middleware.WithVersion).
			Add(middleware.AsJSON).
			Add(middleware.CheckAPIVersion(version.Version)).
			Resolve(accessTokenRouteSet.Create),
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

	var g rungroup.Group

	if cfg.HTTPConfig.SecureListenAddress != "" {
		// The default server for draupnir which will listen on TLS
		server := http.Server{
			Addr:    cfg.HTTPConfig.SecureListenAddress,
			Handler: router,
		}

		g.Add(
			func() error {
				return server.ListenAndServeTLS(cfg.HTTPConfig.TLSCertificatePath, cfg.HTTPConfig.TLSPrivateKeyPath)
			},
			func(error) { server.Shutdown(context.Background()) },
		)
	}

	if cfg.HTTPConfig.InsecureListenAddress != "" {
		// If configured, then allow connections via a non-TLS port.
		serverInsecure := http.Server{
			Addr:    cfg.HTTPConfig.InsecureListenAddress,
			Handler: router,
		}

		g.Add(
			func() error { return serverInsecure.ListenAndServe() },
			func(error) { serverInsecure.Shutdown(context.Background()) },
		)
	}

	if cfg.HTTPConfig.SecureListenAddress == "" && cfg.HTTPConfig.InsecureListenAddress == "" {
		return errors.New("Neither a secure or insecure listen was address specified")
	}

	{
		// We clean out old instances that have invalid tokens periodically as access
		// to the PostgreSQL instances only relies on certificate authentication. This
		// means that is situations, such as a user being offboarded, they will lose
		// access to the draupnir, but not their instances.
		logger = logger.With("component", "cleaner")

		instanceCleaner := NewInstanceCleaner(logger, sentryClient, instanceStore, executor, authenticator)
		cleanInterval, err := time.ParseDuration(cfg.CleanInterval)
		if err != nil {
			return errors.Wrap(err, "invalid clean interval")
		}

		cleanerCtx, cleanerCancel := context.WithCancel(context.Background())

		g.Add(
			func() error { return instanceCleaner.Start(cleanerCtx, cleanInterval) },
			func(error) { cleanerCancel() },
		)
	}

	if cfg.EnableWhitelisting {
		whitelisterInterval, err := time.ParseDuration(cfg.WhitelisterInterval)
		if err != nil {
			return errors.Wrap(err, "invalid whitelister update interval")
		}

		whitelisterCtx, whitelisterCancel := context.WithCancel(context.Background())

		g.Add(
			func() error { return whitelister.Start(whitelisterCtx, whitelisterInterval) },
			func(error) { whitelisterCancel() },
		)
	}

	if err := g.Run(); err != nil {
		return errors.Wrap(err, "could not start HTTP servers")
	}
	return nil
}

func createOauthConfig(c config.OAuthConfig) oauth2.Config {
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

func parseTrustedProxies(cidrs []string) ([]*net.IPNet, error) {
	var trusted []*net.IPNet

	for _, c := range cidrs {
		_, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			return nil, err
		}

		trusted = append(trusted, ipnet)
	}

	return trusted, nil
}

func createAuthenticator(c config.Config, oauthConfig oauth2.Config) auth.Authenticator {
	authenticator := auth.GoogleAuthenticator{
		OAuthClient:            auth.GoogleOAuthClient{Config: &oauthConfig},
		SharedSecret:           c.SharedSecret,
		TrustedUserEmailDomain: c.TrustedUserEmailDomain,
	}
	if c.Environment == "test" {
		authenticator.OAuthClient = auth.IntegrationTestOAuthClient{}
	}
	return authenticator
}

func createImageStore(db *sql.DB) store.ImageStore {
	return store.DBImageStore{DB: db}
}

func createInstanceStore(db *sql.DB, cfg config.Config) store.InstanceStore {
	return store.DBInstanceStore{DB: db, PublicHostname: cfg.PublicHostname}
}

func createWhitelistedAddressStore(db *sql.DB) store.WhitelistedAddressStore {
	return store.DBWhitelistedAddressStore{DB: db}
}

func createExecutor(c config.Config) exec.Executor {
	return exec.OSExecutor{DataPath: c.DataPath}
}
