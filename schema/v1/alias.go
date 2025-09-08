// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"github.com/invopop/jsonschema"
	"github.com/package-url/packageurl-go"
)

// AliasMap is a map of aliases
type AliasMap map[string]Alias

// JSONSchemaExtend extends the JSON schema for an alias map
func (AliasMap) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.PropertyNames = &jsonschema.Schema{
		// TODO: figure out if there is a better pattern to use here
		Pattern: InputNamePattern.String(),
	}
}

// Alias defines the structure of a maru2 uses alias
//
// Using the JSON schema, one of type or path is required and mutually exclusive
type Alias struct {
	Type         string `json:"type,omitempty"`
	BaseURL      string `json:"base-url,omitempty"`
	TokenFromEnv string `json:"token-from-env,omitempty"`
	Path         string `json:"path,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for an alias
func (Alias) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Description = "An alias to a package URL or a local file path"

	schema.Properties = nil
	schema.Required = nil
	schema.AdditionalProperties = nil

	var one uint64 = 1

	localProps := jsonschema.NewProperties()
	localProps.Set("path", &jsonschema.Schema{
		Type:        "string",
		Description: "Relative path to workflow",
		MinLength:   &one,
	})

	remoteProps := jsonschema.NewProperties()
	remoteProps.Set("type", &jsonschema.Schema{
		Type: "string",
		Description: `Package URL type:

scheme:type/namespace/name@version?qualifiers#subpath

https://github.com/package-url/purl-spec#purl`,
		Enum: []any{packageurl.TypeGithub, packageurl.TypeGitlab},
	})
	remoteProps.Set("base-url", &jsonschema.Schema{
		Type:        "string",
		Description: "Base URL for the underlying client (e.g. https://mygitlab.com )",
	})
	remoteProps.Set("token-from-env", &jsonschema.Schema{
		Type:        "string",
		Description: "Environment variable containing the token for authentication",
		Pattern:     EnvVariablePattern.String(),
	})

	schema.OneOf = []*jsonschema.Schema{
		{
			// Local file alias - only path is allowed
			Type:                 "object",
			Description:          "Local file alias",
			Properties:           localProps,
			Required:             []string{"path"},
			AdditionalProperties: jsonschema.FalseSchema,
		},
		{
			// Remote alias - type is required, path is not allowed
			Type:                 "object",
			Description:          "Package URL alias (GitHub, GitLab, etc.) https://github.com/package-url/purl-spec#purl",
			Properties:           remoteProps,
			Required:             []string{"type"},
			AdditionalProperties: jsonschema.FalseSchema,
		},
	}
}
