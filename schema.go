// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"github.com/invopop/jsonschema"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

// WorkflowSchema generates the schema for either a given version, or all versions in one meta schema
func WorkflowSchema(version string) *jsonschema.Schema {
	var schema *jsonschema.Schema

	switch version {
	case v0.SchemaVersion:
		schema = v0.WorkFlowSchema()
	case v1.SchemaVersion:
		schema = v1.WorkFlowSchema()
	default:
		schema = &jsonschema.Schema{
			If: &jsonschema.Schema{
				Properties: jsonschema.NewProperties(),
			},
			Then: v1.WorkFlowSchema(),
			Else: &jsonschema.Schema{
				If: &jsonschema.Schema{
					Properties: jsonschema.NewProperties(),
				},
			},
			ID:      "https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json",
			Version: jsonschema.Version,
		}

		schema.If.Properties.Set("schema-version", &jsonschema.Schema{
			Type: "string",
			Enum: []any{v1.SchemaVersion},
		})

		schema.Else.If.Properties.Set("schema-version", &jsonschema.Schema{
			Type: "string",
			Enum: []any{v0.SchemaVersion},
		})
		schema.Else.Then = v0.WorkFlowSchema()
	}

	return schema
}
