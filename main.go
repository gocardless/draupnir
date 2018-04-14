package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gocardless/draupnir/client/config"
	"github.com/gocardless/draupnir/models"
	"github.com/gocardless/draupnir/server"
	clientPkg "github.com/gocardless/draupnir/server/api/client"
	"github.com/gocardless/draupnir/version"
	"github.com/prometheus/common/log"
	"github.com/urfave/cli"
)

const quickStart string = `
QUICK START:
		draupnir-client authenticate
		eval $(draupnir-client new)
		psql
`

func main() {
	logger := log.With("app", "draupnir")
	var err error

	app := cli.NewApp()
	app.Name = "draupnir"
	app.Version = version.Version
	app.Usage = "A client for draupnir"
	app.CustomAppHelpTemplate = fmt.Sprintf("%s%s", cli.AppHelpTemplate, quickStart)
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "insecure",
			Usage: "don't validate certificates when connecting to draupnir",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "server",
			Usage: "start the draupnir server",
			Action: func(c *cli.Context) error {
				err := server.Run(logger)
				if err != nil {
					logger.With("error", err.Error()).Fatal("Failed to start server")
				}
				return nil
			},
		},
		{
			Name:        "config",
			Aliases:     []string{},
			Usage:       "get and set config values",
			Description: "Get and set config values",
			Subcommands: []cli.Command{
				{
					Name:      "show",
					Usage:     "show the current configuration",
					UsageText: "draupnir config show",
					Action: func(c *cli.Context) error {
						cfg := loadConfig(logger)

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
				},
				{
					Name:  "set",
					Usage: "set a config value",
					UsageText: `draupnir config set [key] [value]

[key] can take the following values:
    domain: The domain of the draupnir server.
    database: The default database to connect to. If not set, defaults to the PGDATABASE environment variable.`,
					Action: func(c *cli.Context) error {
						if len(c.Args()) != 2 {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.Fatal("Invalid arguments")
						}
						key := c.Args().First()
						val := c.Args()[1]

						cfg := loadConfig(logger)
						switch strings.ToLower(key) {
						case "domain":
							cfg.Domain = val
							storeConfig(cfg, logger)
						case "database":
							cfg.Database = val
							storeConfig(cfg, logger)
						default:
							logger.With("key", key).Fatal("Invalid key")
						}
						return nil
					},
				},
			},
		},
		{
			Name:    "authenticate",
			Aliases: []string{},
			Usage:   "authenticate with google",
			Action: func(c *cli.Context) error {
				cfg := loadConfig(logger)
				client := NewClient(c, logger)

				if cfg.Token.RefreshToken != "" {
					logger.Info("You're already authenticated")
					return nil
				}

				state := fmt.Sprintf("%d", rand.Int31())

				url := fmt.Sprintf("https://%s/authenticate?state=%s", cfg.Domain, state)
				err := exec.Command("open", url).Run()
				if err != nil {
					fmt.Printf("Visit this link in your browser: %s\n", url)
				}

				token, err := client.CreateAccessToken(state)
				if err != nil {
					logger.With("error", err).Fatal("Could not create access token")
				}

				cfg.Token = token
				storeConfig(cfg, logger)

				logger.Info("Successfully authenticated.")
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
						client := NewClient(c, logger)

						instances, err := client.ListInstances()
						if err != nil {
							logger.With("error", err).Fatal("Could not fetch instances")
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
						var image models.Image
						client := NewClient(c, logger)

						if c.NArg() == 0 {
							image, err = client.GetLatestImage()
						} else {
							image, err = client.GetImage(c.Args().First())
						}

						if err != nil {
							logger.With("error", err).Fatal("Could not fetch image")
						}

						instance, err := client.CreateInstance(image)
						if err != nil {
							logger.With("error", err).Fatal("Could not create instance")
						}

						logger.With("id", instance.ID).With("image", image.ID).Info("Created instance")
						fmt.Println(InstanceToString(instance))
						return nil
					},
				},
				{
					Name:  "destroy",
					Usage: "destroy an instance",
					Action: func(c *cli.Context) error {
						id := c.Args().First()
						if id == "" {
							logger.Fatal("Must supply an instance id")
						}

						client := NewClient(c, logger)

						instance, err := client.GetInstance(id)
						if err != nil {
							logger.With("error", err).Fatal("Could not fetch instance")
						}

						err = client.DestroyInstance(instance)
						if err != nil {
							logger.With("error", err).Fatal("Could not destroy instance")
						}

						logger.With("id", instance.ID).Info("Destroyed instance")
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
						client := NewClient(c, logger)

						images, err := client.ListImages()

						if err != nil {
							logger.With("error", err).Fatal("Could not fetch images")
						}
						for _, image := range images {
							fmt.Println(ImageToString(image))
						}
						return nil
					},
				},
				{
					Name:  "create",
					Usage: "create a new image",
					UsageText: `draupnir images create [backedUpAt] [anon.sql]

[backedUpAt] an iso8601 timestamp defining when this backup was completed
[anonyimse.sql] path to an anonymisation script that will be run on image finalisation`,
					Action: func(c *cli.Context) error {
						var image models.Image
						client := NewClient(c, logger)

						if len(c.Args()) != 2 {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.Fatal("Invalid command arguments")
						}

						backedUpAt, err := time.Parse(time.RFC3339, c.Args().Get(0))
						if err != nil {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.Fatal("Invalid backedUpAt timestamp")
						}

						anonPath := c.Args().Get(1)
						anon, err := ioutil.ReadFile(anonPath)
						if err != nil {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.Fatal("Invalid anon script")
						}

						image, err = client.CreateImage(backedUpAt, anon)
						if err != nil {
							logger.With("error", err).Fatal("Could not create image")
						}

						fmt.Println(ImageToString(image))
						return nil
					},
				},
				{
					Name:  "finalise",
					Usage: "finalises an image (makes it ready)",
					UsageText: `draupnir images finalise [id]

[id] the image ID to finalise`,
					Action: func(c *cli.Context) error {
						var image models.Image
						client := NewClient(c, logger)

						if len(c.Args()) != 1 {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.Fatal("Invalid command arguments")
						}

						imageID, err := strconv.Atoi(c.Args().First())
						if err != nil {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.With("error", err).Fatal("Invalid image ID")
						}

						image, err = client.FinaliseImage(imageID)
						if err != nil {
							logger.With("error", err).Fatal("Could not finalise image")
						}

						fmt.Println(ImageToString(image))
						return nil
					},
				},
				{
					Name:  "destroy",
					Usage: "destroy an image",
					Action: func(c *cli.Context) error {
						id := c.Args().First()
						if id == "" {
							cli.ShowCommandHelp(c, c.Command.Name)
							logger.Fatal("Must supply an image id")
						}

						client := NewClient(c, logger)

						image, err := client.GetImage(id)
						if err != nil {
							logger.With("error", err).Fatal("Could not fetch image")
						}

						err = client.DestroyImage(image)
						if err != nil {
							logger.With("error", err).Fatal("Could not destroy image")
						}

						logger.With("id", image.ID).Info("Destroyed image")
						return nil
					},
				},
			},
		},
		{
			Name:  "env",
			Usage: "show the environment variables to connect to an instance",
			UsageText: `draupnir env [id]

[id] the instance ID to connect to`,
			Action: func(c *cli.Context) error {
				id := c.Args().First()
				if id == "" {
					cli.ShowCommandHelp(c, c.Command.Name)
					logger.Fatal("Must supply an instance id")
				}

				client := NewClient(c, logger)

				instance, err := client.GetInstance(id)
				if err != nil {
					logger.With("error", err).Fatal("Could not fetch instance")
				}

				showExportCommand(loadConfig(logger), instance)
				return nil
			},
		},
		{
			Name:    "new",
			Aliases: []string{},
			Usage:   "create a new instance",
			Action: func(c *cli.Context) error {
				client := NewClient(c, logger)

				image, err := client.GetLatestImage()
				if err != nil {
					logger.With("error", err).Fatal("Could not fetch image")
				}

				instance, err := client.CreateInstance(image)
				if err != nil {
					logger.With("error", err).Fatal("Could not create instance")
				}

				showExportCommand(loadConfig(logger), instance)
				return nil
			},
		},
	}

	app.Run(os.Args)
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

func ImageToString(i models.Image) string {
	return fmt.Sprintf("%2d [ %s - READY: %5t ]", i.ID, i.BackedUpAt.Format(time.RFC3339), i.Ready)
}

func InstanceToString(i models.Instance) string {
	return fmt.Sprintf("%2d [ PORT: %d - %s ]", i.ID, i.Port, i.CreatedAt.Format(time.RFC3339))
}

func loadConfig(logger log.Logger) config.Config {
	cfg, err := config.Load()
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not load configuration")
	}
	return cfg
}

func storeConfig(cfg config.Config, logger log.Logger) {
	err := config.Store(cfg)
	if err != nil {
		logger.With("error", err.Error()).Fatal("Could not store configuration")
	}
}

func NewClient(c *cli.Context, logger log.Logger) clientPkg.Client {
	cfg := loadConfig(logger)
	return clientPkg.NewClient(
		fmt.Sprintf("https://%s", cfg.Domain),
		cfg.Token,
		c.GlobalBool("insecure"),
	)
}
