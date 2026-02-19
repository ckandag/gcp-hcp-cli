package main

import (
	"os"

	gcphcpcli "github.com/ckandag/gcp-hcp-cli/pkg/cli"
)

func main() {
	if err := gcphcpcli.Execute(); err != nil {
		os.Exit(1)
	}
}
