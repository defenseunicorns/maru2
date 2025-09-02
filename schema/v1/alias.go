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

// Alias defines how an alias should be resolved
type Alias struct {
	Type         string `json:"type"`
	Base         string `json:"base,omitempty"`
	TokenFromEnv string `json:"token-from-env,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for an alias
func (Alias) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Description = "An alias to a package URL"

	if typ, ok := schema.Properties.Get("type"); ok && typ != nil {
		typ.Description = "Type of the alias, maps to a package URL type"
		typ.Enum = []any{packageurl.TypeGithub, packageurl.TypeGitlab}
	}

	if base, ok := schema.Properties.Get("base"); ok && base != nil {
		base.Description = "Base URL for the underlying client (e.g. https://mygitlab.com )"
	}

	if tokenFromEnv, ok := schema.Properties.Get("token-from-env"); ok && tokenFromEnv != nil {
		tokenFromEnv.Description = "Environment variable containing the token for authentication"
		tokenFromEnv.Pattern = EnvVariablePattern.String()
	}
}
