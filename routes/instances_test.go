package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gocardless/draupnir/models"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
)

func TestInstanceCreate(t *testing.T) {
	recorder := httptest.NewRecorder()

	request := createInstanceRequest{ImageID: "1"}
	body := bytes.NewBuffer([]byte{})
	jsonapi.MarshalOnePayload(body, &request)
	req := httptest.NewRequest("POST", "/instances", body)

	instanceStore := FakeInstanceStore{
		_Create: func(instance models.Instance) (models.Instance, error) {
			assert.Equal(t, 1, instance.ImageID)
			assert.True(t, instance.Port > 5432, "port is greater than 5432")
			assert.True(t, instance.Port < 6000, "port is less than 6000")
			return models.Instance{
				ID:        1,
				ImageID:   1,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
			}, nil
		},
	}

	imageStore := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			assert.Equal(t, 1, id)
			return models.Image{ID: 1, Ready: true}, nil
		},
	}

	executor := FakeExecutor{
		_CreateInstance: func(instanceID int, imageID int, port int) error {
			assert.Equal(t, 1, instanceID)
			assert.Equal(t, 1, imageID)
			return nil
		},
	}

	routeSet := Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
		Authenticator: AllowAll{},
		Logger:        log.NewNopLogger(),
	}
	routeSet.Create(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, createInstanceFixture, response)
}

func TestInstanceCreateReturnsErrorWithUnreadyImage(t *testing.T) {
	recorder := httptest.NewRecorder()

	request := createInstanceRequest{ImageID: "1"}
	body := bytes.NewBuffer([]byte{})
	jsonapi.MarshalOnePayload(body, &request)

	req := httptest.NewRequest("POST", "/instances", body)

	instanceStore := FakeInstanceStore{
		_Create: func(image models.Instance) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
			}, nil
		},
	}

	imageStore := FakeImageStore{
		_Get: func(id int) (models.Image, error) {
			return models.Image{ID: 1, Ready: false}, nil
		},
	}

	executor := FakeExecutor{
		_CreateInstance: func(instanceID int, imageID int, port int) error {
			return nil
		},
	}

	routeSet := Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
		Authenticator: AllowAll{},
		Logger:        log.NewNopLogger(),
	}
	routeSet.Create(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Equal(t, unreadyImageError, response)
}

func TestInstanceCreateReturnsErrorWithInvalidPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := map[string]string{"this is": "not a valid JSON API request payload"}
	body := bytes.NewBuffer([]byte{})
	json.NewEncoder(body).Encode(&request)
	req := httptest.NewRequest("POST", "/instances", body)
	logger, output := NewFakeLogger()

	Instances{Authenticator: AllowAll{}, Logger: logger}.Create(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, invalidJSONError, response)
	assert.Contains(t, output.String(), "not a jsonapi representation")
}

func TestInstanceCreateWithInvalidImageID(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := createInstanceRequest{ImageID: "garbage"}
	body := bytes.NewBuffer([]byte{})
	jsonapi.MarshalOnePayload(body, &request)
	logger, output := NewFakeLogger()

	req := httptest.NewRequest("POST", "/instances", body)

	routeSet := Instances{
		Executor:      FakeExecutor{},
		Authenticator: AllowAll{},
		Logger:        logger,
	}
	routeSet.Create(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, badImageIDError, response)
	assert.Contains(t, output.String(), "parsing \\\"garbage\\\": invalid syntax")
}

func TestInstanceList(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/instances", nil)

	store := FakeInstanceStore{
		_List: func() ([]models.Instance, error) {
			return []models.Instance{
				models.Instance{
					ID:        1,
					ImageID:   1,
					Port:      5432,
					CreatedAt: timestamp(),
					UpdatedAt: timestamp(),
					UserEmail: "test@draupnir",
				},
				models.Instance{
					ID:        2,
					ImageID:   1,
					Port:      5433,
					CreatedAt: timestamp(),
					UpdatedAt: timestamp(),
					UserEmail: "otheruser@draupnir",
				},
			}, nil
		},
	}

	routeSet := Instances{InstanceStore: store, Authenticator: AllowAll{}}
	routeSet.List(recorder, req)

	var response jsonapi.ManyPayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, listInstancesFixture, response)
}

func TestInstanceGet(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "test@draupnir",
			}, nil
		},
	}

	routeSet := Instances{InstanceStore: store, Authenticator: AllowAll{}}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", routeSet.Get)
	router.ServeHTTP(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, getInstanceFixture, response)
}

func TestInstanceGetFromWrongUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			assert.Equal(t, 1, id)

			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "otheruser@draupnir",
			}, nil
		},
	}

	// AllowAll will return a user email of test@draupnir
	routeSet := Instances{
		InstanceStore: store,
		Authenticator: AllowAll{},
	}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", routeSet.Get)
	router.ServeHTTP(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, notFoundError, response)
}

func TestInstanceDestroy(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "test@draupnir",
			}, nil
		},
		_Destroy: func(instance models.Instance) error {
			return nil
		},
	}

	executor := FakeExecutor{
		_DestroyInstance: func(instanceID int) error {
			return nil
		},
	}

	routeSet := Instances{InstanceStore: store, Executor: executor, Authenticator: AllowAll{}, Logger: log.NewNopLogger()}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", routeSet.Destroy).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
}

func TestInstanceDestroyFromWrongUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "otheruser@draupnir",
			}, nil
		},
		_Destroy: func(instance models.Instance) error {
			return nil
		},
	}

	executor := FakeExecutor{
		_DestroyInstance: func(instanceID int) error {
			return nil
		},
	}

	// AllowAll will return a user email of test@draupnir
	routeSet := Instances{InstanceStore: store, Executor: executor, Authenticator: AllowAll{}}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", routeSet.Destroy).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	var response APIError
	decodeJSON(recorder.Body, &response)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, notFoundError, response)
}
