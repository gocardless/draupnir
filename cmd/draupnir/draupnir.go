package main

import (
	"os"

	"github.com/gocardless/draupnir/pkg/cmd"
)

func main() {
	if err := cmd.Root().Execute(); err != nil {
		os.Exit(1)
	}
}
