package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gocardless/draupnir/cmd/util"
)

const usage = `
Butler
USAGE: butler IMAGE_PATH INSTANCE_PATH --pgctl PG_CTL_PATH --log LOG_PATH

Butler will:
- create a btrfs snapshot of IMAGE_PATH at INSTANCE_PATH
- start postgres via pg_ctl, logging to LOG_PATH

Exit codes:
1: invalid arguments provided
2: internal error
`

// Butler
// This tool will create an instance from an image and start postgres
// on that instance. It takes as arguments:
//   pgctl - the path to the pg_ctl tool
//   image-path - the path to the image
//   instance-path - the path where the instance should be created
func main() {
	var (
		pgctl   string
		logPath string
		output  []byte
		err     error
	)

	flag.StringVar(&pgctl, "pgctl", "", "Path to the pg_ctl executable")
	flag.StringVar(&logPath, "log", "", "Path to the log file for postgres")

	flag.Parse()

	if pgctl == "" || logPath == "" {
		fmt.Print(usage)
		os.Exit(1)
	}

	if len(flag.Args()) != 2 {
		fmt.Printf("Given %d args, exepected 2", len(flag.Args()))
	}

	imagePath := flag.Arg(0)
	instancePath := flag.Arg(1)

	// create instance snapshot
	output, err = util.Execute(0, "btrfs", "subvolume", "snapshot", imagePath, instancePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)

	pguid, err := util.GetUID("postgres")
	if err != nil {
		log.Fatal(err.Error())
	}

	// start postgres
	port := util.RandomPort()
	portArg := fmt.Sprintf("\"-p %d\"", port)

	log.Printf("Attempting to start postgres on port %d with uid %d", port, pguid)
	output, err = util.Execute(pguid,
		pgctl,
		"-D", instancePath,
		"-o", portArg,
		"-l", logPath,
		"start",
	)
	if err != nil {
		log.Fatalf("Failed to start postgres: %s", err.Error())
	}
	log.Printf("%s", output)
}
