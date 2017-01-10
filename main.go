package main

import (
  "github.com/kelseyhightower/envconfig"
  "github.com/gorilla/mux"
  "github.com/gocardless/draupnir/routes"
  "log"
  "fmt"
  "net/http"
)

var version string

type Config struct {
  Port int `required:"true"`
}

func main() {
  var c Config
  err := envconfig.Process("draupnir", &c)
  if err != nil {
    log.Fatal(err.Error())
  }

  router := mux.NewRouter()
  router.HandleFunc("/health_check", routes.HealthCheck)
  router.HandleFunc("/images", routes.ListImages)

  http.Handle("/", router)

  err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
  if err != nil {
    log.Fatal(err.Error())
  }
}
