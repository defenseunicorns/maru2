// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"testing"

	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
)

func TestConfigBasedResolver(t *testing.T) {
	tests := []struct {
		name            string
		inputType       string
		inputQualifiers map[string]string
		aliasConfig     *config.Config
		wantType        string
		wantQualifiers  map[string]string
		wantResolved    bool
	}{
		{
			name:            "no alias",
			inputType:       packageurl.TypeGithub,
			inputQualifiers: map[string]string{},
			aliasConfig: &config.Config{
				Aliases: map[string]config.Alias{},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{},
			wantResolved:   false,
		},
		{
			name:            "simple alias",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliasConfig: &config.Config{
				Aliases: map[string]config.Alias{
					"custom": {
						Type: packageurl.TypeGithub,
					},
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
			aliasConfig: &config.Config{
				Aliases: map[string]config.Alias{
					"gl": {
						Type: packageurl.TypeGitlab,
						Base: "https://gitlab.example.com",
					},
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
			aliasConfig: &config.Config{
				Aliases: map[string]config.Alias{
					"gl": {
						Type: packageurl.TypeGitlab,
						Base: "https://gitlab.example.com",
					},
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
			aliasConfig: &config.Config{
				Aliases: map[string]config.Alias{
					"another": {
						Type:         packageurl.TypeGithub,
						TokenFromEnv: "GITHUB2_TOKEN",
					},
				},
			},
			wantType:       packageurl.TypeGithub,
			wantQualifiers: map[string]string{QualifierTokenFromEnv: "GITHUB2_TOKEN"},
			wantResolved:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewConfigBasedPackageAliasMapper(tt.aliasConfig)

			qualifiers := packageurl.QualifiersFromMap(tt.inputQualifiers)
			inputPURL := packageurl.PackageURL{
				Type:       tt.inputType,
				Namespace:  "test",
				Name:       "repo",
				Version:    DefaultVersion,
				Qualifiers: qualifiers,
				Subpath:    "path/to/file.yaml",
			}

			resolvedPURL, isResolved := resolver.ResolveAlias(inputPURL)

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
