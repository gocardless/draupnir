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
		OAuthClient:  auth.GoogleOAuthClient{},
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
	router.HandleFunc("/health_check", routes.HealthCheck)

	router.HandleFunc("/access_tokens", accessTokenRouteSet.Create).Methods("POST")
	router.HandleFunc("/oauth_callback", accessTokenRouteSet.Callback).Methods("GET")
	router.HandleFunc("/authenticate", accessTokenRouteSet.Authenticate).Methods("GET")

	router.HandleFunc("/images", imageRouteSet.List).Methods("GET")
	router.HandleFunc("/images", imageRouteSet.Create).Methods("POST")
	router.HandleFunc("/images/{id}", imageRouteSet.Get).Methods("GET")
	router.HandleFunc("/images/{id}/done", imageRouteSet.Done).Methods("POST")
	router.HandleFunc("/images/{id}", imageRouteSet.Destroy).Methods("DELETE")

	router.HandleFunc("/instances", instanceRouteSet.List).Methods("GET")
	router.HandleFunc("/instances", instanceRouteSet.Create).Methods("POST")
	router.HandleFunc("/instances/{id}", instanceRouteSet.Get).Methods("GET")
	router.HandleFunc("/instances/{id}", instanceRouteSet.Destroy).Methods("DELETE")

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
