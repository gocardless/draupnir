package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gocardless/draupnir/version"
	"github.com/stretchr/testify/assert"
)

func TestCheckApiVersionWithNoVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)

	assert.Equal(t, CheckAPIVersion(recorder, req), false)
	assert.Equal(t, recorder.Code, http.StatusBadRequest)

	var response APIError
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, response, missingApiVersion)
}

func TestCheckApiVersionWithMismatchingVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header["Draupnir-Version"] = []string{"0.0.0"}

	assert.Equal(t, CheckAPIVersion(recorder, req), false)
	assert.Equal(t, recorder.Code, http.StatusBadRequest)

	var response APIError
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, response, invalidApiVersion("0.0.0"))
}

func TestCheckApiVersionWithMatchingVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header["Draupnir-Version"] = []string{version.Version}

	assert.Equal(t, CheckAPIVersion(recorder, req), true)
}
