package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestSetHeaders(t *testing.T) {
	handler := func(w http.ResponseWriter, h *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "hi!")
	}

	version := "1.0.0"
	wrappedHandler := SetHeaders(version, handler)

	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	router.HandleFunc("/test", wrappedHandler)
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, []string{mediaType}, recorder.HeaderMap["Content-Type"])
	assert.Equal(t, []string{version}, recorder.HeaderMap["Draupnir-Version"])
	assert.Equal(t, string(recorder.Body.Bytes()), "hi!")
}
