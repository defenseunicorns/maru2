// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main provides the entry point for the application.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/defenseunicorns/maru2"
)

func run(root string) error {
	schema := maru2.WorkFlowSchema()

	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(root, "maru2.schema.json"), b, 0644)
}

// main is the entry point for the application
func main() {
	// usage: `go run gen/main.go`
	if err := run(""); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
