package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gocardless/draupnir/auth"
	"github.com/gocardless/draupnir/models"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func decodeJSON(r io.Reader, out interface{}) {
	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		log.Panic(err)
	}
}

func TestGetImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/images/1", nil)

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

	var response jsonapi.OnePayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, response, getImageFixture)
}

func TestGetImageWhenAuthenticationFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/images/1", nil)

	authenticator := FakeAuthenticator{
		_AuthenticateRequest: func(r *http.Request) (string, error) {
			return "", errors.New("Invalid email address")
		},
	}

	handler := http.HandlerFunc(Images{Authenticator: authenticator}.Get)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestListImages(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/images", nil)

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

	handler := Images{ImageStore: store, Authenticator: AllowAll{}}.List
	handler(recorder, req)

	var response jsonapi.ManyPayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, response, listImagesFixture)
}

func TestCreateImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := createImageRequest{
		BackedUpAt: timestamp(),
		Anon:       "SELECT * FROM foo;",
	}
	body := bytes.NewBuffer([]byte{})
	jsonapi.MarshalOnePayload(body, &request)

	req := httptest.NewRequest("POST", "/images", body)

	executor := FakeExecutor{
		_CreateBtrfsSubvolume: func(id int) error { assert.Equal(t, id, 1); return nil },
	}

	store := FakeImageStore{
		_Create: func(image models.Image) (models.Image, error) {
			assert.Equal(t, image.Anon, "SELECT * FROM foo;")
			return models.Image{
				ID:         1,
				BackedUpAt: image.BackedUpAt,
				Ready:      false,
				CreatedAt:  timestamp(),
				UpdatedAt:  timestamp(),
			}, nil
		},
	}

	routeSet := Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}
	routeSet.Create(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, createImageFixture, response)
}

func TestImageCreateReturnsErrorWithInvalidPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"this is": "not a valid JSON API request payload"}`
	req := httptest.NewRequest("POST", "/images", strings.NewReader(body))

	routeSet := Images{Authenticator: AllowAll{}}
	routeSet.Create(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, invalidJSONError, response)
}

func TestImageCreateReturnsErrorWhenSubvolumeCreationFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := createImageRequest{
		BackedUpAt: timestamp(),
		Anon:       "SELECT * FROM foo;",
	}
	body := bytes.NewBuffer([]byte{})
	jsonapi.MarshalOnePayload(body, &request)
	req := httptest.NewRequest("POST", "/images", body)

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
	routeSet.Create(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Equal(t, internalServerError, response)
}

func TestImageDone(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/images/1/done", nil)

	image := models.Image{
		ID:         1,
		BackedUpAt: timestamp(),
		Ready:      false,
		CreatedAt:  timestamp(),
		UpdatedAt:  timestamp(),
	}

	store := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			assert.Equal(t, 1, id)

			return image, nil
		},
		_MarkAsReady: func(i models.Image) (models.Image, error) {
			assert.Equal(t, image, i)

			i.Ready = true
			return i, nil
		},
	}

	executor := FakeExecutor{
		_FinaliseImage: func(i models.Image) error {
			assert.Equal(t, image, i)

			return nil
		},
	}

	routeSet := Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}/done", routeSet.Done)
	router.ServeHTTP(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, doneImageFixture, response)
}

func TestImageDoneWithNonNumericID(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/images/bad_id/done", nil)

	routeSet := Images{Authenticator: AllowAll{}}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}/done", routeSet.Done)
	router.ServeHTTP(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, notFoundError, response)
}

func TestImageDestroy(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/images/1", nil)

	image := models.Image{
		ID:         1,
		BackedUpAt: timestamp(),
		Ready:      false,
		CreatedAt:  timestamp(),
		UpdatedAt:  timestamp(),
	}

	store := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			assert.Equal(t, 1, id)

			return image, nil
		},
		_Destroy: func(i models.Image) error {
			assert.Equal(t, image, i)
			return nil
		},
	}

	executor := FakeExecutor{
		_DestroyImage: func(imageID int) error {
			assert.Equal(t, 1, imageID)
			return nil
		},
	}

	router := mux.NewRouter()
	routeSet := Images{ImageStore: store, Executor: executor, Authenticator: AllowAll{}}
	router.HandleFunc("/images/{id}", routeSet.Destroy).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
}

func TestImageDestroyFromUploadUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/images/1", nil)

	image := models.Image{
		ID:         1,
		BackedUpAt: timestamp(),
		Ready:      false,
		CreatedAt:  timestamp(),
		UpdatedAt:  timestamp(),
	}

	imageStore := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			assert.Equal(t, 1, id)
			return image, nil
		},
		_Destroy: func(i models.Image) error {
			assert.Equal(t, image, i)
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
			assert.Equal(t, 1, imageID)
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
	routeSet := Images{ImageStore: imageStore, InstanceStore: instanceStore, Executor: executor, Authenticator: authenticator}
	router.HandleFunc("/images/{id}", routeSet.Destroy).Methods("DELETE")
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
