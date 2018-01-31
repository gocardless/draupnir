package routes

import (
	"net/http"

	"github.com/gocardless/draupnir/version"
)

// CheckAPIVersion checks that the request has a Draupnir-Version header with a
// version that matches the server's major version, and is equal to or lower
// than the minor version.
//
// If the version doesn't match or the header is missing, it renders a 400 Bad
// Request.
func CheckAPIVersion(serverVersion string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			versions := r.Header["Draupnir-Version"]
			if len(versions) == 0 {
				RenderError(w, http.StatusBadRequest, missingApiVersion)
				return
			}

			requestVersion := versions[0]
			requestMajor, requestMinor, _ := version.ParseSemver(requestVersion)
			major, minor, _ := version.ParseSemver(serverVersion)

			if major != requestMajor || minor < requestMinor {
				RenderError(w, http.StatusBadRequest, invalidApiVersion(requestVersion))
				return
			}

			next(w, r)
		}
	}
}
