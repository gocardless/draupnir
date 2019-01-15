package cmd

import (
	"github.com/gocardless/draupnir/pkg/server"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

func Server(logger log.Logger) *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the draupnir server",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			err = server.Run(logger)
			if err != nil {
				logger.With("error", err.Error()).Fatal("Failed to start server")
			}
			return nil
		},
	}
}
