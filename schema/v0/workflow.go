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
	Inputs        InputMap         `json:"inputs,omitempty"  jsonschema_description:"Input parameters for the workflow"`
	Tasks         TaskMap          `json:"tasks,omitempty"   jsonschema_description:"Map of tasks where the key is the task name, the task named 'default' is called when no task is specified"`
	Aliases       map[string]Alias `json:"aliases,omitempty" jsonschema_description:"Aliases for package URLs to create shorthand references\nSee https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#package-url-aliases"`
}

// JSONSchemaExtend extends the JSON schema for a workflow
func (Workflow) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schemaVersion, ok := schema.Properties.Get("schema-version"); ok && schemaVersion != nil {
		schemaVersion.Description = "Workflow schema version. For v0 breaking changes can be expected without any migration pathway."
		schemaVersion.Enum = []any{SchemaVersion}
		schemaVersion.AdditionalProperties = jsonschema.FalseSchema
	}
}

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{ExpandedStruct: true, DoNotReference: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/schema/v0/schema.json"

	return schema
}
