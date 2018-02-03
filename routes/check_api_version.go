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

			major, minor, _, err := version.ParseSemver(serverVersion)

			// If we can't parse our server version then we shouldn't react by rejecting all
			// requests.
			if err == nil {
				requestVersion := versions[0]
				requestMajor, requestMinor, _, err := version.ParseSemver(requestVersion)

				if err != nil || major != requestMajor || minor < requestMinor {
					RenderError(w, http.StatusBadRequest, invalidApiVersion(requestVersion))
					return
				}
			}

			next(w, r)
		}
	}
}
