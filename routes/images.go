package routes

import (
  "net/http"
  "encoding/json"
)

func ListImages(w http.ResponseWriter, r *http.Request) {
  mockResponse := make([]interface{}, 0)
  err := json.NewEncoder(w).Encode(mockResponse)
  if err != nil {
    http.Error(w, "json encoding failed", http.StatusInternalServerError)
  }
}
