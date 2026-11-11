package main

import (
	"fmt"
	"os"
)

var (
	version   = "v0.1.0"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
