package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gocardless/draupnir/cmd/util"
)

func main() {
	var (
		id    string
		root  string
		pgctl string
	)

	flag.StringVar(&id, "id", "", "ID of the image")
	flag.StringVar(&root, "root", "", "Path to the root of the image directory")
	flag.StringVar(&pgctl, "pgctl", "", "Path to the pg_ctl executable")

	flag.Parse()

	if root == "" {
		fmt.Printf("Must provide --root flag\n")
		os.Exit(1)
	}

	if id == "" {
		fmt.Printf("Must provide --id flag\n")
		os.Exit(1)
	}

	if pgctl == "" {
		fmt.Printf("Must provide --pgctl flag\n")
		os.Exit(1)
	}
}

func finaliseImage(root string, id string, pgctl string) {
	imagePath := filepath.Join(root, "image_uploads", id)
	snapshotPath := filepath.Join(root, "image_snapshots", id)

	var (
		output []byte
		err    error
	)

	pguid, err := util.GetUID("postgres")
	if err != nil {
		log.Fatal(err.Error())
	}

	output, err = util.Execute(0, "chown", "-R", "postgres", imagePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)

	output, err = util.Execute(0, "chmod", "700", imagePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)

	// TODO: copy in config files

	// delete postmaster files
	output, err = util.Execute(0, "rm", "-f", filepath.Join(imagePath, "postmaster.*"))
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)

	// start postgres
	portArg := fmt.Sprintf("\"-p %d\"", util.RandomPort())
	output, err = util.Execute(
		pguid,
		pgctl, "-D", imagePath, "-o", portArg, "-l", "/dev/null", "start",
	)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)

	// TODO: run anonymisation function

	// stop postgres
	output, err = util.Execute(pguid, pgctl, "-D", imagePath, "stop")
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)

	// create snapshot
	output, err = util.Execute(0, "btrfs", "subvolume", "snapshot", imagePath, snapshotPath)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%s", output)
}
