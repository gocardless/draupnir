package routes

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func TestAuthenticate(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/authenticate?state=foo", nil)
	if err != nil {
		t.Fatal(err)
	}

	routeSet := AccessTokens{
		Callbacks: make(map[string]chan OAuthCallback),
		Client:    fakeOauthConfig(),
	}

	router := mux.NewRouter()
	router.HandleFunc("/authenticate", routeSet.Authenticate)
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
}

func TestCallback(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := ""

	path := oauthCallbackPath(state, code, _error)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	oauthClient := FakeOAuthClient{
		_Exchange: func(ctx context.Context, _code string) (*oauth2.Token, error) {
			assert.Equal(t, code, _code)
			return &oauth2.Token{AccessToken: "the-access-token"}, nil
		},
	}

	routeSet := AccessTokens{Callbacks: callbacks, Client: &oauthClient}

	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", routeSet.Callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err = responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, []string{"text/html"}, response.Header["Content-Type"])
	assert.Contains(t, responseBody.String(), "Success!")

	select {
	case result := <-callback:
		assert.Equal(t, OAuthCallback{Token: oauth2.Token{AccessToken: "the-access-token"}, Error: nil}, result)
	default:
		t.Fatal("Received nothing in channel")
	}
}

func TestCallbackWithResponseError(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := "some_error"

	path := oauthCallbackPath(state, code, _error)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	routeSet := AccessTokens{Callbacks: callbacks}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", routeSet.Callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err = responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Contains(t, responseBody.String(), "There was an error")

	select {
	case result := <-callback:
		err = result.Error
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

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	routeSet := AccessTokens{Callbacks: callbacks}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", routeSet.Callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err = responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Contains(t, responseBody.String(), "There was an error")
	assert.Contains(t, responseBody.String(), "OAuth callback response code is empty")

	select {
	case result := <-callback:
		err = result.Error
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

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	oauthClient := FakeOAuthClient{
		_Exchange: func(ctx context.Context, _code string) (*oauth2.Token, error) {
			assert.Equal(t, code, _code)
			return &oauth2.Token{}, errors.New("token exchange failed")
		},
	}

	routeSet := AccessTokens{Callbacks: callbacks, Client: &oauthClient}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", routeSet.Callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err = responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Contains(t, responseBody.String(), "There was an error")
	assert.Contains(t, responseBody.String(), "token exchange failed")

	select {
	case result := <-callback:
		err = result.Error
		assert.Equal(t, "Token exchange error: token exchange failed", err.Error())
	default:
		t.Fatal("Received nothing in channel")
	}
}

func TestCallbackWithTimedOutTokenExchange(t *testing.T) {
	state := "foo"
	code := "some_code"
	_error := ""

	path := oauthCallbackPath(state, code, _error)

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		t.Fatal(err)
	}

	callback := make(chan OAuthCallback, 1)
	callbacks := make(map[string]chan OAuthCallback)
	callbacks[state] = callback

	oauthClient := FakeOAuthClient{
		_Exchange: func(ctx context.Context, _code string) (*oauth2.Token, error) {
			assert.Equal(t, code, _code)
			select {
			case <-ctx.Done():
				return &oauth2.Token{}, errors.New("timeout")
			}
		},
	}

	routeSet := AccessTokens{Callbacks: callbacks, Client: &oauthClient}
	router := mux.NewRouter()
	router.HandleFunc("/oauth_callback", routeSet.Callback)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	response := recorder.Result()

	responseBody := bytes.Buffer{}
	if _, err = responseBody.ReadFrom(response.Body); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode)
	assert.Contains(t, responseBody.String(), "There was an error")
	assert.Contains(t, responseBody.String(), "timeout")

	select {
	case result := <-callback:
		err = result.Error
		assert.Equal(t, "Token exchange error: timeout", err.Error())
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
