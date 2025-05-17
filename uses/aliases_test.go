// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"testing"
	"testing/fstest"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/require"
)

func TestConfigBasedResolver(t *testing.T) {
	tests := []struct {
		name            string
		inputType       string
		inputQualifiers map[string]string
		aliasConfig     *AliasConfig
		wantType        string
		wantQualifiers  map[string]string
		wantResolved    bool
	}{
		{
			name:            "no alias",
			inputType:       "github",
			inputQualifiers: map[string]string{},
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{},
			},
			wantType:       "github",
			wantQualifiers: map[string]string{},
			wantResolved:   false,
		},
		{
			name:            "simple alias",
			inputType:       "custom",
			inputQualifiers: map[string]string{},
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{
					"custom": {
						Type: "github",
					},
				},
			},
			wantType:       "github",
			wantQualifiers: map[string]string{},
			wantResolved:   true,
		},
		{
			name:            "alias with base",
			inputType:       "gl",
			inputQualifiers: map[string]string{},
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{
					"gl": {
						Type: "gitlab",
						Base: "https://gitlab.example.com",
					},
				},
			},
			wantType:       "gitlab",
			wantQualifiers: map[string]string{"base": "https://gitlab.example.com"},
			wantResolved:   true,
		},
		{
			name:            "alias with overridden base",
			inputType:       "gl",
			inputQualifiers: map[string]string{"base": "https://my-gitlab.com"},
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{
					"gl": {
						Type: "gitlab",
						Base: "https://gitlab.example.com",
					},
				},
			},
			wantType:       "gitlab",
			wantQualifiers: map[string]string{"base": "https://my-gitlab.com"},
			wantResolved:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewConfigBasedResolver(tt.aliasConfig)

			qualifiers := packageurl.QualifiersFromMap(tt.inputQualifiers)
			inputPURL := packageurl.PackageURL{
				Type:       tt.inputType,
				Namespace:  "test",
				Name:       "repo",
				Version:    "main",
				Qualifiers: qualifiers,
				Subpath:    "path/to/file.yaml",
			}

			resolvedPURL, isResolved := resolver.ResolveAlias(inputPURL)

			require.Equal(t, tt.wantResolved, isResolved)
			require.Equal(t, tt.wantType, resolvedPURL.Type)

			resolvedQualifiers := resolvedPURL.Qualifiers.Map()
			require.Equal(t, tt.wantQualifiers, resolvedQualifiers)

			require.Equal(t, inputPURL.Namespace, resolvedPURL.Namespace)
			require.Equal(t, inputPURL.Name, resolvedPURL.Name)
			require.Equal(t, inputPURL.Version, resolvedPURL.Version)
			require.Equal(t, inputPURL.Subpath, resolvedPURL.Subpath)
		})
	}
}

func TestFileSystemConfigLoader(t *testing.T) {
	configContent := `aliases:
  gl:
    type: gitlab
    base: https://gitlab.example.com
  gh:
    type: github
`
	fsys := fstest.MapFS{
		"etc/maru2/aliases.yaml": &fstest.MapFile{
			Data: []byte(configContent),
			Mode: 0644,
		},
	}

	loader := NewFileSystemConfigLoader(fsys, "etc/maru2/aliases.yaml")
	config, err := loader.LoadConfig()
	require.NoError(t, err)

	require.Len(t, config.Aliases, 2)

	glAlias, ok := config.Aliases["gl"]
	require.True(t, ok)
	require.Equal(t, "gitlab", glAlias.Type)
	require.Equal(t, "https://gitlab.example.com", glAlias.Base)

	ghAlias, ok := config.Aliases["gh"]
	require.True(t, ok)
	require.Equal(t, "github", ghAlias.Type)
	require.Empty(t, ghAlias.Base)

	loader = NewFileSystemConfigLoader(fsys, "nonexistent-file.yaml")
	config, err = loader.LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	require.Empty(t, config.Aliases)
}

func TestConfigLoaderWithInvalidYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"invalid.yaml": &fstest.MapFile{
			Data: []byte(`invalid: yaml: content`),
			Mode: 0644,
		},
	}

	loader := NewFileSystemConfigLoader(fsys, "invalid.yaml")
	_, err := loader.LoadConfig()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse alias config file")
}
