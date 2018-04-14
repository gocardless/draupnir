package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gocardless/draupnir/server/api/auth"
	apiErrors "github.com/gocardless/draupnir/server/api/errors"
	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
)

type FailureAuthenticator struct{}

func (f FailureAuthenticator) AuthenticateRequest(r *http.Request) (string, error) {
	return "", errors.New("could not authenticate")
}

func TestAuthenticateSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	authenticator := auth.FakeAuthenticator{
		MockAuthenticateRequest: func(r *http.Request) (string, error) {
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

	authenticator := auth.FakeAuthenticator{
		MockAuthenticateRequest: func(r *http.Request) (string, error) {
			return "", errors.New("could not authenticate")
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) error {
		t.Fatal("this route should never be called")
		return nil
	}

	NewRequestLogger(logger)(Authenticate(authenticator)(handler))(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)

	var response apiErrors.Error
	err := json.NewDecoder(recorder.Body).Decode(&response)

	assert.Nil(t, err, "failed to decode response into APIError")
	assert.EqualValues(t, apiErrors.UnauthorizedError, response)
}
