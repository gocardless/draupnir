package routes

import (
	"context"
	"errors"
	"net/http"

	"github.com/gocardless/draupnir/auth"
	"github.com/gocardless/draupnir/routes/chain"
)

const authUserKey key = 2

// Authenticate uses the provided authenticator to authenticate the request.
// On success, it yields to the next handler in the chain.
// On failure, it renders 401 Unauthorized.
func Authenticate(authenticator auth.Authenticator) chain.Middleware {
	return func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			logger, err := GetLogger(r)
			if err != nil {
				return err
			}

			email, err := authenticator.AuthenticateRequest(r)
			if err != nil {
				logger.Info(err.Error())
				RenderError(w, http.StatusUnauthorized, unauthorizedError)
				return nil
			}

			r = r.WithContext(context.WithValue(r.Context(), authUserKey, email))
			return next(w, r)
		}
	}
}

func GetAuthenticatedUser(r *http.Request) (string, error) {
	user, ok := r.Context().Value(authUserKey).(string)
	if !ok {
		return "", errors.New("Could not acquire authenticated user")
	}
	return user, nil
}
