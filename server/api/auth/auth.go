package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	google "google.golang.org/api/oauth2/v1"
)

const UPLOAD_USER_EMAIL = "upload"

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
	OAuthClient            OAuthClient
	SharedSecret           string
	TrustedUserEmailDomain string
}

func (g GoogleAuthenticator) AuthenticateRequest(r *http.Request) (string, error) {
	var accessToken string
	_, err := fmt.Sscanf(r.Header.Get("Authorization"), "Bearer %s", &accessToken)
	if err != nil {
		return "", fmt.Errorf("Error extracting token from Authorization header: %s", err.Error())
	}

	// abr uses a shared secret to authenticate
	if accessToken == g.SharedSecret {
		return UPLOAD_USER_EMAIL, nil
	}

	email, err := g.OAuthClient.LookupAccessToken(accessToken)
	if err != nil {
		return "", fmt.Errorf("Error looking up access token: %s", err.Error())
	}

	if !strings.HasSuffix(email, g.TrustedUserEmailDomain) {
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

type GoogleOAuthClient struct {
	Config *oauth2.Config
}

func (g GoogleOAuthClient) LookupAccessToken(refreshToken string) (string, error) {
	// Use the refresh token to obtain an access token
	token := &oauth2.Token{RefreshToken: refreshToken}
	tokenSource := g.Config.TokenSource(context.Background(), token)
	token, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("Error acquiring access token: %s", err.Error())
	}

	service, err := google.New(http.DefaultClient)
	if err != nil {
		return "", fmt.Errorf("Error initialising google oauth client: %s", err.Error())
	}
	tokenInfo, err := service.Tokeninfo().AccessToken(token.AccessToken).Do()
	if err != nil {
		return "", fmt.Errorf("Error getting info from Google: %s", err.Error())
	}
	return tokenInfo.Email, nil
}

// IntegrationTestOAuthClient is used for integration tests
type IntegrationTestOAuthClient struct{}

func (f IntegrationTestOAuthClient) LookupAccessToken(accessToken string) (string, error) {
	if accessToken == "the-integration-access-token" {
		return "integration-test@gocardless.com", nil
	}
	return "", errors.New("Invalid access token")
}
