package routes

import (
	"net/http"

	raven "github.com/getsentry/raven-go"
	"github.com/gocardless/draupnir/routes/chain"
	"github.com/prometheus/common/log"
)

func NewErrorHandler(logger log.Logger) chain.TerminatingMiddleware {
	return func(next chain.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := next(w, r)
			if err != nil {
				logger.With("http_request", r).Error(err.Error())
				RenderError(w, http.StatusInternalServerError, internalServerError)
			}
		}
	}
}

func NewSentryErrorHandler(logger log.Logger, sentry *raven.Client) chain.TerminatingMiddleware {
	return func(next chain.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			err := next(w, r)
			if err != nil {
				logger.With("http_request", r).Error(err.Error())
				RenderError(w, http.StatusInternalServerError, internalServerError)
				sentry.CaptureError(err, map[string]string{})
			}
		}
	}
}
