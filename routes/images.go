package routes

import (
  "net/http"
  "encoding/json"
  "github.com/gocardless/draupnir/models"
  "github.com/gocardless/draupnir/store"
  "time"
)

type Images struct {
  Store store.ImageStore
}

func (i Images) List(w http.ResponseWriter, r *http.Request) {
  mockResponse := make([]models.Image, 0)
  time, err := time.Parse(time.RFC3339, "2016-01-01T12:33:44.567Z")
  mockResponse = append(
    mockResponse,
    models.Image{ID: 1, BackedUpAt: time, Ready: false},
  )
  err = json.NewEncoder(w).Encode(mockResponse)
  if err != nil {
    http.Error(w, "json encoding failed", http.StatusInternalServerError)
  }
}
