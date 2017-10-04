package routes

import "net/http"

const mediaType = "application/json"

// SetHeaders takes an HTTP handler and wraps it, setting the following HTTP
// headers on the response:
// Content-Type: routes.mediaType (application/json)
// Draupnir-Version: the current draupnir version
func SetHeaders(version string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", mediaType)
		w.Header().Set("Draupnir-Version", version)
		handler(w, r)
	}
}
