package routes

import (
  "net/http/httptest"
  "net/http"
  "testing"
  "time"
  "github.com/stretchr/testify/assert"
  "github.com/gocardless/draupnir/models"
)

type FakeImageStore struct {}
func (s FakeImageStore) List() ([]models.Image, error) {
  loc, err := time.LoadLocation("UTC")
  if err != nil {
    panic(err.Error())
  }
  timestamp := time.Date(2016, 1, 1, 12, 33, 44, 567000, loc)
  return []models.Image{
    models.Image{ID: 1, BackedUpAt: timestamp, Ready: false},
  }, nil
}

func TestListImages(t *testing.T) {
  recorder := httptest.NewRecorder()
  req, err := http.NewRequest("GET", "/images", nil)
  if err != nil {
    t.Fatal(err)
  }


  handler := http.HandlerFunc(Images{Store: FakeImageStore{}}.List)
  handler.ServeHTTP(recorder, req)

  assert.Equal(t, http.StatusOK, recorder.Code)

  expected := `[{"id":1,"backed_up_at":"2016-01-01T12:33:44.567Z","ready":false}]
`
  assert.Equal(t, expected, string(recorder.Body.Bytes()))
}
