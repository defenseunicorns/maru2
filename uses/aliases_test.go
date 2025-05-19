// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
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
			inputType:       packageurl.TypeGithub,
			inputQualifiers: map[string]string{},
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{},
			},
			wantType:       packageurl.TypeGithub,
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
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{
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
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{
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
			aliasConfig: &AliasConfig{
				Aliases: map[string]AliasDefinition{
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
			resolver := NewConfigBasedResolver(tt.aliasConfig)

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
  another:
    type: github
    token-from-env: GITHUB_TOKEN
`

	cfg := &AliasConfig{
		Aliases: map[string]AliasDefinition{
			"gl": {
				Type: packageurl.TypeGitlab,
				Base: "https://gitlab.example.com",
			},
			"gh": {
				Type: packageurl.TypeGithub,
			},
			"another": {
				Type:         packageurl.TypeGithub,
				TokenFromEnv: "GITHUB_TOKEN",
			},
		},
	}

	fsys := afero.NewMemMapFs()
	err := afero.WriteFile(fsys, "etc/maru2/aliases.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewFileSystemConfigLoader(fsys, "etc/maru2/aliases.yaml")
	config, err := loader.LoadConfig()
	require.NoError(t, err)

	require.Len(t, config.Aliases, 3)

	glAlias, ok := config.Aliases["gl"]
	require.True(t, ok)
	require.Equal(t, packageurl.TypeGitlab, glAlias.Type)
	require.Equal(t, "https://gitlab.example.com", glAlias.Base)

	ghAlias, ok := config.Aliases["gh"]
	require.True(t, ok)
	require.Equal(t, packageurl.TypeGithub, ghAlias.Type)
	require.Empty(t, ghAlias.Base)

	loader = NewFileSystemConfigLoader(fsys, "nonexistent-file.yaml")
	config, err = loader.LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	require.Empty(t, config.Aliases)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Setenv("HOME", "")
		loader, err = DefaultConfigLoader()
		require.Nil(t, loader)
		require.EqualError(t, err, "$HOME is not defined")

		tmpDir := t.TempDir()
		err = os.Mkdir(filepath.Join(tmpDir, ".maru2"), 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".maru2", "aliases.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		t.Setenv("HOME", tmpDir)
		loader, err = DefaultConfigLoader()
		require.NoError(t, err)
		config, err = loader.LoadConfig()
		require.NoError(t, err)
		require.Equal(t, cfg.Aliases, config.Aliases)
	}
}

func TestConfigLoaderWithInvalidYAML(t *testing.T) {
	fsys := afero.NewMemMapFs()
	err := afero.WriteFile(fsys, "invalid.yaml", []byte(`invalid: yaml: content`), 0644)
	require.NoError(t, err)

	loader := NewFileSystemConfigLoader(fsys, "invalid.yaml")
	_, err = loader.LoadConfig()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse alias config file")
}
