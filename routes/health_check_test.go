package routes

import (
  "net/http/httptest"
  "net/http"
  "testing"
  . "github.com/onsi/gomega"
)

func TestHealthCheck(t *testing.T) {
  RegisterTestingT(t)
  recorder := httptest.NewRecorder()
  req, err := http.NewRequest("GET", "/health_check", nil)
  if err != nil {
    t.Fatal(err)
  }
  handler := http.HandlerFunc(HealthCheck)
  handler.ServeHTTP(recorder, req)

  Expect(recorder.Code).To(Equal(http.StatusOK))

  Expect(string(recorder.Body.Bytes())).To(Equal("OK"))
}
