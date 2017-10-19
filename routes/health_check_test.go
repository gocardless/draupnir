package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/health_check", nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := http.HandlerFunc(HealthCheck)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, recorder.Code, http.StatusOK)

	var response map[string]string
	err = json.NewDecoder(recorder.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, response, map[string]string{"status": "ok"})
}
