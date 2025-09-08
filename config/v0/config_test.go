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
	t.Run("valid config", func(t *testing.T) {
		configContent := `schema-version: v0
aliases:
  gl:
    type: gitlab
    base-url: https://gitlab.example.com
  gh:
    type: github
  another:
    type: github
    token-from-env: GITHUB_TOKEN
fetch-policy: if-not-present
`

		tcfg := &Config{
			SchemaVersion: SchemaVersion,
			Aliases: v1.AliasMap{
				"gh": {
					Type: packageurl.TypeGithub,
				},
				"gl": {
					Type:    packageurl.TypeGitlab,
					BaseURL: "https://gitlab.example.com",
				},
				"another": {
					Type:         packageurl.TypeGithub,
					TokenFromEnv: "GITHUB_TOKEN",
				},
			},
			FetchPolicy: uses.FetchPolicyIfNotPresent,
		}

		reader := strings.NewReader(configContent)
		cfg, err := LoadConfig(reader)
		require.NoError(t, err)

		assert.Equal(t, tcfg, cfg)
	})

	t.Run("empty config", func(t *testing.T) {
		configContent := `schema-version: v0`
		reader := strings.NewReader(configContent)
		cfg, err := LoadConfig(reader)
		require.NoError(t, err)

		assert.Equal(t, SchemaVersion, cfg.SchemaVersion)
		assert.Empty(t, cfg.Aliases)
		assert.Equal(t, uses.DefaultFetchPolicy, cfg.FetchPolicy)
	})

	t.Run("invalid yaml", func(t *testing.T) {
		configContent := `invalid: yaml: content`
		reader := strings.NewReader(configContent)
		cfg, err := LoadConfig(reader)
		assert.Nil(t, cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "mapping value is not allowed in this context")
	})

	t.Run("unsupported schema version", func(t *testing.T) {
		configContent := `schema-version: v999
aliases:
  gh:
    type: github
`
		reader := strings.NewReader(configContent)
		cfg, err := LoadConfig(reader)
		assert.Nil(t, cfg)
		require.EqualError(t, err, `unsupported config schema version: expected "v0", got "v999"`)
	})

	t.Run("failed to parse config file", func(t *testing.T) {
		configContent := `schema-version: v0
aliases: "invalid-type-should-be-map"
fetch-policy: if-not-present
`
		reader := strings.NewReader(configContent)
		cfg, err := LoadConfig(reader)
		assert.Nil(t, cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse config file:")
	})

	t.Run("validation error", func(t *testing.T) {
		configContent := `schema-version: v0
aliases:
  invalid:
    type: invalid-type
`
		reader := strings.NewReader(configContent)
		cfg, err := LoadConfig(reader)
		assert.Nil(t, cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "aliases.invalid.type")
	})

	t.Run("read error", func(t *testing.T) {
		reader := iotest.ErrReader(assert.AnError)
		cfg, err := LoadConfig(reader)
		assert.Nil(t, cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read config file:")
	})

	t.Run("one byte reader", func(t *testing.T) {
		configContent := `schema-version: v0
aliases:
  gh:
    type: github
`
		reader := iotest.OneByteReader(strings.NewReader(configContent))
		cfg, err := LoadConfig(reader)
		require.NoError(t, err)
		assert.Equal(t, SchemaVersion, cfg.SchemaVersion)
		assert.Len(t, cfg.Aliases, 1)
	})

	t.Run("half reader", func(t *testing.T) {
		configContent := `schema-version: v0
aliases:
  gh:
    type: github
fetch-policy: always
`
		reader := iotest.HalfReader(strings.NewReader(configContent))
		cfg, err := LoadConfig(reader)
		require.NoError(t, err)
		assert.Equal(t, SchemaVersion, cfg.SchemaVersion)
		assert.Equal(t, uses.FetchPolicyAlways, cfg.FetchPolicy)
	})

	t.Run("data err reader", func(t *testing.T) {
		configContent := `schema-version: v0`
		reader := iotest.DataErrReader(strings.NewReader(configContent))
		cfg, err := LoadConfig(reader)
		require.NoError(t, err)
		assert.Equal(t, SchemaVersion, cfg.SchemaVersion)
	})
}

func TestLoadDefaultConfig(t *testing.T) {
	t.Run("nonexistent config file", func(t *testing.T) {
		// Create a temporary directory that doesn't contain a config file
		tmpDir := t.TempDir()

		// Set up environment to use our temporary directory
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		t.Cleanup(func() {
			os.Setenv("HOME", originalHome)
		})

		cfg, err := LoadDefaultConfig()
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Empty(t, cfg.Aliases)
		assert.Equal(t, uses.DefaultFetchPolicy, cfg.FetchPolicy)
	})

	t.Run("valid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".maru2")
		err := os.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configContent := `schema-version: v0
aliases:
  gh:
    type: github
fetch-policy: always
`
		configPath := filepath.Join(configDir, config.DefaultFileName)
		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Set up environment to use our temporary directory
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		t.Cleanup(func() {
			os.Setenv("HOME", originalHome)
		})

		cfg, err := LoadDefaultConfig()
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Len(t, cfg.Aliases, 1)
		assert.Equal(t, uses.FetchPolicyAlways, cfg.FetchPolicy)
	})

	t.Run("invalid config file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".maru2")
		err := os.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configContent := `schema-version: v999`
		configPath := filepath.Join(configDir, config.DefaultFileName)
		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Set up environment to use our temporary directory
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		t.Cleanup(func() {
			os.Setenv("HOME", originalHome)
		})

		_, err = LoadDefaultConfig()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to load config file:")
	})

	t.Run("config file is directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".maru2")
		err := os.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		// Create a directory with the config file name
		configPath := filepath.Join(configDir, config.DefaultFileName)
		err = os.Mkdir(configPath, 0o755)
		require.NoError(t, err)

		// Set up environment to use our temporary directory
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		t.Cleanup(func() {
			os.Setenv("HOME", originalHome)
		})

		_, err = LoadDefaultConfig()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to load config file:")
	})

	t.Run("permission denied", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := filepath.Join(tmpDir, ".maru2")
		err := os.MkdirAll(configDir, 0o755)
		require.NoError(t, err)

		configContent := `schema-version: v0`
		configPath := filepath.Join(configDir, config.DefaultFileName)
		err = os.WriteFile(configPath, []byte(configContent), 0o000)
		require.NoError(t, err)

		// Set up environment to use our temporary directory
		originalHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		t.Cleanup(func() {
			os.Setenv("HOME", originalHome)
		})

		_, err = LoadDefaultConfig()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to open config file:")
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
					"gh": {
						Type: packageurl.TypeGithub,
					},
					"gl": {
						Type:    packageurl.TypeGitlab,
						BaseURL: "https://gitlab.example.com",
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
				Aliases: v1.AliasMap{
					"invalid": {
						Type: "invalid-type",
					},
				},
				FetchPolicy: uses.FetchPolicyIfNotPresent,
			},
			expectedErr: "aliases.invalid.type: aliases.invalid.type must be one of the following: \"github\", \"gitlab\"",
		},
		{
			name: "invalid token environment variable format",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: v1.AliasMap{
					"gh": {
						Type:         packageurl.TypeGithub,
						TokenFromEnv: "123-invalid",
					},
				},
				FetchPolicy: uses.FetchPolicyIfNotPresent,
			},
			expectedErr: "aliases.gh.token-from-env: Does not match pattern '^[a-zA-Z_]+[a-zA-Z0-9_]*$'",
		},
		{
			name: "invalid fetch policy",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: v1.AliasMap{
					"gh": {
						Type: packageurl.TypeGithub,
					},
				},
				FetchPolicy: uses.FetchPolicy("invalid-policy"),
			},
			expectedErr: "fetch-policy: fetch-policy must be one of the following: \"always\", \"if-not-present\", \"never\"",
		},
		{
			name: "multiple validation errors",
			config: &Config{
				SchemaVersion: SchemaVersion,
				Aliases: v1.AliasMap{
					"invalid": {
						Type:         "invalid-type",
						TokenFromEnv: "123-invalid",
					},
				},
				FetchPolicy: "invalid-policy",
			},
			expectedErr: "aliases.invalid.type: aliases.invalid.type must be one of the following",
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
