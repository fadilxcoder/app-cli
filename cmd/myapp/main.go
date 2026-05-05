package main

import (
	"fmt"
	"os"

	"github.com/fadilxcoder/app-cli/internal/cli"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

func main() {
	if err := cli.Execute(Version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
