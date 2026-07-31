package main

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/cli"
)

func main() {
	if len(os.Args) <= 1 {
		runTUI()
		return
	}

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
