// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main is the entry point for the application.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

func main() {
	var schema *jsonschema.Schema

	version := ""
	if len(os.Args) > 1 {
		version = os.Args[1]
	}

	switch version {
	case v0.SchemaVersion:
		schema = v0.WorkFlowSchema()
	default:
		schema = &jsonschema.Schema{
			If: &jsonschema.Schema{
				Properties: jsonschema.NewProperties(),
			},
			Then: &jsonschema.Schema{
				Properties: jsonschema.NewProperties(),
			},
			ID:      "https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json",
			Version: jsonschema.Version,
		}

		schema.If.Properties.Set("schema-version", &jsonschema.Schema{
			Type: "string",
			Enum: []any{v0.SchemaVersion},
		})

		schema.Then = v0.WorkFlowSchema()
	}

	b, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, string(b))
}
