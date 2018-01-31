package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckApiVersionWithNoVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)

	CheckAPIVersion("1.0.0")(
		func(w http.ResponseWriter, h *http.Request) {
			t.Fatal("this route should never be called")
		},
	)(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusBadRequest)

	var response APIError
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, response, missingApiVersion)
}

func TestCheckApiVersionWithHigherVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header["Draupnir-Version"] = []string{"0.0.0"}

	CheckAPIVersion("1.0.0")(
		func(w http.ResponseWriter, h *http.Request) {
			t.Fatal("this route should never be called")
		},
	)(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusBadRequest)

	var response APIError
	err := json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, response, invalidApiVersion("0.0.0"))
}

func TestCheckApiVersionWithLowerMinorVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header["Draupnir-Version"] = []string{"1.0.0"}

	CheckAPIVersion("1.1.0")(
		func(w http.ResponseWriter, h *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		},
	)(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusAccepted)
}

func TestCheckApiVersionWithMatchingVersionHeader(t *testing.T) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header["Draupnir-Version"] = []string{"1.0.0"}

	CheckAPIVersion("1.0.0")(
		func(w http.ResponseWriter, h *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		},
	)(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusAccepted)
}
