// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"cmp"
	"slices"

	"github.com/invopop/jsonschema"

	"github.com/defenseunicorns/maru2/uses"
)

// DefaultTaskName is the default task name
const DefaultTaskName = "default"

// SchemaVersionV0 is the current schema version for workflows
const SchemaVersionV0 = "v0"

// Versioned is a tiny struct used to grab the schema version for a workflow
type Versioned struct {
	// SchemaVersion is the workflow schema that this workflow follows
	SchemaVersion string `json:"schema-version"`
}

// Workflow is a wrapper struct around the input map and task map
//
// It represents a "tasks.yaml" file
type Workflow struct {
	SchemaVersion string                `json:"schema-version"`
	Inputs        InputMap              `json:"inputs,omitempty"`
	Tasks         TaskMap               `json:"tasks,omitempty"`
	Aliases       map[string]uses.Alias `json:"aliases,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for a workflow
func (Workflow) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schemaVersion, ok := schema.Properties.Get("schema-version"); ok && schemaVersion != nil {
		schemaVersion.Description = "Workflow schema version"
		schemaVersion.Enum = []any{SchemaVersionV0}
		schemaVersion.AdditionalProperties = jsonschema.FalseSchema
	}

	if inputs, ok := schema.Properties.Get("inputs"); ok && inputs != nil {
		inputs.Description = "Input parameters for the workflow"
		inputs.PatternProperties = map[string]*jsonschema.Schema{
			InputNamePattern.String(): {
				Ref:         "#/$defs/InputParameter",
				Description: "Input parameter for the workflow",
			},
		}
		inputs.AdditionalProperties = jsonschema.FalseSchema
	}

	if tasks, ok := schema.Properties.Get("tasks"); ok && tasks != nil {
		tasks.Description = "Map of tasks where the key is the task name, the task named 'default' is called when no task is specified"
		tasks.PatternProperties = map[string]*jsonschema.Schema{
			TaskNamePattern.String(): {
				Ref:         "#/$defs/Task",
				Description: "A task definition, aka a collection of steps",
			},
		}

		tasks.AdditionalProperties = jsonschema.FalseSchema
	}

	if aliases, ok := schema.Properties.Get("aliases"); ok && aliases != nil {
		aliases.Description = `Aliases for package URLs to create shorthand references

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#package-url-aliases`
		aliases.PatternProperties = map[string]*jsonschema.Schema{
			// TODO: figure out if there is a better pattern to use here
			InputNamePattern.String(): {
				Ref:         "#/$defs/Alias",
				Description: "An alias to a package URL",
			},
		}
		aliases.AdditionalProperties = jsonschema.FalseSchema
	}
}

// Task is a list of steps
type Task []Step

// TaskMap is a map of tasks, where the key is the task name
type TaskMap map[string]Task

// Find returns a task by name
func (tm TaskMap) Find(call string) (Task, bool) {
	task, ok := tm[call]
	return task, ok
}

// OrderedTaskNames returns a list of task names in alphabetical order
//
// The default task is always first
func (tm TaskMap) OrderedTaskNames() []string {
	names := make([]string, 0, len(tm))
	for k := range tm {
		names = append(names, k)
	}
	slices.SortStableFunc(names, func(a, b string) int {
		if a == DefaultTaskName {
			return -1
		}
		if b == DefaultTaskName {
			return 1
		}
		return cmp.Compare(a, b)
	})
	return names
}

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json"

	return schema
}
