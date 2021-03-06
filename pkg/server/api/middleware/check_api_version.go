package middleware

import (
	"net/http"

	"github.com/gocardless/draupnir/pkg/server/api"
	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/gocardless/draupnir/pkg/version"
)

// CheckAPIVersion checks that the request has a Draupnir-Version header with a
// version that matches the server's major version, and is equal to or lower
// than the minor version.
//
// If the version doesn't match or the header is missing, it renders a 400 Bad
// Request.
func CheckAPIVersion(serverVersion string) chain.Middleware {
	return func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			versions := r.Header["Draupnir-Version"]
			if len(versions) == 0 {
				api.MissingApiVersion.Render(w, http.StatusBadRequest)
				return nil
			}

			major, minor, _, err := version.ParseSemver(serverVersion)

			// If we can't parse our server version then we shouldn't react by rejecting all
			// requests.
			if err == nil {
				requestVersion := versions[0]
				requestMajor, requestMinor, _, err := version.ParseSemver(requestVersion)

				if err != nil || major != requestMajor || minor < requestMinor {
					api.InvalidApiVersion(requestVersion).Render(w, http.StatusBadRequest)
					return nil
				}
			}

			return next(w, r)
		}
	}
}
