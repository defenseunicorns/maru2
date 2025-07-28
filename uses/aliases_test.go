// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigBasedResolver(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		inputType       string
		inputQualifiers map[string]string
		aliases         map[string]Alias
		wantType        string
		wantQualifiers  map[string]string
		wantResolved    bool
	}{
		{
			name:            "no alias",
			inputType:       packageurl.TypeGithub,
			inputQualifiers: map[string]string{},
			aliases:         map[string]Alias{},
			wantType:        packageurl.TypeGithub,
			wantQualifiers:  map[string]string{},
			wantResolved:    false,
		},
		{
			name:            "simple alias",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliases: map[string]Alias{
				"custom": {
					Type: packageurl.TypeGithub,
				},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{},
			wantResolved:   true,
		},
		{
			name:            "alias with base",
			inputType:       "gl",
			inputQualifiers: map[string]string{},
			aliases: map[string]Alias{
				"gl": {
					Type: packageurl.TypeGitlab,
					Base: "https://gitlab.example.com",
				},
			},
			wantType:       packageurl.TypeGitlab,
			wantQualifiers: map[string]string{QualifierBaseURL: "https://gitlab.example.com"},
			wantResolved:   true,
		},
		{
			name:            "alias with overridden base",
			inputType:       "gl",
			inputQualifiers: map[string]string{QualifierBaseURL: "https://my-gitlab.com"},
			aliases: map[string]Alias{
				"gl": {
					Type: packageurl.TypeGitlab,
					Base: "https://gitlab.example.com",
				},
			},
			wantType:       packageurl.TypeGitlab,
			wantQualifiers: map[string]string{QualifierBaseURL: "https://my-gitlab.com"},
			wantResolved:   true,
		},
		{
			name:            "alias with token from env",
			inputType:       "another",
			inputQualifiers: map[string]string{},
			aliases: map[string]Alias{
				"another": {
					Type:         packageurl.TypeGithub,
					TokenFromEnv: "GITHUB2_TOKEN",
				},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{QualifierTokenFromEnv: "GITHUB2_TOKEN"},
			wantResolved:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			qualifiers := packageurl.QualifiersFromMap(tt.inputQualifiers)
			inputPURL := packageurl.PackageURL{
				Type:       tt.inputType,
				Namespace:  "test",
				Name:       "repo",
				Version:    DefaultVersion,
				Qualifiers: qualifiers,
				Subpath:    "path/to/file.yaml",
			}

			resolvedPURL, isResolved := ResolveAlias(inputPURL, tt.aliases)

			assert.Equal(t, tt.wantResolved, isResolved)
			assert.Equal(t, tt.wantType, resolvedPURL.Type)

			resolvedQualifiers := resolvedPURL.Qualifiers.Map()
			assert.Equal(t, tt.wantQualifiers, resolvedQualifiers)

			assert.Equal(t, inputPURL.Namespace, resolvedPURL.Namespace)
			assert.Equal(t, inputPURL.Name, resolvedPURL.Name)
			assert.Equal(t, inputPURL.Version, resolvedPURL.Version)
			assert.Equal(t, inputPURL.Subpath, resolvedPURL.Subpath)
		})
	}
}

func TestAliasSchema(t *testing.T) {
	t.Parallel()
	f, err := os.Open("../maru2.schema.json")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	curr := schema["$defs"].(map[string]any)["Alias"]
	b, err := json.Marshal(curr)
	require.NoError(t, err)

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	aliasSchema := reflector.Reflect(&Alias{})
	aliasSchema.Version = ""
	aliasSchema.ID = ""
	b2, err := json.Marshal(aliasSchema)
	require.NoError(t, err)

	assert.JSONEq(t, string(b), string(b2))
}
