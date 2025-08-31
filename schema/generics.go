// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package schema provides the workfow types and schema for maru2
package schema

// Versioned is a tiny struct used to grab the schema version for a workflow
type Versioned struct {
	// SchemaVersion is the workflow schema that this workflow follows
	SchemaVersion string `json:"schema-version"`
}

// With is a map of string keys and WithEntry values used to pass parameters to called tasks and within steps
type With = map[string]any

// Env is a map of environment variable names to values
type Env = map[string]any
