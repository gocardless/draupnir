package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocardless/draupnir/models"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type FakeImageStore struct {
	_List        func() ([]models.Image, error)
	_Get         func(int) (models.Image, error)
	_Create      func(models.Image) (models.Image, error)
	_Destroy     func(models.Image) error
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

func (s FakeImageStore) Destroy(image models.Image) error {
	return s._Destroy(image)
}

func (s FakeImageStore) MarkAsReady(image models.Image) (models.Image, error) {
	return s._MarkAsReady(image)
}

type FakeExecutor struct {
	_CreateBtrfsSubvolume func(id int) error
	_FinaliseImage        func(image models.Image) error
	_CreateInstance       func(imageID int, instanceID int, port int) error
	_DestroyImage         func(id int) error
	_DestroyInstance      func(id int) error
}

func (e FakeExecutor) CreateBtrfsSubvolume(id int) error {
	return e._CreateBtrfsSubvolume(id)
}

func (e FakeExecutor) FinaliseImage(image models.Image) error {
	return e._FinaliseImage(image)
}

func (e FakeExecutor) CreateInstance(imageID int, instanceID int, port int) error {
	return e._CreateInstance(imageID, instanceID, port)
}

func (e FakeExecutor) DestroyImage(id int) error {
	return e._DestroyImage(id)
}

func (e FakeExecutor) DestroyInstance(id int) error {
	return e._DestroyInstance(id)
}

func TestGetImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/images/1", nil)

	store := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			return models.Image{
				ID:         1,
				BackedUpAt: timestamp(),
				Ready:      false,
				CreatedAt:  timestamp(),
				UpdatedAt:  timestamp(),
			}, nil
		},
	}

	routeSet := Images{Store: store}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}", routeSet.Get)
	router.ServeHTTP(recorder, req)

	expected, err := json.Marshal(getImageFixture)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
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

	expected, err := json.Marshal(listImagesFixture)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestCreateImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data":{"type":"images","attributes":{"backed_up_at": "2016-01-01T12:33:44.567Z","anonymisation_script":"SELECT * FROM foo;"}}}`
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

	expected, err := json.Marshal(createImageFixture)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestImageCreateReturnsErrorWithInvalidPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"this is": "not a valid JSON API request payload"}`
	req, err := http.NewRequest("POST", "/images", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	routeSet := Images{}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	expected, err := json.Marshal(invalidJSONError)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestImageCreateReturnsErrorWhenSubvolumeCreationFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data": { "type": "images", "attributes": { "backed_up_at": "2016-01-01T12:33:44.567Z"} } }`
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
	expected, err := json.Marshal(internalServerError)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestImageDone(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/images/1/done", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			return models.Image{
				ID:         1,
				BackedUpAt: timestamp(),
				Ready:      false,
				CreatedAt:  timestamp(),
				UpdatedAt:  timestamp(),
			}, nil
		},
		_MarkAsReady: func(image models.Image) (models.Image, error) {
			image.Ready = true
			return image, nil
		},
	}

	executor := FakeExecutor{
		_FinaliseImage: func(image models.Image) error {
			return nil
		},
	}

	routeSet := Images{Store: store, Executor: executor}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}/done", routeSet.Done)
	router.ServeHTTP(recorder, req)

	expected, err := json.Marshal(doneImageFixture)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestImageDoneWithNonNumericID(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/images/bad_id/done", nil)
	if err != nil {
		t.Fatal(err)
	}

	routeSet := Images{Store: nil, Executor: nil}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}/done", routeSet.Done)
	router.ServeHTTP(recorder, req)

	expected, err := json.Marshal(notFoundError)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestImageDestroy(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/images/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			return models.Image{
				ID:         1,
				BackedUpAt: timestamp(),
				Ready:      false,
				CreatedAt:  timestamp(),
				UpdatedAt:  timestamp(),
			}, nil
		},
		_Destroy: func(image models.Image) error {
			return nil
		},
	}

	executor := FakeExecutor{
		_DestroyImage: func(imageID int) error {
			return nil
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/images/{id}", Images{Store: store, Executor: executor}.Destroy).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
}

func timestamp() time.Time {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	return time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
}
