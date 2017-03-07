package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gocardless/draupnir/models"
	"github.com/stretchr/testify/assert"
)

type FakeInstanceStore struct {
	_Create func(models.Instance) (models.Instance, error)
}

func (s FakeInstanceStore) Create(image models.Instance) (models.Instance, error) {
	return s._Create(image)
}

func TestCreateInstance(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data":{"type":"instances","attributes":{"image_id":"1"}}}`

	req, err := http.NewRequest("POST", "/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", mediaType)
	if err != nil {
		t.Fatal(err)
	}

	store := FakeInstanceStore{
		_Create: func(image models.Instance) (models.Instance, error) {
			return models.Instance{
				ID:        1,
				ImageID:   1,
				CreatedAt: timestamp(),
				UpdatedAt: timestamp(),
			}, nil
		},
	}

	executor := FakeExecutor{
		_CreateInstance: func(instanceID int, imageID int, port int) error {
			return nil
		},
	}
	routeSet := Instances{Store: store, Executor: executor}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	expected, err := json.Marshal(createInstanceFixture)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}

func TestInstanceCreateWithInvalidImageID(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := `{"data":{"type":"instances","attributes":{"image_id":"garbage"}}}`

	req, err := http.NewRequest("POST", "/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", mediaType)
	if err != nil {
		t.Fatal(err)
	}

	routeSet := Instances{Executor: FakeExecutor{}}
	handler := http.HandlerFunc(routeSet.Create)
	handler.ServeHTTP(recorder, req)

	expected, err := json.Marshal(badImageIDError)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, append(expected, byte('\n')), recorder.Body.Bytes())
}
