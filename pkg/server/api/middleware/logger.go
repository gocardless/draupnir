package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gocardless/draupnir/pkg/server/api/chain"
	"github.com/prometheus/common/log"
)

type key int

// This, sadly is exported so we can inject fake loggers in tests.
// See routes.createRequest in server/api/routes/fakes.go
const LoggerKey key = 1

func NewRequestLogger(logger log.Logger) chain.Middleware {
	return func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			// To capture the response, we replace the response writer with a response
			// recorder.
			recorder := httptest.NewRecorder()

			// Add a collection of headers that might be useful to log
			scopedLogger := logger.
				With("method", r.Method).
				With("path", r.URL.String()).
				With("headers__host", r.Header.Get("Host")).
				With("headers__content_type", r.Header.Get("Content-Type")).
				With("headers__x_forwarded_for", r.Header.Get("X-Forwarded-For")).
				With("headers__x_cloud_trace_context", r.Header.Get("X-Cloud-Trace-Context")).
				With("headers__draupnir_version", r.Header.Get("Draupnir-Version")).
				With("headers__user_agent", r.Header.Get("User-Agent"))

			// This coupling between middlewares isn't great, but it is valuable to
			// get the IP address injected into the logger early in the chain.
			userIPAddress, err := GetUserIPAddress(r)
			if err != nil {
				logger.Warn("Unable to annotate request logger with real user IP address: ", err)
			} else {
				scopedLogger = scopedLogger.
					With("client_ip_address", userIPAddress)
			}

			// Inject the logger into the request's context
			r = r.WithContext(context.WithValue(r.Context(), LoggerKey, &scopedLogger))

			// Call the next middleware and time it
			start := time.Now()
			err = next(recorder, r)
			duration := time.Since(start)

			requestLine := fmt.Sprintf(
				"%s %s %d %f",
				r.Method,
				r.URL.String(),
				recorder.Code,
				duration.Seconds(),
			)

			scopedLogger.
				With("status", recorder.Code).
				With("duration", duration.Seconds()).
				Info(requestLine)

			// Copy the headers and body from the recorder to the response writer
			for k, v := range recorder.HeaderMap {
				w.Header()[k] = v
			}
			w.WriteHeader(recorder.Code)
			recorder.Body.WriteTo(w)
			return err
		}
	}
}

func GetLogger(r *http.Request) (log.Logger, error) {
	logger, ok := r.Context().Value(LoggerKey).(*log.Logger)
	if !ok {
		return nil, errors.New("Could not acquire logger")
	}
	return *logger, nil
}
