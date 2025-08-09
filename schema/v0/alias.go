// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
	"github.com/invopop/jsonschema"
	"github.com/package-url/packageurl-go"
)

// Alias defines how an alias should be resolved
type Alias struct {
	Type         string `json:"type"`
	Base         string `json:"base,omitempty"`
	TokenFromEnv string `json:"token-from-env,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for an alias
func (Alias) JSONSchemaExtend(schema *jsonschema.Schema) {
	if typ, ok := schema.Properties.Get("type"); ok && typ != nil {
		typ.Description = "Type of the alias, maps to a package URL type"
		typ.Enum = []any{packageurl.TypeGithub, packageurl.TypeGitlab}
	}

	if base, ok := schema.Properties.Get("base"); ok && base != nil {
		base.Description = "Base URL for the underlying client (e.g. https://mygitlab.com )"
	}

	if tokenFromEnv, ok := schema.Properties.Get("token-from-env"); ok && tokenFromEnv != nil {
		tokenFromEnv.Description = "Environment variable containing the token for authentication"
		tokenFromEnv.Pattern = "^[a-zA-Z_]+[a-zA-Z0-9_]*$" // EnvVariablePattern.String(), a little bit of copying never hurt anyone
	}
}
