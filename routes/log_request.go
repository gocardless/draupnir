package routes

import (
	"log"
	"net/http"
	"net/http/httptest"
	"time"
)

func LogRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// To capture the response, we replace the response writer with a response
		// recorder.
		recorder := httptest.NewRecorder()

		start := time.Now()

		// Call the next middleware
		next(recorder, r)

		duration := time.Since(start)

		// Log the request
		// TODO: allow the logger to be injected
		// TODO: support different log formats (e.g. JSON)
		log.Printf(
			"%s %s %d %f\n",
			r.Method,
			r.URL.String(),
			recorder.Code,
			duration.Seconds(),
		)

		// Copy the headers and body from the recorder to the response writer
		for k, v := range recorder.HeaderMap {
			w.Header()[k] = v
		}
		w.WriteHeader(recorder.Code)
		recorder.Body.WriteTo(w)
	}
}
