package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"golang.org/x/oauth2"
)

func authorise(clientID string, clientSecret string) (oauth2.Token, error) {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: "https://www.googleapis.com/oauth2/v4/token",
		},
		RedirectURL: "http://127.0.0.1:8181",
	}

	url := config.AuthCodeURL("state")
	err := exec.Command("open", url).Run()
	if err != nil {
		fmt.Printf("Visit the URL for the auth dialog: %v\n", url)
	}

	token := make(chan oauth2.Token, 1)
	tokenError := make(chan error, 1)
	go listenForOAuthCallback(config, token, tokenError)

	select {
	case t := <-token:
		return t, nil
	case e := <-tokenError:
		return oauth2.Token{}, e
	case <-time.After(time.Second * 15):
		return oauth2.Token{}, errors.New("Request timed out")
	}
}

func listenForOAuthCallback(config *oauth2.Config, result chan oauth2.Token, resultError chan error) {
	http.ListenAndServe("127.0.0.1:8181", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		r.ParseForm()

		respError := r.Form.Get("error")
		respCode := r.Form.Get("code")

		if respError != "" {
			resultError <- errors.New(respError)
			return
		}

		if respCode != "" {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			token, err := config.Exchange(ctx, respCode)

			if err != nil {
				resultError <- err
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<h1>Success!</h1><h3>You can close this tab</h3><script>window.close()</script>"))

			result <- *token
		}
	}))
}
