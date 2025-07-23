// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	maru2cmd "github.com/defenseunicorns/maru2/cmd"
)

// this application is only for internal testing only
// in order to replicate / dogfood the user experience of
// embedding maru2 as a sub-command
//
// the following is the preferred minimal amount of setup needed
func main() {
	internalRoot := &cobra.Command{
		Use: "internal",
	}

	// small cobra wrapper
	wrap := &cobra.Command{
		Use:                "run",
		SilenceErrors:      true,
		SilenceUsage:       true,
		DisableFlagParsing: true,
		Run: func(_ *cobra.Command, _ []string) {
			os.Args = os.Args[1:]
			code := maru2cmd.Main()
			os.Exit(code)
		},
	}

	internalRoot.AddCommand(wrap)

	// multi-call binary w/ wrapper
	switch filepath.Base(os.Args[0]) {
	case "maru2":
		os.Exit(maru2cmd.Main())
	default:
		internalRoot.Execute()
	}
}
