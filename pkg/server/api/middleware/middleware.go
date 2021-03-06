package middleware

import (
	"net/http"

	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/gocardless/draupnir/pkg/version"
)

func AsJSON(next chain.Handler) chain.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		return next(w, r)
	}
}

func WithVersion(next chain.Handler) chain.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Draupnir-Version", version.Version)
		return next(w, r)
	}
}
