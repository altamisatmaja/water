package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "water",
	Short:   "Visual brain of MCP agents",
	Long:    "Water captures and visualizes agent events into a local knowledge graph.",
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
