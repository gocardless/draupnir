package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func shouldNeverBeCalled(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, h *http.Request) {
		t.Fatal("this route should never be called")
	}
}

func respondsWithStatus(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, h *http.Request) {
		w.WriteHeader(status)
	}
}

func TestCheckApiVersion(t *testing.T) {
	testCases := []struct {
		name          string
		headerVersion string
		handler       http.HandlerFunc
		apiError      APIError
		code          int
	}{
		{
			"when version matches, calls handler",
			"1.1.0",
			respondsWithStatus(http.StatusAccepted),
			APIError{},
			http.StatusAccepted,
		},
		{
			"when minor is lower, calls handler",
			"1.0.0",
			respondsWithStatus(http.StatusAccepted),
			APIError{},
			http.StatusAccepted,
		},
		{
			"when minor is higher, responds with error",
			"1.2.0",
			shouldNeverBeCalled(t),
			invalidApiVersion("1.2.0"),
			http.StatusBadRequest,
		},
		{
			"when header major version is different, responds with error",
			"0.1.0",
			shouldNeverBeCalled(t),
			invalidApiVersion("0.1.0"),
			http.StatusBadRequest,
		},
		{
			"when header is missing, responds with error",
			"",
			shouldNeverBeCalled(t),
			missingApiVersion,
			http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/foo", nil)

			if tc.headerVersion != "" {
				req.Header["Draupnir-Version"] = []string{tc.headerVersion}
			}

			CheckAPIVersion("1.1.0")(tc.handler)(recorder, req)

			if tc.apiError.ID != "" {
				var response APIError
				err := json.NewDecoder(recorder.Body).Decode(&response)

				assert.Nil(t, err, "failed to decode response into APIError")
				assert.EqualValues(t, tc.apiError, response)
			}

			assert.Equal(t, tc.code, recorder.Code)
		})
	}
}
