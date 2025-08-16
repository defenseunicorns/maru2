// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"

	"github.com/defenseunicorns/maru2"
	maru2cmd "github.com/defenseunicorns/maru2/cmd"
)

// this application is only for internal testing only
// in order to replicate / dogfood the user experience of
// embedding maru2 as a sub-command
//
// the following is the preferred minimal amount of setup needed
func main() {
	root := &cobra.Command{
		Use: "internal",
	}

	cli := maru2cmd.NewRootCmd()
	// rename maru2 -> run (or w/e you wish to call the command)
	cli.Use = "run"
	cli.Aliases = []string{"maru2", "r"}

	root.AddCommand(cli)

	// standard setup for context and logger
	// customize as you see fit
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM)
	defer cancel()

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())
	// end context and logger setup

	// register uds, zarf, kubectl and other shortcuts
	executablePath, err := getFinalExecutablePath()
	if err != nil {
		logger.Fatal(err)
	}

	maru2.RegisterWhichShortcut("uds", executablePath)
	maru2.RegisterWhichShortcut("zarf", executablePath+" zarf")
	maru2.RegisterWhichShortcut("kubectl", executablePath+" zarf tools kubectl")
	// end registration

	// run the root, handle the errors
	// ExecuteContextC is the most preferred method
	cmd, err := root.ExecuteContextC(ctx)
	if err != nil {
		// the below is a copy-paste from maru2, as the formatting of
		// logging maru2's final error is left up to implementation
		//
		// the below is what users will see if they use maru2 as a standalone CLI

		logger.Print("")

		if errors.Is(cmd.Context().Err(), context.DeadlineExceeded) {
			logger.Error("task timed out")
		}

		var tErr *maru2.TraceError
		if errors.As(err, &tErr) && len(tErr.Trace) > 0 {
			trace := tErr.Trace
			slices.Reverse(trace)
			if len(trace) == 1 {
				logger.Error(tErr)
				logger.Error(trace[0])
			} else {
				logger.Error(tErr, "traceback (most recent call first)", strings.Join(trace, "\n"))
			}
		} else {
			logger.Error(err)
		}
	}

	// calculate the exit code from the CLI execution
	code := maru2cmd.ParseExitCode(err)
	os.Exit(code)
}

// getFinalExecutablePath returns the absolute path to the current executable, following any symlinks along the way.
//
// copied from https://github.com/defenseunicorns/pkg/blob/main/exec/utils.go while I figure out if I want to use this lib
func getFinalExecutablePath() (string, error) {
	binaryPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// In case the binary is symlinked somewhere else, get the final destination
	return filepath.EvalSymlinks(binaryPath)
}
