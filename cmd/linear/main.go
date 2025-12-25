package main

import (
	"os"

	"github.com/juanbermudez/agent-linear-cli/internal/cmd"
)

// Version information (set at build time)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := cmd.NewRootCmd(version, commit, date)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
