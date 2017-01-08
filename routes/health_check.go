package routes

import (
  "net/http"
  "fmt"
)

func HealthCheck(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusOK)
  fmt.Fprintf(w, "OK")
}
