package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocardless/draupnir/auth"
	"github.com/gocardless/draupnir/models"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

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

	routeSet := Images{ImageStore: store, Authenticator: AllowAll{}}
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

func TestGetImageWhenAuthenticationFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/images/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	authenticator := FakeAuthenticator{
		_AuthenticateRequest: func(r *http.Request) (string, error) {
			return "", errors.New("Invalid email address")
		},
	}

	handler := http.HandlerFunc(Images{Authenticator: authenticator}.Get)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
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

	handler := http.HandlerFunc(Images{ImageStore: store, Authenticator: AllowAll{}}.List)
	handler.ServeHTTP(recorder, req)

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

	routeSet := Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}
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

	routeSet := Images{Authenticator: AllowAll{}}
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
	routeSet := Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}
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

	routeSet := Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}
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

	routeSet := Images{Authenticator: AllowAll{}}
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
	router.HandleFunc("/images/{id}", Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}.Destroy).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
}

func TestImageDestroyFromUploadUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/images/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	imageStore := FakeImageStore{
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

	destroyedImages := make([]int, 0)

	instanceStore := FakeInstanceStore{
		_List: func() ([]models.Instance, error) {
			return []models.Instance{
				models.Instance{ID: 1, ImageID: 1},
				models.Instance{ID: 2, ImageID: 2},
				models.Instance{ID: 3, ImageID: 1},
			}, nil
		},
		_Destroy: func(instance models.Instance) error {
			destroyedImages = append(destroyedImages, instance.ID)
			return nil
		},
	}

	executor := FakeExecutor{
		_DestroyImage: func(imageID int) error {
			return nil
		},
		_DestroyInstance: func(id int) error {
			return nil
		},
	}

	authenticator := FakeAuthenticator{
		_AuthenticateRequest: func(r *http.Request) (string, error) {
			return auth.UPLOAD_USER_EMAIL, nil
		},
	}

	router := mux.NewRouter()
	router.HandleFunc(
		"/images/{id}",
		Images{ImageStore: imageStore, InstanceStore: instanceStore, Executor: executor, Authenticator: authenticator}.Destroy,
	).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
	assert.Equal(t, []int{1, 3}, destroyedImages)
}

func timestamp() time.Time {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	return time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
}
