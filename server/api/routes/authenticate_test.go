package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticateSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	authenticator := FakeAuthenticator{
		_AuthenticateRequest: func(r *http.Request) (string, error) {
			return "some_user@domain.org", nil
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	Authenticate(authenticator)(handler)(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAuthenticateFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	logger := log.NewNopLogger()

	authenticator := FakeAuthenticator{
		_AuthenticateRequest: func(r *http.Request) (string, error) {
			return "", errors.New("could not authenticate")
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) error {
		t.Fatal("this route should never be called")
		return nil
	}

	NewRequestLogger(logger)(Authenticate(authenticator)(handler))(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	var response APIError
	err := json.NewDecoder(recorder.Body).Decode(&response)

	assert.Nil(t, err, "failed to decode response into APIError")
	assert.EqualValues(t, unauthorizedError, response)
}
