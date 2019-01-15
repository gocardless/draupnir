package cmd

import (
	"fmt"
	"os"

	"github.com/gocardless/draupnir/pkg/client/config"
	"github.com/gocardless/draupnir/pkg/models"
	api "github.com/gocardless/draupnir/pkg/server/api/client"
	"github.com/gocardless/draupnir/pkg/version"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

const quickStart string = `
  # Create a new database instance and connect to it
  draupnir-client authenticate
  eval $(draupnir-client new)
  psql
`

func Root() *cobra.Command {
	logger := log.With("app", "draupnir")

	command := &cobra.Command{
		Use:     "draupnir",
		Version: version.Version,
		Short:   "Draupnir provides database instances on-demand",
		Example: quickStart,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	var insecure bool
	command.PersistentFlags().BoolVar(&insecure, "insecure", false, "Don't validate certificates when connecting")

	command.AddCommand(Server(logger))
	command.AddCommand(Config(logger))
	command.AddCommand(Authenticate(logger, insecure))
	command.AddCommand(Instances(logger, insecure))
	command.AddCommand(Images(logger, insecure))
	command.AddCommand(Env(logger, insecure))
	command.AddCommand(New(logger, insecure))

	return command
}

func Env(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "env",
		Short:   "Show the environment variables to connect to an instance",
		Example: "draupnir env [instance id]",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			id := args[0]
			instance, err := client.GetInstance(id)
			if err != nil {
				logger.With("error", err).Fatal("Could not fetch instance")
			}

			showExportCommand(cfg, instance)
			return nil
		},
	}
}

func New(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "new",
		Short:   "create a new instance",
		Example: "draupnir new",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			image, err := client.GetLatestImage()
			if err != nil {
				logger.With("error", err).Fatal("Could not fetch image")
			}

			instance, err := client.CreateInstance(image)
			if err != nil {
				logger.With("error", err).Fatal("Could not create instance")
			}

			showExportCommand(cfg, instance)
			return nil
		},
	}
}

// TODO: init client in root command and pass down
func NewApiClient(cfg config.Config, insecure bool) api.Client {
	return api.NewClient(fmt.Sprintf("https://%s", cfg.Domain), cfg.Token, insecure)
}

func showExportCommand(config config.Config, instance models.Instance) {
	// The database precedence is config -> environment variable -> 'postgres'
	database := config.Database
	if database == "" {
		database = os.Getenv("PGDATABASE")
	}
	if database == "" {
		database = "postgres"
	}
	fmt.Printf(
		"export PGHOST=%s PGPORT=%d PGUSER=postgres PGPASSWORD='' PGDATABASE=%s\n",
		config.Domain,
		instance.Port,
		database,
	)
}
