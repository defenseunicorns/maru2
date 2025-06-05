// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/defenseunicorns/maru2"
)

func main() {
	schema := maru2.WorkFlowSchema()

	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, string(b))
}
