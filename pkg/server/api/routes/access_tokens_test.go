package routes

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gocardless/draupnir/pkg/server/api/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func TestAuthenticate(t *testing.T) {
	req, recorder, _ := createRequest(t, "GET", "/authenticate?state=foo", nil)

	routeSet := AccessTokens{
		Callbacks: make(map[string]chan OAuthCallback),
		Client:    auth.FakeOauthConfig(),
	}

	errorHandler := FakeErrorHandler{}
	router := mux.NewRouter()
	router.HandleFunc("/authenticate", errorHandler.Handle(routeSet.Authenticate))
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	expectedRedirect := fmt.Sprintf(
		"https://example.org/auth?access_type=offline&client_id=%s&redirect_uri=%s&response_type=%s&scope=%s&state=%s",
		"the-client-id",
		url.QueryEscape("https://draupnir.org/redirect"),
		"code",
		"the-scope",
		"foo",
	)

	assert.Equal(t, http.StatusFound, response.StatusCode)
	assert.Equal(t, []string{expectedRedirect}, response.Header["Location"])
	assert.Equal(t, 0, len(recorder.Body.Bytes()))
	assert.Nil(t, errorHandler.Error)
}

func TestCallback(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := ""

	path := oauthCallbackPath(state, code, _error)

	req, recorder, logs := createRequest(t, "GET", path, nil)

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	oauthClient := auth.FakeOAuthClient{
		MockExchange: func(ctx context.Context, _code string) (*oauth2.Token, error) {
			assert.Equal(t, code, _code)
			return &oauth2.Token{RefreshToken: "the-access-token"}, nil
		},
	}

	errorHandler := FakeErrorHandler{}

	routeSet := AccessTokens{Callbacks: callbacks, Client: &oauthClient}

	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", errorHandler.Handle(routeSet.Callback))
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, []string{"text/html"}, response.Header["Content-Type"])
	assert.Contains(t, responseBody.String(), "Success!")
	assert.Empty(t, logs.String())
	assert.Nil(t, errorHandler.Error)

	select {
	case result := <-callback:
		assert.Equal(t, OAuthCallback{Token: oauth2.Token{RefreshToken: "the-access-token"}, Error: nil}, result)
	default:
		t.Fatal("Received nothing in channel")
	}
}

func TestCallbackWithResponseError(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := "some_error"

	path := oauthCallbackPath(state, code, _error)

	req, recorder, _ := createRequest(t, "GET", path, nil)

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	errorHandler := FakeErrorHandler{}

	routeSet := AccessTokens{Callbacks: callbacks}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", errorHandler.Handle(routeSet.Callback))
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Empty(t, responseBody.String())
	assert.Equal(t, "some_error", errorHandler.Error.Error())

	select {
	case result := <-callback:
		err := result.Error
		assert.Equal(t, _error, err.Error())
	default:
		t.Fatal("Received nothing in channel")
	}
}

func TestCallbackWithEmptyResponseCode(t *testing.T) {
	state := "foo"
	code := ""
	_error := ""

	path := oauthCallbackPath(state, code, _error)

	req, recorder, logs := createRequest(t, "GET", path, nil)

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	errorHandler := FakeErrorHandler{}

	routeSet := AccessTokens{Callbacks: callbacks}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", errorHandler.Handle(routeSet.Callback))
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Empty(t, responseBody.String())
	assert.Contains(t, logs.String(), "msg=\"empty oauth response code\"")
	assert.Equal(t, "OAuth callback response code is empty", errorHandler.Error.Error())

	select {
	case result := <-callback:
		err := result.Error
		assert.Equal(t, "OAuth callback response code is empty", err.Error())
	default:
		t.Fatal("Received nothing in channel")
	}
}

func TestCallbackWithFailedTokenExchange(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := ""

	path := oauthCallbackPath(state, code, _error)

	req, recorder, logs := createRequest(t, "GET", path, nil)

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	oauthClient := auth.FakeOAuthClient{
		MockExchange: func(ctx context.Context, _code string) (*oauth2.Token, error) {
			assert.Equal(t, code, _code)
			return &oauth2.Token{}, errors.New("token exchange failed")
		},
	}

	errorHandler := FakeErrorHandler{}

	routeSet := AccessTokens{Callbacks: callbacks, Client: &oauthClient}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", errorHandler.Handle(routeSet.Callback))
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Empty(t, responseBody.String())
	assert.Empty(t, logs.String())
	assert.Equal(t, "token exchange error: token exchange failed", errorHandler.Error.Error())

	select {
	case result := <-callback:
		err := result.Error
		assert.Equal(t, "token exchange error: token exchange failed", err.Error())
	default:
		t.Fatal("Received nothing in channel")
	}
}

func TestCallbackWithTimedOutTokenExchange(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := ""

	path := oauthCallbackPath(state, code, _error)

	req, recorder, logs := createRequest(t, "GET", path, nil)

	// Set the request to time out immediately
	ctx, _ := context.WithTimeout(req.Context(), 0)
	req = req.WithContext(ctx)

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	oauthClient := auth.FakeOAuthClient{
		MockExchange: func(ctx context.Context, _code string) (*oauth2.Token, error) {
			assert.Equal(t, code, _code)
			select {
			case <-ctx.Done():
				return &oauth2.Token{}, errors.New("timeout")
			}
		},
	}

	errorHandler := FakeErrorHandler{}

	routeSet := AccessTokens{Callbacks: callbacks, Client: &oauthClient}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", errorHandler.Handle(routeSet.Callback))
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err := responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Empty(t, responseBody.String())
	assert.Empty(t, logs.String())
	assert.Equal(t, errorHandler.Error.Error(), "token exchange error: timeout")

	select {
	case result := <-callback:
		err := result.Error
		assert.Equal(t, "token exchange error: timeout", err.Error())
	default:
		t.Fatal("Received nothing in channel")
	}
}

func oauthCallbackPath(state string, code string, _error string) string {
	return fmt.Sprintf(
		"/oauth_callback?state=%s&code=%s&error=%s",
		state,
		code,
		_error,
	)
}
