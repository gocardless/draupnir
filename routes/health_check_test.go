package routes

import (
  "net/http/httptest"
  "net/http"
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
  recorder := httptest.NewRecorder()
  req, err := http.NewRequest("GET", "/health_check", nil)
  if err != nil {
    t.Fatal(err)
  }
  handler := http.HandlerFunc(HealthCheck)
  handler.ServeHTTP(recorder, req)

  assert.Equal(t, recorder.Code, http.StatusOK)

  assert.Equal(t, string(recorder.Body.Bytes()), "OK")
}
