// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application
package main

import (
	"os"

	"github.com/defenseunicorns/maru2/cmd"
)

func main() {
	code := cmd.Main()
	os.Exit(code)
}
