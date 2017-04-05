package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/routes"
	"github.com/gocardless/draupnir/store"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
)

var version string

type Config struct {
	Port        int    `required:"true"`
	DatabaseUrl string `required:"true" split_words:"true"`
	RootDir     string `required:"true" split_words:"true"`
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

	imageStore := store.DBImageStore{DB: db}

	executor := exec.OSExecutor{RootDir: c.RootDir}
	imageRouteSet := routes.Images{
		Store:    imageStore,
		Executor: executor,
	}

	instanceStore := store.DBInstanceStore{DB: db}

	instanceRouteSet := routes.Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
	}

	router := mux.NewRouter()
	router.HandleFunc("/health_check", routes.HealthCheck)

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

	err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
