package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gocardless/draupnir/client"
	"github.com/gocardless/draupnir/models"
	"github.com/urfave/cli"
)

type Config struct {
	Domain      string
	AccessToken string
}

var version string
var clientID string
var clientSecret string

func LoadConfig() (Config, error) {
	config := Config{Domain: "set-me-to-a-real-domain"}
	file, err := os.Open(os.Getenv("HOME") + "/.draupnir")
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			err = StoreConfig(config)
			return config, err
		}
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	return config, err
}

func StoreConfig(config Config) error {
	file, err := os.Create(os.Getenv("HOME") + "/.draupnir")
	if err != nil {
		return err
	}
	err = json.NewEncoder(file).Encode(config)
	return err
}

func main() {
	CONFIG, err := LoadConfig()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	client := client.Client{URL: "http://" + CONFIG.Domain, AccessToken: CONFIG.AccessToken}

	app := cli.NewApp()
	app.Name = "draupnir"
	app.Usage = "A client for draupnir"
	app.Commands = []cli.Command{
		{
			Name:    "config",
			Aliases: []string{},
			Usage:   "show the current configuration",
			Action: func(c *cli.Context) error {
				fmt.Printf("%+v\n", CONFIG)
				return nil
			},
		},
		{
			Name:    "authenticate",
			Aliases: []string{},
			Usage:   "authenticate with google",
			Action: func(c *cli.Context) error {
				token, err := authorise(clientID, clientSecret)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}

				CONFIG.AccessToken = token.AccessToken
				err = StoreConfig(CONFIG)
				if err != nil {
					fmt.Printf("error: %s\n", err)
					return err
				}
				fmt.Println("Successfully authenticated.")
				return nil
			},
		},
		{
			Name:    "instances",
			Aliases: []string{},
			Usage:   "manage your instances",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list your instances",
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
					Name:  "create",
					Usage: "create a new instance",
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
					Name: "destroy",
					Usage: "destroy an instance",
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
			},
		},
		{
			Name:    "images",
			Aliases: []string{},
			Usage:   "manage images",
			Subcommands: []cli.Command{
				{
					Name:  "list",
					Usage: "list available images",
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
					Name:  "destroy",
					Usage: "destroy an image",
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

				fmt.Printf("export PGHOST=%s PGPORT=%d PGUSER=postgres\n", CONFIG.Domain, instance.Port)

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
