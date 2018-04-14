package routes

import (
	"net/http"

	raven "github.com/getsentry/raven-go"
	"github.com/gocardless/draupnir/server/api/routes/chain"
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
			RenderError(w, http.StatusInternalServerError, internalServerError)
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
