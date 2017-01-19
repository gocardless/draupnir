package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocardless/draupnir/exec"
	"github.com/gocardless/draupnir/models"
	"github.com/stretchr/testify/assert"
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

func (s FakeImageStore) Get(id int) (models.Image, error) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	timestamp := time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
	return models.Image{ID: 1, BackedUpAt: timestamp, Ready: false, CreatedAt: timestamp, UpdatedAt: timestamp}, nil
}

func (s FakeImageStore) Create(_ models.Image) (models.Image, error) {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	timestamp := time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
	return models.Image{ID: 1, BackedUpAt: timestamp, Ready: false, CreatedAt: timestamp, UpdatedAt: timestamp}, nil
}

func (s FakeImageStore) MarkAsReady(image models.Image) (models.Image, error) {
	image.Ready = true
	return image, nil
}

type FakeExecutor struct {
	exec.Executor
	_CreateBtrfsSubvolume func(id int) error
}

func (e FakeExecutor) CreateBtrfsSubvolume(id int) error {
	return e._CreateBtrfsSubvolume(id)
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

	executor := FakeExecutor{
		_CreateBtrfsSubvolume: func(id int) error {
			return nil
		},
	}
	routeSet := Images{Store: FakeImageStore{}, Executor: executor}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	expected := `{"id":1,"backed_up_at":"2016-01-01T12:33:44.567Z","ready":false,"created_at":"2016-01-01T12:33:44.567Z","updated_at":"2016-01-01T12:33:44.567Z"}
`
	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, expected, string(recorder.Body.Bytes()))
}

func TestCreateReturnsErrorWhenSubvolumeCreationFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"backed_up_at": "2016-01-01T12:33:44.567Z"}`
	req, err := http.NewRequest("POST", "/images", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	executor := FakeExecutor{
		_CreateBtrfsSubvolume: func(id int) error {
			return errors.New("some btrfs error")
		},
	}
	routeSet := Images{Store: FakeImageStore{}, Executor: executor}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	expected := "error creating btrfs subvolume: some btrfs error\n"
	assert.Equal(t, expected, string(recorder.Body.Bytes()))
}
