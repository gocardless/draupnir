package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/server/api/routes/auth"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func decodeJSON(t *testing.T, r io.Reader, out interface{}) {
	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		t.Fatalf("%s", errors.Wrap(err, "Could not decode JSON").Error())
	}
}

func TestGetImage(t *testing.T) {
	req, recorder, _ := createRequest(t, "GET", "/images/1", nil)

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

	errorHandler := FakeErrorHandler{}
	routeSet := Images{ImageStore: store}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}", errorHandler.Handle(routeSet.Get))
	router.ServeHTTP(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, getImageFixture, response)
	assert.Nil(t, errorHandler.Error)
}

func TestListImages(t *testing.T) {
	req, recorder, _ := createRequest(t, "GET", "/images", nil)

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

	handler := Images{ImageStore: store}.List
	err := handler(recorder, req)

	var response jsonapi.ManyPayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, response, listImagesFixture)
	assert.Nil(t, err)
}

func TestCreateImage(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	request := CreateImageRequest{
		BackedUpAt: timestamp(),
		Anon:       "SELECT * FROM foo;",
	}
	jsonapi.MarshalOnePayload(body, &request)
	req, recorder, _ := createRequest(t, "POST", "/images", body)

	executor := FakeExecutor{
		_CreateBtrfsSubvolume: func(ctx context.Context, id int) error { assert.Equal(t, id, 1); return nil },
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

	routeSet := Images{ImageStore: store, Executor: executor}
	err := routeSet.Create(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, createImageFixture, response)
	assert.Nil(t, err)
}

func TestImageCreateReturnsErrorWithInvalidPayload(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	payload := map[string]string{"this is": "not a valid JSON API request payload"}
	json.NewEncoder(body).Encode(&payload)
	req, recorder, logs := createRequest(t, "POST", "/images", body)

	err := Images{}.Create(recorder, req)

	var response APIError
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, invalidJSONError, response)
	assert.Contains(t, logs.String(), "data is not a jsonapi representation")
	assert.Nil(t, err)
}

func TestImageCreateReturnsErrorWhenSubvolumeCreationFails(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	request := CreateImageRequest{
		BackedUpAt: timestamp(),
		Anon:       "SELECT * FROM foo;",
	}
	jsonapi.MarshalOnePayload(body, &request)
	req, recorder, logs := createRequest(t, "POST", "/images", body)

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
		_CreateBtrfsSubvolume: func(context.Context, int) error {
			return errors.New("some btrfs error")
		},
	}

	routeSet := Images{
		ImageStore: store,
		Executor:   executor,
	}
	err := routeSet.Create(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Body.String())
	assert.Empty(t, logs.String())
	assert.Equal(t, "failed to create btrfs subvolume: some btrfs error", err.Error())
}

func TestImageDone(t *testing.T) {
	req, recorder, _ := createRequest(t, "POST", "/images/1/done", nil)

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
		_FinaliseImage: func(ctx context.Context, i models.Image) error {
			assert.Equal(t, image, i)

			return nil
		},
	}

	errorHandler := FakeErrorHandler{}
	routeSet := Images{ImageStore: store, Executor: executor}
	router := mux.NewRouter()
	router.HandleFunc("/images/{id}/done", errorHandler.Handle(routeSet.Done))
	router.ServeHTTP(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, doneImageFixture, response)
	assert.Nil(t, errorHandler.Error)
}

func TestImageDoneWithNonNumericID(t *testing.T) {
	req, recorder, logs := createRequest(t, "POST", "/images/bad_id/done", nil)

	errorHandler := FakeErrorHandler{}

	router := mux.NewRouter()
	router.HandleFunc("/images/{id}/done", errorHandler.Handle(Images{}.Done))
	router.ServeHTTP(recorder, req)

	var response APIError
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, notFoundError, response)
	assert.Contains(t, logs.String(), "invalid syntax")
	assert.Nil(t, errorHandler.Error)
}

func TestImageDestroy(t *testing.T) {
	req, recorder, logs := createRequest(t, "DELETE", "/images/1", nil)

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
		_DestroyImage: func(ctx context.Context, imageID int) error {
			assert.Equal(t, 1, imageID)
			return nil
		},
	}

	errorHandler := FakeErrorHandler{}

	router := mux.NewRouter()
	routeSet := Images{ImageStore: store, Executor: executor}
	router.HandleFunc("/images/{id}", errorHandler.Handle(routeSet.Destroy)).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
	assert.Contains(t, logs.String(), "destroying image")
	assert.Nil(t, errorHandler.Error)
}

func TestImageDestroyFromUploadUser(t *testing.T) {
	req, recorder, logs := createRequest(t, "DELETE", "/images/1", nil)
	req = req.WithContext(
		context.WithValue(req.Context(), authUserKey, auth.UPLOAD_USER_EMAIL),
	)

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
		_DestroyImage: func(ctx context.Context, imageID int) error {
			assert.Equal(t, 1, imageID)
			return nil
		},
		_DestroyInstance: func(context.Context, int) error {
			return nil
		},
	}

	errorHandler := FakeErrorHandler{}

	router := mux.NewRouter()
	routeSet := Images{
		ImageStore:    imageStore,
		InstanceStore: instanceStore,
		Executor:      executor,
	}
	router.HandleFunc("/images/{id}", errorHandler.Handle(routeSet.Destroy)).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
	assert.Equal(t, []int{1, 3}, destroyedImages)
	assert.Contains(t, logs.String(), "destroying instance")
	assert.Contains(t, logs.String(), "destroying image")
	assert.Nil(t, errorHandler.Error)
}

func timestamp() time.Time {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err.Error())
	}
	return time.Date(2016, 1, 1, 12, 33, 44, 567000000, loc)
}
