package cmd

import (
	"fmt"
	"math/rand"
	"os/exec"

	"github.com/gocardless/draupnir/pkg/client/config"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

func Authenticate(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:   "authenticate",
		Short: "Authenticate with Google",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			if cfg.Token.RefreshToken != "" {
				logger.Info("You're already authenticated")
				return nil
			}

			state := fmt.Sprintf("%d", rand.Int31())

			url := fmt.Sprintf("https://%s/authenticate?state=%s", cfg.Domain, state)
			err = exec.Command("open", url).Run()
			if err != nil {
				fmt.Printf("Visit this link in your browser: %s\n", url)
			}

			token, err := client.CreateAccessToken(state)
			if err != nil {
				return errors.Wrap(err, "Could not create access token")
			}

			cfg.Token = token
			err = config.Store(cfg)
			if err != nil {
				return errors.Wrap(err, "Could not store configuration")
			}

			logger.Info("Successfully authenticated.")
			return nil
		},
	}
}
