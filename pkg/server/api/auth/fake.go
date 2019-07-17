package auth

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

type FakeAuthenticator struct {
	MockAuthenticateRequest func(r *http.Request) (string, string, error)
	MockIsRefreshTokenValid func(string) (bool, error, error)
}

func (f FakeAuthenticator) AuthenticateRequest(r *http.Request) (string, string, error) {
	return f.MockAuthenticateRequest(r)
}

func (f FakeAuthenticator) IsRefreshTokenValid(refreshToken string) (bool, error, error) {
	return f.MockIsRefreshTokenValid(refreshToken)
}

type FakeOAuthClient struct {
	MockAuthCodeURL func(string, ...oauth2.AuthCodeOption) string
	MockExchange    func(context.Context, string) (*oauth2.Token, error)
}

func (c *FakeOAuthClient) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c.MockAuthCodeURL(state, opts...)
}

func (c *FakeOAuthClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return c.MockExchange(ctx, code)
}

func FakeOauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "the-client-id",
		ClientSecret: "the-client-secret",
		Scopes:       []string{"the-scope"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.org/auth",
			TokenURL: "https://example.org/token",
		},
		RedirectURL: "https://draupnir.org/redirect",
	}
}
