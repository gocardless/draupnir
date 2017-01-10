package routes

import (
  "net/http/httptest"
  "net/http"
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestListImages(t *testing.T) {
  recorder := httptest.NewRecorder()
  req, err := http.NewRequest("GET", "/images", nil)
  if err != nil {
    t.Fatal(err)
  }
  handler := http.HandlerFunc(ListImages)
  handler.ServeHTTP(recorder, req)

  assert.Equal(t, http.StatusOK, recorder.Code)

  assert.Equal(t, "[]\n", string(recorder.Body.Bytes()))
}
