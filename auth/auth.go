package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/api/oauth2/v1"
)

type Authenticator interface {
	// AuthenticateRequest takes an HTTP request and
	// attempts to authenticate it.
	// It returns the email address of the authenticated
	// user, or an error.
	// TODO: maybe this should be Authenticate(string) (string, error)
	// Taking the Authorization header and returning the email address
	AuthenticateRequest(*http.Request) (string, error)
}

type GoogleAuthenticator struct {
	OAuthClient  OAuthClient
	SharedSecret string
}

func (g GoogleAuthenticator) AuthenticateRequest(r *http.Request) (string, error) {
	var accessToken string
	_, err := fmt.Sscanf(r.Header.Get("Authorization"), "Bearer %s", &accessToken)
	if err != nil {
		return "", err
	}

	// abr uses a shared secret to authenticate
	if accessToken == g.SharedSecret {
		return "upload", nil
	}

	email, err := g.OAuthClient.LookupAccessToken(accessToken)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(email, "@gocardless.com") {
		return "", errors.New("Email not valid")
	}

	return email, nil
}

type OAuthClient interface {
	// LookupAccessToken takes an access token
	// and returns the email address associated
	// with it
	LookupAccessToken(string) (string, error)
}

type GoogleOAuthClient struct{}

func (g GoogleOAuthClient) LookupAccessToken(accessToken string) (string, error) {
	service, err := oauth2.New(http.DefaultClient)
	if err != nil {
		return "", err
	}
	tokenInfo, err := service.Tokeninfo().AccessToken(accessToken).Do()
	if err != nil {
		return "", err
	}
	return tokenInfo.Email, nil
}

type FakeOAuthClient struct{}

func (f FakeOAuthClient) LookupAccessToken(accessToken string) (string, error) {
	if accessToken == "the-integration-access-token" {
		return "integration-test@gocardless.com", nil
	}
	return "", errors.New("Invalid access token")
}
