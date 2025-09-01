// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"github.com/invopop/jsonschema"
)

// SchemaVersion is the current schema version for workflows
const SchemaVersion = "v1"

// Workflow represents a "tasks.yaml" file
type Workflow struct {
	SchemaVersion string   `json:"schema-version"`
	Aliases       AliasMap `json:"aliases,omitempty"`
	Tasks         TaskMap  `json:"tasks,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for a workflow
func (Workflow) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schemaVersion, ok := schema.Properties.Get("schema-version"); ok && schemaVersion != nil {
		schemaVersion.Description = "Workflow schema version."
		schemaVersion.Enum = []any{SchemaVersion}
		schemaVersion.AdditionalProperties = jsonschema.FalseSchema
	}
	if tasks, ok := schema.Properties.Get("tasks"); ok && tasks != nil {
		tasks.Description = "Map of tasks where the key is the task name, the task named 'default' is called when no task is specified"
	}
	if aliases, ok := schema.Properties.Get("aliases"); ok && aliases != nil {
		aliases.Description = `Aliases for package URLs to create shorthand references
See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#package-url-aliases`
	}
}

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/schema/v1/schema.json"

	return schema
}
