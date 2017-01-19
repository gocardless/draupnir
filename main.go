package main

import (
	"database/sql"
	"fmt"
	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/routes"
	"github.com/gocardless/draupnir/store"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"log"
	"net/http"
)

var version string

type Config struct {
	Port        int    `required:"true"`
	DatabaseUrl string `required:"true" split_words:"true"`
}

func main() {
	var c Config
	err := envconfig.Process("draupnir", &c)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("PORT: %d", c.Port)
	log.Printf("DATABASE_URL: %s", c.DatabaseUrl)

	db, err := sql.Open("postgres", c.DatabaseUrl)
	if err != nil {
		log.Fatalf("Cannot connect to database: %s", err.Error())
	}

	imageStore := store.DBImageStore{DB: db}

	imageRouteSet := routes.Images{
		Store:    imageStore,
		Executor: exec.OSExecutor{},
	}

	router := mux.NewRouter()
	router.HandleFunc("/health_check", routes.HealthCheck)
	router.HandleFunc("/images", imageRouteSet.List).Methods("GET")
	router.HandleFunc("/images", imageRouteSet.Create).Methods("POST")
	router.HandleFunc("/images/{id}/done", imageRouteSet.Done).Methods("POST")

	http.Handle("/", router)

	err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
	if err != nil {
		log.Fatal(err.Error())
	}
}
