package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gocardless/draupnir/client"
	"github.com/gocardless/draupnir/models"
	"github.com/urfave/cli"
)

func main() {
	DOMAIN := os.Getenv("DRAUPNIR_DOMAIN")
	if DOMAIN == "" {
		fmt.Println("You must set DRAUPNIR_DOMAIN")
		return
	}
	client := client.Client{URL: "http://" + DOMAIN}

	app := cli.NewApp()
	app.Name = "draupnir"
	app.Usage = "A client for draupnir"
	app.Commands = []cli.Command{
		{
			Name:    "list-instances",
			Aliases: []string{},
			Usage:   "list your instances",
			Action: func(c *cli.Context) error {
				instances, err := client.ListInstances()
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}
				for _, instance := range instances {
					fmt.Println(InstanceToString(instance))
				}
				return nil
			},
		},
		{
			Name:    "list-images",
			Aliases: []string{},
			Usage:   "list your images",
			Action: func(c *cli.Context) error {
				images, err := client.ListImages()

				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}
				for _, image := range images {
					fmt.Println(ImageToString(image))
				}
				return nil
			},
		},
		{
			Name:    "create-instance",
			Aliases: []string{},
			Usage:   "create a new instance",
			Action: func(c *cli.Context) error {
				id := c.Args().First()
				if id == "" {
					fmt.Println("error: must supply an image id")
					return nil
				}

				image, err := client.GetImage(id)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				instance, err := client.CreateInstance(image)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				fmt.Println("Created")
				fmt.Println(InstanceToString(instance))
				return nil
			},
		},
		{
			Name:    "destroy-instance",
			Aliases: []string{},
			Usage:   "destroy an instance",
			Action: func(c *cli.Context) error {
				id := c.Args().First()
				if id == "" {
					fmt.Println("error: must supply an instance id")
					return nil
				}

				instance, err := client.GetInstance(id)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				err = client.DestroyInstance(instance)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				fmt.Printf("Destroyed %d\n", instance.ID)
				return nil
			},
		},
		{
			Name:    "destroy-image",
			Aliases: []string{},
			Usage:   "destroy an image",
			Action: func(c *cli.Context) error {
				id := c.Args().First()
				if id == "" {
					fmt.Println("error: must supply an image id")
					return nil
				}

				image, err := client.GetImage(id)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				err = client.DestroyImage(image)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				fmt.Printf("Destroyed %d\n", image.ID)
				return nil
			},
		},
		{
			Name:    "env",
			Aliases: []string{},
			Usage:   "show the environment variables to connect to an instance",
			Action: func(c *cli.Context) error {
				id := c.Args().First()
				if id == "" {
					fmt.Println("error: must supply an instance id")
					return nil
				}

				instance, err := client.GetInstance(id)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				fmt.Printf("export PGHOST=%s PGPORT=%d PGUSER=postgres\n", DOMAIN, instance.Port)

				return nil
			},
		},
	}

	app.Run(os.Args)
}

func ImageToString(i models.Image) string {
	return fmt.Sprintf("%d [ %s - READY: %5t ]", i.ID, i.BackedUpAt.Format(time.RFC3339), i.Ready)
}

func InstanceToString(i models.Instance) string {
	return fmt.Sprintf("%2d [ PORT: %d - %s ]", i.ID, i.Port, i.CreatedAt.Format(time.RFC3339))
}
