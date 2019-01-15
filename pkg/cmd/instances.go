package cmd

import (
	"fmt"
	"time"

	"github.com/gocardless/draupnir/pkg/client/config"
	"github.com/gocardless/draupnir/pkg/models"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

func Instances(logger log.Logger, insecure bool) *cobra.Command {
	c := &cobra.Command{
		Use:   "instances",
		Short: "Manage your instances",
	}

	c.AddCommand(InstancesList(logger, insecure))
	c.AddCommand(InstancesCreate(logger, insecure))
	c.AddCommand(InstancesDestroy(logger, insecure))
	return c
}

func InstancesList(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List your instances",
		Example: "draupnir list",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			instances, err := client.ListInstances()
			if err != nil {
				return errors.Wrap(err, "Could not fetch instances")
			}
			for _, instance := range instances {
				fmt.Println(InstanceToString(instance))
			}
			return nil
		},
	}
}

func InstancesCreate(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "create",
		Short:   "Create a new instance",
		Args:    cobra.MaximumNArgs(1),
		Example: "draupnir instances create [image id]",
		RunE: func(cmd *cobra.Command, args []string) error {
			var image models.Image

			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			if len(args) == 0 {
				image, err = client.GetLatestImage()
			} else {
				image, err = client.GetImage(args[0])
			}

			if err != nil {
				return errors.Wrap(err, "Could not fetch image")
			}

			instance, err := client.CreateInstance(image)
			if err != nil {
				return errors.Wrap(err, "Could not create instance")
			}

			logger.With("id", instance.ID).With("image", image.ID).Info("Created instance")
			fmt.Println(InstanceToString(instance))
			return nil
		},
	}
}

func InstancesDestroy(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy an instance",
		Example: "draupnir instances destroy [image id]",
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
				return errors.Wrap(err, "Could not fetch instance")
			}

			err = client.DestroyInstance(instance)
			if err != nil {
				return errors.Wrap(err, "Could not destroy instance")
			}

			logger.With("id", instance.ID).Info("Destroyed instance")
			return nil
		},
	}
}

func InstanceToString(i models.Instance) string {
	return fmt.Sprintf("%2d [ PORT: %d - %s ]", i.ID, i.Port, i.CreatedAt.Format(time.RFC3339))
}
