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
	AuthenticateRequest(*http.Request) (string, string, error)
	IsRefreshTokenValid(string) (bool, error, error)
}

type GoogleAuthenticator struct {
	OAuthClient            OAuthClient
	SharedSecret           string
	TrustedUserEmailDomain string
}

func (g GoogleAuthenticator) AuthenticateRequest(r *http.Request) (string, string, error) {
	var refreshToken string
	_, err := fmt.Sscanf(r.Header.Get("Authorization"), "Bearer %s", &refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("Error extracting token from Authorization header: %s", err.Error())
	}

	// abr uses a shared secret to authenticate
	if refreshToken == g.SharedSecret {
		return UPLOAD_USER_EMAIL, "", nil
	}

	email, err := g.OAuthClient.LookupAccessToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("Error looking up access token: %s", err.Error())
	}

	if !strings.HasSuffix(email, g.TrustedUserEmailDomain) {
		return "", "", errors.New("Email not valid")
	}

	return email, refreshToken, nil
}

// IsRefreshTokenValid checks if a refresh token is valid by requesting a new
// access token with it.
// The first return parameter will return true only if the token is currently
// valid, i.e. the user has not been suspended or revoked their token.
// The second return parameter is an error that is populated only if there has
// been an error when attempting to determine if the token is valid.
// The third return parameter is an error that is populated only if the token
// is not currently valid, so can be used to determine the reason for its invalidity.
func (g GoogleAuthenticator) IsRefreshTokenValid(refreshToken string) (bool, error, error) {
	_, err := g.OAuthClient.LookupAccessToken(refreshToken)
	if err != nil {
		// invalid_grant is the error code returned when a user is deleted,
		// suspended, or the application access has been revoked
		if strings.Contains(err.Error(), "invalid_grant") {
			return false, nil, err
		}
		return false, err, nil
	}
	return true, nil, nil
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

func (f IntegrationTestOAuthClient) LookupAccessToken(refreshToken string) (string, error) {
	if refreshToken == "the-integration-access-token" {
		return "integration-test@gocardless.com", nil
	}
	return "", errors.New("Invalid access token")
}
