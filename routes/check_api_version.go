package routes

import (
	"net/http"
	"github.com/gocardless/draupnir/version"
)

func CheckAPIVersion(w http.ResponseWriter, r *http.Request) bool {
	versions := r.Header["Draupnir-Version"]
	if len(versions) == 0 {
		RenderError(w, http.StatusBadRequest, missingApiVersion)
		return false
	}
	requestVersion := versions[0]

	if requestVersion != version.Version {
		RenderError(w, http.StatusBadRequest, invalidApiVersion(requestVersion))
		return false
	}

	return true
}
