package main

import (
  "github.com/kelseyhightower/envconfig"
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

  http.HandleFunc("/health_check", func (w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "OK")
  })

  err = http.ListenAndServe(fmt.Sprintf(":%d", c.Port), nil)
  if err != nil {
    log.Fatal(err.Error())
  }
}
