package main

import (
  "github.com/kelseyhightower/envconfig"
  "github.com/gorilla/mux"
  "log"
  "fmt"
  "net/http"
  "github.com/gocardless/draupnir/routes"
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

  http.Handle("/", router)

  err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
  if err != nil {
    log.Fatal(err.Error())
  }
}
