package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gocardless/draupnir/models"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestInstanceCreate(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data":{"type":"instances","attributes":{"image_id":"1"}}}`

	req, err := http.NewRequest("POST", "/instances", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

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
			return models.Image{ID: 1, Ready: true}, nil
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
	}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	expected, err := json.Marshal(createInstanceFixture)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceCreateReturnsErrorWithUnreadyImage(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data":{"type":"instances","attributes":{"image_id":"1"}}}`

	req, err := http.NewRequest("POST", "/instances", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

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
	}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	expected, err := json.Marshal(unreadyImageError)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceCreateReturnsErrorWithInvalidPayload(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"this is": "not a valid JSON API request payload"}`
	req, err := http.NewRequest("POST", "/instances", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(Instances{Authenticator: AllowAll{}}.Create)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	expected, err := json.Marshal(invalidJSONError)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceCreateWithInvalidImageID(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data":{"type":"instances","attributes":{"image_id":"garbage"}}}`

	req, err := http.NewRequest("POST", "/instances", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	routeSet := Instances{Executor: FakeExecutor{}, Authenticator: AllowAll{}}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	expected, err := json.Marshal(badImageIDError)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceList(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/instances", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeInstanceStore{
		_List: func() ([]models.Instance, error) {
			return []models.Instance{
				models.Instance{
					ID:        1,
					ImageID:   1,
					Port:      5432,
					CreatedAt: timestamp(),
					UpdatedAt: timestamp(),
					UserEmail: "hmac@gocardless.com",
				},
				models.Instance{
					ID:        2,
					ImageID:   1,
					Port:      5433,
					CreatedAt: timestamp(),
					UpdatedAt: timestamp(),
					UserEmail: "alice@gocardless.com",
				},
			}, nil
		},
	}

	handler := http.HandlerFunc(Instances{InstanceStore: store, Authenticator: AllowAll{}}.List)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	expected, err := json.Marshal(listInstancesFixture)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceGet(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/instances/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "hmac@gocardless.com",
			}, nil
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", Instances{InstanceStore: store, Authenticator: AllowAll{}}.Get)
	router.ServeHTTP(recorder, req)

	expected, err := json.Marshal(getInstanceFixture)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceGetFromWrongUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/instances/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "alice@gocardless.com",
			}, nil
		},
	}

	router := mux.NewRouter()
	router.HandleFunc("/instances/{id}", Instances{InstanceStore: store, Authenticator: AllowAll{}}.Get)
	router.ServeHTTP(recorder, req)

	expected, err := json.Marshal(notFoundError)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceDestroy(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/instances/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "hmac@gocardless.com",
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

	router := mux.NewRouter()
	router.HandleFunc(
		"/instances/{id}",
		Instances{InstanceStore: store, Executor: executor, Authenticator: AllowAll{}}.Destroy,
	).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
}

func TestInstanceDestroyFromWrongUser(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/instances/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeInstanceStore{
		_Get: func(id int) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				Port:      5432,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
				UserEmail: "alice@gocardless.com",
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

	router := mux.NewRouter()
	// AllowAll will return a user email of hmac@gocardless.com
	router.HandleFunc(
		"/instances/{id}",
		Instances{InstanceStore: store, Executor: executor, Authenticator: AllowAll{}}.Destroy,
	).Methods("DELETE")
	router.ServeHTTP(recorder, req)

	expected, err := json.Marshal(notFoundError)
	if err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}
