package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocardless/draupnir/models"
	"github.com/stretchr/testify/assert"
)

type FakeImageStore struct {
	_List        func() ([]models.Image, error)
	_Get         func(int) (models.Image, error)
	_Create      func(models.Image) (models.Image, error)
	_MarkAsReady func(models.Image) (models.Image, error)
}

func (s FakeImageStore) List() ([]models.Image, error) {
	return s._List()
}

func (s FakeImageStore) Get(id int) (models.Image, error) {
	return s._Get(id)
}

func (s FakeImageStore) Create(image models.Image) (models.Image, error) {
	return s._Create(image)
}

func (s FakeImageStore) MarkAsReady(image models.Image) (models.Image, error) {
	return s._MarkAsReady(image)
}

type FakeExecutor struct {
	_CreateBtrfsSubvolume func(id int) error
	_FinaliseImage        func(id int) error
}

func (e FakeExecutor) CreateBtrfsSubvolume(id int) error {
	return e._CreateBtrfsSubvolume(id)
}

func (e FakeExecutor) FinaliseImage(id int) error {
	return e._FinaliseImage(id)
}

func TestListImages(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/images", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeImageStore{
		_List: func() ([]models.Image, error) {
			return []models.Image{
				models.Image{
					ID:         1,
					BackedUpAt: timestamp(),
					Ready:      false,
					CreatedAt:  timestamp(),
					UpdatedAt:  timestamp(),
				},
			}, nil
		},
	}

	handler := http.HandlerFunc(Images{Store: store}.List)
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

	store := FakeImageStore{
		_Create: func(image models.Image) (models.Image, error) {
			return models.Image{
				ID:         1,
				BackedUpAt: timestamp(),
				Ready:      false,
				CreatedAt:  timestamp(),
				UpdatedAt:  timestamp(),
			}, nil
		},
	}

	routeSet := Images{Store: store, Executor: executor}
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

	store := FakeImageStore{
		_Create: func(image models.Image) (models.Image, error) {
			return models.Image{
				ID:         1,
				BackedUpAt: timestamp(),
				Ready:      false,
				CreatedAt:  timestamp(),
				UpdatedAt:  timestamp(),
			}, nil
		},
	}

	executor := FakeExecutor{
		_CreateBtrfsSubvolume: func(id int) error {
			return errors.New("some btrfs error")
		},
	}
	routeSet := Images{Store: store, Executor: executor}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	expected := "error creating btrfs subvolume: some btrfs error\n"
	assert.Equal(t, expected, string(recorder.Body.Bytes()))
}

func timestamp() time.Time {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	return time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
}
