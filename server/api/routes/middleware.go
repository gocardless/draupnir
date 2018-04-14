package routes

import (
	"net/http"

	"github.com/gocardless/draupnir/server/api/chain"
	"github.com/gocardless/draupnir/version"
)

func AsJSON(next chain.Handler) chain.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		next(w, r)
		return nil
	}
}

func WithVersion(next chain.Handler) chain.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Draupnir-Version", version.Version)
		next(w, r)
		return nil
	}
}
