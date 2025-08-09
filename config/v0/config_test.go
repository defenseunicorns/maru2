// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/config"
	v0 "github.com/defenseunicorns/maru2/schema/v0"
	"github.com/defenseunicorns/maru2/uses"
)

func TestFileSystemConfigLoader(t *testing.T) {
	configContent := `schema-version: v0
aliases:
  gl:
    type: gitlab
    base: https://gitlab.example.com
  gh:
    type: github
  another:
    type: github
    token-from-env: GITHUB_TOKEN
`

	fsys := afero.NewMemMapFs()
	err := afero.WriteFile(fsys, "etc/maru2/config.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(afero.NewBasePathFs(fsys, "etc/maru2"))
	require.NoError(t, err)

	assert.Len(t, cfg.Aliases, 3)

	glAlias, ok := cfg.Aliases["gl"]
	assert.True(t, ok)
	assert.Equal(t, packageurl.TypeGitlab, glAlias.Type)
	assert.Equal(t, "https://gitlab.example.com", glAlias.Base)

	ghAlias, ok := cfg.Aliases["gh"]
	assert.True(t, ok)
	assert.Equal(t, packageurl.TypeGithub, ghAlias.Type)
	assert.Empty(t, ghAlias.Base)

	cfg, err = LoadConfig(afero.NewBasePathFs(fsys, "nonexistent-dir"))
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Empty(t, cfg.Aliases)

	t.Run("invalid config", func(t *testing.T) {
		fsys = afero.NewMemMapFs()
		err = afero.WriteFile(fsys, "invalid/config.yaml", []byte(`invalid: yaml: content`), 0644)
		require.NoError(t, err)
		_, err = LoadConfig(afero.NewBasePathFs(fsys, "invalid"))
		require.EqualError(t, err, "[1:10] mapping value is not allowed in this context\n>  1 | invalid: yaml: content\n                ^\n")
	})

	t.Run("nonexistent config", func(t *testing.T) {
		cfg, err := LoadConfig(afero.NewBasePathFs(fsys, "nonexistent"))
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Empty(t, cfg.Aliases)
	})

	t.Run("read error", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := filepath.Join(tmpDir, config.DefaultFileName)
		err = os.Mkdir(configDir, 0755)
		require.NoError(t, err)

		_, err := LoadConfig(afero.NewBasePathFs(afero.NewOsFs(), tmpDir))
		require.EqualError(t, err, fmt.Sprintf("failed to read config file: read %s: is a directory", configDir))
	})

	t.Run("open error", func(t *testing.T) {
		tmpDir := t.TempDir()

		configPath := filepath.Join(tmpDir, config.DefaultFileName)
		err = os.WriteFile(configPath, []byte(`valid: yaml`), 0000)
		require.NoError(t, err)

		_, err = LoadConfig(afero.NewBasePathFs(afero.NewOsFs(), tmpDir))
		require.EqualError(t, err, fmt.Sprintf("failed to open config file: open %s: permission denied", filepath.Join(tmpDir, config.DefaultFileName)))
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: map[string]v0.Alias{
					"gh": {
						Type: packageurl.TypeGithub,
					},
					"gl": {
						Type: packageurl.TypeGitlab,
						Base: "https://gitlab.example.com",
					},
					"custom": {
						Type:         packageurl.TypeGithub,
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
				FetchPolicy: uses.FetchPolicyIfNotPresent,
			},
		},
		{
			name: "invalid alias type",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: map[string]v0.Alias{
					"invalid": {
						Type: "invalid-type",
					},
				},
				FetchPolicy: uses.FetchPolicyIfNotPresent,
			},
			wantErr: true,
			errMsg:  "aliases.invalid.type: aliases.invalid.type must be one of the following: \"github\", \"gitlab\"",
		},
		{
			name: "invalid token environment variable format",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: map[string]v0.Alias{
					"gh": {
						Type:         packageurl.TypeGithub,
						TokenFromEnv: "123-invalid",
					},
				},
				FetchPolicy: uses.FetchPolicyIfNotPresent,
			},
			wantErr: true,
			errMsg:  "aliases.gh.token-from-env: Does not match pattern '^[a-zA-Z_]+[a-zA-Z0-9_]*$'",
		},
		{
			name: "invalid fetch policy",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: map[string]v0.Alias{
					"gh": {
						Type: packageurl.TypeGithub,
					},
				},
				FetchPolicy: uses.FetchPolicy("invalid-policy"),
			},
			wantErr: true,
			errMsg:  "fetch-policy: fetch-policy must be one of the following: \"always\", \"if-not-present\", \"never\"",
		},
		{
			name: "multiple validation errors",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: map[string]v0.Alias{
					"invalid": {
						Type:         "invalid-type",
						TokenFromEnv: "123-invalid",
					},
				},
				FetchPolicy: "invalid-policy",
			},
			wantErr: true,
			// We're not testing the exact error message here as the order of errors might vary
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
