// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
	"github.com/invopop/jsonschema"
)

// SchemaVersion is the current schema version for workflows
const SchemaVersion = "v0"

// Workflow is a wrapper struct around the input map and task map
//
// It represents a "tasks.yaml" file
type Workflow struct {
	SchemaVersion string           `json:"schema-version"`
	Inputs        InputMap         `json:"inputs,omitempty"`
	Tasks         TaskMap          `json:"tasks,omitempty"`
	Aliases       map[string]Alias `json:"aliases,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for a workflow
func (Workflow) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schemaVersion, ok := schema.Properties.Get("schema-version"); ok && schemaVersion != nil {
		schemaVersion.Description = "Workflow schema version. For v0 breaking changes can be expected without any migration pathway."
		schemaVersion.Enum = []any{SchemaVersion}
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

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/schema/v0/schema.json"

	return schema
}
