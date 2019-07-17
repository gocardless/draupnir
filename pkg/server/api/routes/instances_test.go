package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gocardless/draupnir/pkg/models"
	"github.com/gocardless/draupnir/pkg/server/api"
	"github.com/gocardless/draupnir/pkg/server/api/auth"
	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/gocardless/draupnir/pkg/server/api/middleware"
	"github.com/google/jsonapi"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

var fakeCredentialsMap = map[string][]byte{
	"ca.crt":     []byte("-----BEGIN CERTIFICATE-----CA..."),
	"client.crt": []byte("-----BEGIN CERTIFICATE-----client..."),
	"client.key": []byte("-----BEGIN PRIVATE KEY-----client..."),
}

func TestInstanceCreate(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	request := CreateInstanceRequest{ImageID: "1"}
	jsonapi.MarshalOnePayload(body, &request)
	req, recorder, _ := createRequest(t, "POST", "/instances", body)

	instanceStore := FakeInstanceStore{
		_Create: func(instance models.Instance) (models.Instance, error) {
			assert.Equal(t, 1, instance.ImageID)
			assert.True(t, instance.Port > 5432, "port is greater than 5432")
			assert.True(t, instance.Port < 6000, "port is less than 6000")
			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
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
		_CreateInstance: func(ctx context.Context, instanceID int, imageID int, port int) error {
			assert.Equal(t, 1, instanceID)
			assert.Equal(t, 1, imageID)
			return nil
		},
		_RetrieveInstanceCredentials: func(ctx context.Context, id int) (map[string][]byte, error) {
			assert.Equal(t, 1, id)
			return fakeCredentialsMap, nil
		},
	}

	routeSet := Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
	}
	err := routeSet.Create(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, createInstanceFixture, response)
	assert.Nil(t, err)
}

func TestInstanceCreateReturnsErrorWithUnreadyImage(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	request := CreateInstanceRequest{ImageID: "1"}
	jsonapi.MarshalOnePayload(body, &request)
	req, recorder, _ := createRequest(t, "POST", "/instances", body)

	instanceStore := FakeInstanceStore{
		_Create: func(image models.Instance) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
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
		_CreateInstance: func(ctx context.Context, instanceID int, imageID int, port int) error {
			return nil
		},
	}

	routeSet := Instances{
		InstanceStore: instanceStore,
		ImageStore:    imageStore,
		Executor:      executor,
	}
	err := routeSet.Create(recorder, req)

	var response api.Error
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	assert.Equal(t, api.UnreadyImageError, response)
	assert.Nil(t, err)
}

func TestInstanceCreateReturnsErrorWithInvalidPayload(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	request := map[string]string{"this is": "not a valid JSON API request payload"}
	json.NewEncoder(body).Encode(&request)
	req, recorder, logs := createRequest(t, "POST", "/instances", body)

	err := Instances{}.Create(recorder, req)

	var response api.Error
	decodeJSON(t, recorder.Body, &response)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, api.InvalidJSONError, response)
	assert.Contains(t, logs.String(), "not a jsonapi representation")
}

func TestInstanceCreateWithInvalidImageID(t *testing.T) {
	body := bytes.NewBuffer([]byte{})
	request := CreateInstanceRequest{ImageID: "garbage"}
	jsonapi.MarshalOnePayload(body, &request)
	req, recorder, logs := createRequest(t, "POST", "/instances", body)

	routeSet := Instances{
		Executor: FakeExecutor{},
	}
	err := routeSet.Create(recorder, req)

	var response api.Error
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, api.BadImageIDError, response)
	assert.Contains(t, logs.String(), "parsing \\\"garbage\\\": invalid syntax")
	assert.Nil(t, err)
}

func TestInstanceList(t *testing.T) {
	req, recorder, _ := createRequest(t, "GET", "/instances", nil)

	store := FakeInstanceStore{
		_List: func() ([]models.Instance, error) {
			return []models.Instance{
				models.Instance{
					ID:        1,
					Hostname:  "draupnir-server.example.com",
					ImageID:   1,
					Port:      5432,
					CreatedAt: timestamp(),
					UpdatedAt: timestamp(),
					UserEmail: "test@draupnir",
				},
				models.Instance{
					ID:        2,
					Hostname:  "draupnir-server.example.com",
					ImageID:   1,
					Port:      5433,
					CreatedAt: timestamp(),
					UpdatedAt: timestamp(),
					UserEmail: "otheruser@draupnir",
				},
			}, nil
		},
	}

	routeSet := Instances{InstanceStore: store}
	err := routeSet.List(recorder, req)

	var response jsonapi.ManyPayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, listInstancesFixture, response)
	assert.Nil(t, err)
}

func TestInstanceGet(t *testing.T) {
	req, recorder, _ := createRequest(t, "GET", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "test@draupnir",
			}, nil
		},
	}

	executor := FakeExecutor{
		_RetrieveInstanceCredentials: func(ctx context.Context, id int) (map[string][]byte, error) {
			assert.Equal(t, 1, id)
			return fakeCredentialsMap, nil
		},
	}

	errorHandler := FakeErrorHandler{}
	routeSet := Instances{
		InstanceStore: store,
		Executor:      executor,
	}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", errorHandler.Handle(routeSet.Get))
	router.ServeHTTP(recorder, req)

	var response jsonapi.OnePayload
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, getInstanceFixture, response)
	assert.Nil(t, errorHandler.Error)
}

func TestInstanceGetFromWrongUser(t *testing.T) {
	req, recorder, _ := createRequest(t, "GET", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			assert.Equal(t, 1, id)

			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "otheruser@draupnir",
			}, nil
		},
	}

	routeSet := Instances{
		InstanceStore: store,
	}

	errorHandler := FakeErrorHandler{}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", errorHandler.Handle(routeSet.Get))
	router.ServeHTTP(recorder, req)

	var response api.Error
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, api.NotFoundError, response)
	assert.Nil(t, errorHandler.Error)
}

func TestInstanceDestroy(t *testing.T) {
	req, recorder, _ := createRequest(t, "DELETE", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
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
		_DestroyInstance: func(ctx context.Context, instanceID int) error {
			return nil
		},
	}

	routeSet := Instances{InstanceStore: store, Executor: executor}

	errorHandler := FakeErrorHandler{}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", errorHandler.Handle(routeSet.Destroy)).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
	assert.Nil(t, errorHandler.Error)
}

func TestInstanceDestroyFromWrongUser(t *testing.T) {
	req, recorder, _ := createRequest(t, "DELETE", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
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
		_DestroyInstance: func(ctx context.Context, instanceID int) error {
			return nil
		},
	}

	routeSet := Instances{InstanceStore: store, Executor: executor}

	errorHandler := FakeErrorHandler{}
	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", errorHandler.Handle(routeSet.Destroy)).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	var response api.Error
	decodeJSON(t, recorder.Body, &response)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, api.NotFoundError, response)
	assert.Nil(t, errorHandler.Error)
}

func TestInstanceDestroyFromUploadUser(t *testing.T) {
	req, recorder, _ := createRequest(t, "DELETE", "/instances/1", nil)

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				Hostname:  "draupnir-server.example.com",
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
		_DestroyInstance: func(ctx context.Context, instanceID int) error {
			return nil
		},
	}

	authenticator := auth.FakeAuthenticator{
		MockAuthenticateRequest: func(r *http.Request) (string, string, error) {
			return auth.UPLOAD_USER_EMAIL, "", nil
		},
	}

	errorHandler := FakeErrorHandler{}
	routeSet := Instances{InstanceStore: store, Executor: executor}
	router := mux.NewRouter()
	route := chain.New(errorHandler.Handle).
		Add(middleware.Authenticate(authenticator)).
		Resolve(routeSet.Destroy)
	router.HandleFunc("/instances/{id}", route).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
	assert.Nil(t, errorHandler.Error)
}
