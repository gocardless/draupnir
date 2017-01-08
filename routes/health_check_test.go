package routes

import (
  "net/http/httptest"
  "net/http"
  "testing"
)

func TestHealthCheck(t *testing.T) {
  recorder := httptest.NewRecorder()
  req, err := http.NewRequest("GET", "/health_check", nil)
  if err != nil {
    t.Fatal(err)
  }
  handler := http.HandlerFunc(HealthCheck)
  handler.ServeHTTP(recorder, req)

  if recorder.Code != http.StatusOK {
    t.Errorf(
      "wrong status code:\n\texpected %v\n\tgot %v",
      http.StatusCreated,
      recorder.Code,
    )
  }

  body := string(recorder.Body.Bytes())
  expected := "OK"
  if body != expected {
    t.Errorf("wrong body:\n\texpected %s\n\tgot %s", expected, body)
  }
}
