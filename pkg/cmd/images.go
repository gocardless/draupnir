package cmd

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/gocardless/draupnir/pkg/client/config"
	"github.com/gocardless/draupnir/pkg/models"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

func Images(logger log.Logger, insecure bool) *cobra.Command {
	c := &cobra.Command{
		Use:   "images",
		Short: "Manage images",
	}

	c.AddCommand(ImagesList(logger, insecure))
	c.AddCommand(ImagesCreate(logger, insecure))
	c.AddCommand(ImagesFinalise(logger, insecure))
	c.AddCommand(ImagesDestroy(logger, insecure))
	return c
}

func ImagesList(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List available images",
		Example: "draupnir images list",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}
			client := NewApiClient(cfg, insecure)

			images, err := client.ListImages()

			if err != nil {
				return errors.Wrap(err, "Could not fetch images")
			}

			for _, image := range images {
				fmt.Println(ImageToString(image))
			}
			return nil
		},
	}
}

func ImagesCreate(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new image",
		Example: `
draupnir images create [backedUpAt] [anon.sql]

[backedUpAt] an iso8601 timestamp defining when this backup was completed
[anonyimse.sql] path to an anonymisation script that will be run on image finalisation`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			backedUpAt, err := time.Parse(time.RFC3339, args[0])
			if err != nil {
				return errors.Wrap(err, "Invalid backedUpAt timestamp")
			}

			anonPath := args[1]
			anon, err := ioutil.ReadFile(anonPath)
			if err != nil {
				return errors.Wrap(err, "Invalid anonymisation script")
			}

			image, err := client.CreateImage(backedUpAt, anon)
			if err != nil {
				return errors.Wrap(err, "Could not create image")
			}

			fmt.Println(ImageToString(image))
			return nil
		},
	}
}

func ImagesFinalise(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "finalise",
		Short:   "Finalises an image (makes it ready)",
		Example: "draupnir images finalise [image id]",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}

			client := NewApiClient(cfg, insecure)

			imageID, err := strconv.Atoi(args[0])
			if err != nil {
				return errors.Wrap(err, "Invalid image ID")
			}

			image, err := client.FinaliseImage(imageID)
			if err != nil {
				return errors.Wrap(err, "Could not finalise image")
			}

			fmt.Println(ImageToString(image))
			return nil
		},
	}
}

func ImagesDestroy(logger log.Logger, insecure bool) *cobra.Command {
	return &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy an image",
		Example: "draupnir images destroy [image id]",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return errors.Wrap(err, "Could not load configuration")
			}
			client := NewApiClient(cfg, insecure)

			id := args[0]
			image, err := client.GetImage(id)
			if err != nil {
				return errors.Wrap(err, "Could not fetch image")
			}

			err = client.DestroyImage(image)
			if err != nil {
				return errors.Wrap(err, "Could not destroy image")
			}

			logger.With("id", image.ID).Info("Destroyed image")
			return nil
		},
	}
}

func ImageToString(i models.Image) string {
	return fmt.Sprintf("%2d [ %s - READY: %5t ]", i.ID, i.BackedUpAt.Format(time.RFC3339), i.Ready)
}
