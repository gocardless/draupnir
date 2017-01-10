package main

import (
  "github.com/kelseyhightower/envconfig"
  "github.com/gorilla/mux"
  "github.com/gocardless/draupnir/routes"
  "github.com/gocardless/draupnir/store"
  "log"
  "fmt"
  "net/http"
  "database/sql"
)

var version string

type Config struct {
  Port int `required:"true"`
  DatabaseURL string `require:"true"`
}

func main() {
  var c Config
  err := envconfig.Process("draupnir", &c)
  if err != nil {
    log.Fatal(err.Error())
  }

  db, err := sql.Open("postgres", c.DatabaseURL)
  if err != nil {
    log.Fatalf("Cannot connect to database: %s", err.Error())
  }

  imageStore := store.DBImageStore{DB: db}

  imageRouteSet := routes.Images{Store: imageStore}

  router := mux.NewRouter()
  router.HandleFunc("/health_check", routes.HealthCheck)
  router.HandleFunc("/images", imageRouteSet.List)

  http.Handle("/", router)

  err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
  if err != nil {
    log.Fatal(err.Error())
  }
}
