// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/log"

	maru2cmd "github.com/defenseunicorns/maru2/cmd"
)

func main() {
	code := Main()
	os.Exit(code)
}

// Main executes the root command for the maru2-publish CLI.
//
// It returns 0 on success, 1 on failure and logs any errors.
func Main() int {
	cli := maru2cmd.NewPublishCmd()

	ctx := context.Background()

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())

	ctx = log.WithContext(ctx, logger)

	if err := cli.ExecuteContext(ctx); err != nil {
		logger.Error(err)
		return 1
	}
	return 0
}
