package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/gocardless/draupnir/auth"
	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/routes"
	"github.com/gocardless/draupnir/routes/chain"
	"github.com/gocardless/draupnir/store"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
)

var version string

type Config struct {
	Port               int    `required:"true"`
	DatabaseUrl        string `required:"true" split_words:"true"`
	DataPath           string `required:"true" split_words:"true"`
	Environment        string `required:"false"`
	SharedSecret       string `required:"true" split_words:"true"`
	OauthRedirectUrl   string `required:"true" split_words:"true"`
	OauthClientId      string `required:"true" split_words:"true"`
	OauthClientSecret  string `required:"true" split_words:"true"`
	TlsCertificatePath string `required:"true" split_words:"true"`
	TlsPrivateKeyPath  string `required:"true" split_words:"true"`
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
		OAuthClient:  auth.GoogleOAuthClient{Config: &oauthConfig},
		SharedSecret: c.SharedSecret,
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

	router := mux.NewRouter()

	withVersion := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Draupnir-Version", version)
	}
	asJSON := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}
	versionedJSON := chain.New().Add(withVersion).Add(asJSON).Resolve()

	chain.
		FromRoute(router.Methods("GET").Path("/health_check")).
		Add(versionedJSON).
		Add(routes.HealthCheck).
		ToRoute()

	chain.
		FromRoute(router.Methods("GET").Path("/oauth_callback")).
		Add(accessTokenRouteSet.Callback).
		ToRoute()

	chain.
		FromRoute(router.Methods("POST").Path("/access_tokens")).
		Add(versionedJSON).
		Add(accessTokenRouteSet.Create).
		ToRoute()

	chain.
		FromRoute(router.Methods("GET").Path("/authenticate")).
		Add(accessTokenRouteSet.Authenticate).
		ToRoute()

	chain.
		FromRoute(router.Methods("GET").Path("/images")).
		Add(versionedJSON).
		Add(imageRouteSet.List).
		ToRoute()

	chain.
		FromRoute(router.Methods("POST").Path("/images")).
		Add(versionedJSON).
		Add(imageRouteSet.Create).
		ToRoute()

	chain.
		FromRoute(router.Methods("GET").Path("/images/{id}")).
		Add(versionedJSON).
		Add(imageRouteSet.Get).
		ToRoute()

	chain.
		FromRoute(router.Methods("POST").Path("/images/{id}/done")).
		Add(versionedJSON).
		Add(imageRouteSet.Done).
		ToRoute()

	chain.
		FromRoute(router.Methods("DELETE").Path("/images/{id}")).
		Add(versionedJSON).
		Add(imageRouteSet.Destroy).
		ToRoute()

	chain.
		FromRoute(router.Methods("GET").Path("/instances")).
		Add(versionedJSON).
		Add(instanceRouteSet.List).
		ToRoute()

	chain.
		FromRoute(router.Methods("POST").Path("/instances")).
		Add(versionedJSON).
		Add(instanceRouteSet.Create).
		ToRoute()

	chain.
		FromRoute(router.Methods("GET").Path("/instances/{id}")).
		Add(versionedJSON).
		Add(instanceRouteSet.Get).
		ToRoute()

	chain.
		FromRoute(router.Methods("DELETE").Path("/instances/{id}")).
		Add(versionedJSON).
		Add(instanceRouteSet.Destroy).
		ToRoute()

	http.Handle("/", router)

	err = http.ListenAndServeTLS(
		fmt.Sprintf(":%d", c.Port),
		c.TlsCertificatePath,
		c.TlsPrivateKeyPath,
		nil,
	)
	if err != nil {
		log.Fatal(err.Error())
	}
}
