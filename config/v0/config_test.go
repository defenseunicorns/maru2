// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/config"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		reader    io.Reader
		expected  *Config
		expectErr string
	}{
		{
			name: "valid config",
			reader: strings.NewReader(`schema-version: v0
aliases:
  gh:
    type: github
  gl:
    type: gitlab
    base-url: https://gitlab.example.com
    token-from-env: GL_TOKEN
fetch-policy: always`),
			expected: &Config{
				SchemaVersion: SchemaVersion,
				FetchPolicy:   uses.FetchPolicyAlways,
				Aliases: v1.AliasMap{
					"gh": {
						Type: packageurl.TypeGithub,
					},
					"gl": {
						Type:         packageurl.TypeGitlab,
						BaseURL:      "https://gitlab.example.com",
						TokenFromEnv: "GL_TOKEN",
					},
				},
			},
		},
		{
			name:   "empty config uses defaults",
			reader: strings.NewReader(`schema-version: v0`),
			expected: &Config{
				SchemaVersion: SchemaVersion,
				Aliases:       v1.AliasMap{},
				FetchPolicy:   uses.DefaultFetchPolicy,
			},
		},
		{
			name:      "invalid yaml",
			reader:    strings.NewReader(`invalid: yaml: content`),
			expectErr: "mapping value is not allowed in this context",
		},
		{
			name: "unsupported schema version",
			reader: strings.NewReader(`schema-version: v999
aliases:
  gh:
    type: github`),
			expectErr: `unsupported config schema version: expected "v0", got "v999"`,
		},
		{
			name: "invalid structure",
			reader: strings.NewReader(`schema-version: v0
aliases: "should-be-map"`),
			expectErr: "failed to parse config file",
		},
		{
			name: "validation error",
			reader: strings.NewReader(`schema-version: v0
aliases:
  invalid:
    type: bad-type`),
			expectErr: "aliases.invalid.type",
		},
		{
			name:      "reader error",
			reader:    iotest.ErrReader(assert.AnError),
			expectErr: assert.AnError.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := LoadConfig(tt.reader)

			if tt.expectErr != "" {
				assert.Nil(t, cfg)
				require.ErrorContains(t, err, tt.expectErr)
				return
			}

			require.Equal(t, tt.expected, cfg)
		})
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	setupTempHome := func(t *testing.T, configContent string) string {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".maru2")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		if configContent != "" {
			configPath := filepath.Join(configDir, config.DefaultFileName)
			require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o644))
		}

		t.Setenv("HOME", tmpDir)

		return tmpDir
	}

	tests := []struct {
		name          string
		configContent string
		expectErr     string
		expected      *Config
	}{
		{
			name:          "no config file returns defaults",
			configContent: "",
			expected: &Config{
				SchemaVersion: "",
				Aliases:       v1.AliasMap{},
				FetchPolicy:   uses.DefaultFetchPolicy,
			},
		},
		{
			name: "valid config file loads correctly",
			configContent: `schema-version: v0
aliases:
  gh:
    type: github
fetch-policy: always`,
			expected: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: v1.AliasMap{
					"gh": {Type: packageurl.TypeGithub},
				},
				FetchPolicy: uses.FetchPolicyAlways,
			},
		},
		{
			name:          "invalid config file returns error",
			configContent: `schema-version: v999`,
			expectErr:     `failed to load config file: unsupported config schema version: expected "v0", got "v999"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTempHome(t, tt.configContent)

			cfg, err := LoadDefaultConfig()

			if tt.expectErr != "" {
				assert.Nil(t, cfg)
				require.EqualError(t, err, tt.expectErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, cfg)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		config      *Config
		expectedErr string
	}{
		{
			name: "valid config",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: v1.AliasMap{
					"gh": {Type: packageurl.TypeGithub},
					"gl": {Type: packageurl.TypeGitlab, BaseURL: "https://gitlab.example.com"},
				},
				FetchPolicy: uses.FetchPolicyIfNotPresent,
			},
		},
		{
			name: "invalid alias type",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases:       v1.AliasMap{"invalid": {Type: "bad-type"}},
				FetchPolicy:   uses.FetchPolicyIfNotPresent,
			},
			expectedErr: "aliases.invalid.type",
		},
		{
			name: "invalid token env var format",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases:       v1.AliasMap{"gh": {Type: packageurl.TypeGithub, TokenFromEnv: "123-invalid"}},
				FetchPolicy:   uses.FetchPolicyIfNotPresent,
			},
			expectedErr: "Does not match pattern",
		},
		{
			name: "invalid fetch policy",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases:       v1.AliasMap{"gh": {Type: packageurl.TypeGithub}},
				FetchPolicy:   "invalid-policy",
			},
			expectedErr: "fetch-policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := Validate(tt.config)
			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
