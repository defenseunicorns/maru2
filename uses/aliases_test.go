// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"

	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestConfigBasedResolver(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		inputType       string
		inputQualifiers map[string]string
		aliases         v1.AliasMap
		wantType        string
		wantQualifiers  map[string]string
		wantResolved    bool
	}{
		{
			name:            "no alias",
			inputType:       packageurl.TypeGithub,
			inputQualifiers: map[string]string{},
			aliases:         v1.AliasMap{},
			wantType:        packageurl.TypeGithub,
			wantQualifiers:  map[string]string{},
			wantResolved:    false,
		},
		{
			name:            "simple alias",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliases: v1.AliasMap{
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
			aliases: v1.AliasMap{
				"gl": {
					Type:    packageurl.TypeGitlab,
					BaseURL: "https://gitlab.example.com",
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
			aliases: v1.AliasMap{
				"gl": {
					Type:    packageurl.TypeGitlab,
					BaseURL: "https://gitlab.example.com",
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
			aliases: v1.AliasMap{
				"another": {
					Type:         packageurl.TypeGithub,
					TokenFromEnv: "GITHUB2_TOKEN",
				},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{QualifierTokenFromEnv: "GITHUB2_TOKEN"},
			wantResolved:   true,
		},
		{
			name:            "alias not found",
			inputType:       "nonexistent",
			inputQualifiers: map[string]string{},
			aliases: v1.AliasMap{
				"other": {
					Type: packageurl.TypeGithub,
				},
			},
			wantType:       "nonexistent",
			wantQualifiers: map[string]string{},
			wantResolved:   false,
		},
		{
			name:            "empty aliases map",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliases:         v1.AliasMap{},
			wantType:        "custom",
			wantQualifiers:  map[string]string{},
			wantResolved:    false,
		},
		{
			name:            "alias with only base URL",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliases: v1.AliasMap{
				"custom": {
					Type:    packageurl.TypeGitlab,
					BaseURL: "https://gitlab.example.com",
				},
			},
			wantType:       packageurl.TypeGitlab,
			wantQualifiers: map[string]string{QualifierBaseURL: "https://gitlab.example.com"},
			wantResolved:   true,
		},
		{
			name:            "alias with only token from env",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliases: v1.AliasMap{
				"custom": {
					Type:         packageurl.TypeGithub,
					TokenFromEnv: "CUSTOM_TOKEN",
				},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{QualifierTokenFromEnv: "CUSTOM_TOKEN"},
			wantResolved:   true,
		},
		{
			name:            "existing qualifiers preserved and merged",
			inputType:       "custom",
			inputQualifiers: map[string]string{"existing": "value", QualifierBaseURL: "override"},
			aliases: v1.AliasMap{
				"custom": {
					Type:         packageurl.TypeGithub,
					BaseURL:      "https://github.com",
					TokenFromEnv: "GITHUB_TOKEN",
				},
			},
			wantType: packageurl.TypeGithub,
			wantQualifiers: map[string]string{
				"existing":            "value",
				QualifierBaseURL:      "override", // existing value preserved
				QualifierTokenFromEnv: "GITHUB_TOKEN",
			},
			wantResolved: true,
		},
		{
			name:            "alias with path field ignored in resolution",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliases: v1.AliasMap{
				"custom": {
					Type:    packageurl.TypeGithub,
					BaseURL: "https://github.com",
					Path:    "this/should/be/ignored",
				},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{QualifierBaseURL: "https://github.com"},
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

			resolvedPURL, isResolved := ResolvePkgAlias(inputPURL, tt.aliases)

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
