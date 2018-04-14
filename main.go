package main

import (
	"os"

	"github.com/gocardless/draupnir/server"
	"github.com/gocardless/draupnir/version"
	"github.com/prometheus/common/log"
	"github.com/urfave/cli"
)

func main() {
	logger := log.With("app", "draupnir")

	app := cli.NewApp()
	app.Name = "draupnir"
	app.Version = version.Version
	app.Usage = ""

	app.Commands = []cli.Command{
		{
			Name:  "server",
			Usage: "start the draupnir server",
			Action: func(c *cli.Context) error {
				server.Run(logger)
				// TODO: maybe return fatal errors from Run?
				return nil
			},
		},
	}

	app.Run(os.Args)
}
