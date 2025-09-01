// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"

	maru2cmd "github.com/defenseunicorns/maru2/cmd"
)

func main() {
	ctx := context.Background()
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		Level:           log.DebugLevel,
	})
	logger.SetStyles(maru2cmd.DefaultStyles())
	ctx = log.WithContext(ctx, logger)

	if err := maru2cmd.NewMigrateCmd().ExecuteContext(ctx); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}
