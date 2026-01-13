// Staghorn - A shared team layer for Claude Code
package main

import (
	"os"

	"github.com/HartBrook/staghorn/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
