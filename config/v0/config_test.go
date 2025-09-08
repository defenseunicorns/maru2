// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
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
		name        string
		content     string
		expectErr   string
		expectValid bool
	}{
		{
			name: "valid config",
			content: `schema-version: v0
aliases:
  gh:
    type: github
  gl:
    type: gitlab
    base-url: https://gitlab.example.com
    token-from-env: GL_TOKEN
fetch-policy: always`,
			expectValid: true,
		},
		{
			name:        "empty config uses defaults",
			content:     `schema-version: v0`,
			expectValid: true,
		},
		{
			name:      "invalid yaml",
			content:   `invalid: yaml: content`,
			expectErr: "mapping value is not allowed in this context",
		},
		{
			name: "unsupported schema version",
			content: `schema-version: v999
aliases:
  gh:
    type: github`,
			expectErr: `unsupported config schema version: expected "v0", got "v999"`,
		},
		{
			name: "invalid structure",
			content: `schema-version: v0
aliases: "should-be-map"`,
			expectErr: "failed to parse config file",
		},
		{
			name: "validation error",
			content: `schema-version: v0
aliases:
  invalid:
    type: bad-type`,
			expectErr: "aliases.invalid.type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadConfig(strings.NewReader(tt.content))

			if tt.expectErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, SchemaVersion, cfg.SchemaVersion)
			assert.NotNil(t, cfg.Aliases)
			assert.NotEmpty(t, cfg.FetchPolicy)
		})
	}

	// Test reader edge cases
	t.Run("reader edge cases", func(t *testing.T) {
		content := `schema-version: v0
aliases:
  gh:
    type: github`

		// Test various reader implementations work correctly
		readers := map[string]func(string) *strings.Reader{
			"one_byte": func(s string) *strings.Reader { return strings.NewReader(s) },
			"half":     func(s string) *strings.Reader { return strings.NewReader(s) },
		}

		for name, readerFunc := range readers {
			t.Run(name, func(t *testing.T) {
				var reader interface {
					Read([]byte) (int, error)
				}
				switch name {
				case "one_byte":
					reader = iotest.OneByteReader(readerFunc(content))
				case "half":
					reader = iotest.HalfReader(readerFunc(content))
				}

				cfg, err := LoadConfig(reader)
				require.NoError(t, err)
				assert.Len(t, cfg.Aliases, 1)
			})
		}

		// Test read error
		_, err := LoadConfig(iotest.ErrReader(assert.AnError))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read config file")
	})
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

		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		t.Cleanup(func() { os.Setenv("HOME", originalHome) })

		return tmpDir
	}

	t.Run("no config file returns defaults", func(t *testing.T) {
		setupTempHome(t, "")

		cfg, err := LoadDefaultConfig()
		require.NoError(t, err)
		assert.Empty(t, cfg.Aliases)
		assert.Equal(t, uses.DefaultFetchPolicy, cfg.FetchPolicy)
	})

	t.Run("valid config file loads correctly", func(t *testing.T) {
		content := `schema-version: v0
aliases:
  gh:
    type: github
fetch-policy: always`
		setupTempHome(t, content)

		cfg, err := LoadDefaultConfig()
		require.NoError(t, err)
		assert.Len(t, cfg.Aliases, 1)
		assert.Equal(t, uses.FetchPolicyAlways, cfg.FetchPolicy)
	})

	t.Run("invalid config file returns error", func(t *testing.T) {
		setupTempHome(t, `schema-version: v999`)

		_, err := LoadDefaultConfig()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to load config file")
	})
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
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
