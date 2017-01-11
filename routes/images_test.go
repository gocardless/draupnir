package routes

import (
	"github.com/gocardless/draupnir/models"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type FakeImageStore struct{}

func (s FakeImageStore) List() ([]models.Image, error) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	timestamp := time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
	return []models.Image{
		models.Image{ID: 1, BackedUpAt: timestamp, Ready: false, CreatedAt: timestamp, UpdatedAt: timestamp},
	}, nil
}

func (s FakeImageStore) Create(models.Image) (models.Image, error) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	timestamp := time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
	return models.Image{ID: 1, BackedUpAt: timestamp, Ready: false, CreatedAt: timestamp, UpdatedAt: timestamp}, nil
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

	expected := `[{"id":1,"backed_up_at":"2016-01-01T12:33:44.567Z","ready":false,"created_at":"2016-01-01T12:33:44.567Z","updated_at":"2016-01-01T12:33:44.567Z"}]
`
	assert.Equal(t, expected, string(recorder.Body.Bytes()))
}

func TestCreateImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"backed_up_at": "2016-01-01T12:33:44.567Z"}`
	req, err := http.NewRequest("POST", "/images", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(Images{Store: FakeImageStore{}}.Create)
	handler.ServeHTTP(recorder, req)

	expected := `{"id":1,"backed_up_at":"2016-01-01T12:33:44.567Z","ready":false,"created_at":"2016-01-01T12:33:44.567Z","updated_at":"2016-01-01T12:33:44.567Z"}
`
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, expected, string(recorder.Body.Bytes()))
}
