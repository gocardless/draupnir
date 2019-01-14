package middleware

import (
	"net/http"

	raven "github.com/getsentry/raven-go"
	"github.com/gocardless/draupnir/pkg/server/api"
	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/prometheus/common/log"
)

func NewErrorHandler(logger log.Logger) chain.TerminatingMiddleware {
	return func(next chain.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := next(w, r)
			if err != nil {
				logger.With("http_request", r).Error(err.Error())
			}
		}
	}
}

func DefaultErrorRenderer(next chain.Handler) chain.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := next(w, r)
		if err != nil {
			api.InternalServerError.Render(w, http.StatusInternalServerError)
		}
		return err
	}
}

func NewSentryReporter(sentry *raven.Client) chain.Middleware {
	return func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			err := next(w, r)
			if err != nil {
				sentry.CaptureError(err, map[string]string{})
			}
			return err
		}
	}
}
