package cmd

import (
	"fmt"
	"strings"

	"github.com/gocardless/draupnir/pkg/client/config"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

func Config(logger log.Logger) *cobra.Command {
	c := &cobra.Command{
		Use:   "config",
		Short: "Get and set configuration options",
	}
	c.AddCommand(ConfigShow(logger))
	c.AddCommand(ConfigSet(logger))
	return c
}

func ConfigShow(logger log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Short:   "Show the current configuration",
		Example: "draupnir config show",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			domain := cfg.Domain
			accessToken := cfg.Token.AccessToken
			database := cfg.Database

			fmt.Printf("Domain: %s\n", domain)
			if len(accessToken) < 10 {
				// Go doesn't appear to have a safe subslice operation...
				fmt.Printf("Access Token: %s\n", accessToken)
			} else {
				fmt.Printf("Access Token: %s****\n", accessToken[0:10])
			}
			fmt.Printf("Database: %s\n", database)
			return nil
		},
	}
}

func ConfigSet(logger log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Set a config value",
		Example: `
draupnir config set [key] [value]

[key] can take the following values:
    domain: The domain of the draupnir server.
    database: The default database to connect to. If not set, defaults to the PGDATABASE environment variable.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := args[1]

			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			switch strings.ToLower(key) {
			case "domain":
				cfg.Domain = val
			case "database":
				cfg.Database = val
			default:
				return errors.New("Invalid key")
			}

			err = config.Store(cfg)
			if err != nil {
				return errors.Wrap(err, "Could not store configuration")
			}
			return nil
		},
	}
}
