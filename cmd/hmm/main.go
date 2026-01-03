// Package main is the entry point for the HMM CLI.
package main

import (
	"os"

	"github.com/f3rmion/hmm/cmd/hmm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
