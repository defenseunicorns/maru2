// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
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

	root.AddCommand(cli)

	// standard setup for context and logger
	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	var logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())
	// end context and logger setup

	// run the root, handle the errors
	// ExecuteContextC is the most preferred method
	cmd, err := root.ExecuteContextC(ctx)
	if err != nil {
		// the below is a copy-paste from maru2, as the formatting of
		// logging maru2's final error is left up to implementation
		// this is what users will see if they use maru2 as a standalone CLI

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
