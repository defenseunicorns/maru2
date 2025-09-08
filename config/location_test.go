// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/config"
	configv0 "github.com/defenseunicorns/maru2/config/v0"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

func TestDefaultDirectory(t *testing.T) {
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
fetch-policy: always
`

	tcfg := &configv0.Config{
		SchemaVersion: configv0.SchemaVersion,
		Aliases: v1.AliasMap{
			"gl": {
				Type:    packageurl.TypeGitlab,
				BaseURL: "https://gitlab.example.com",
			},
			"gh": {
				Type: packageurl.TypeGithub,
			},
			"another": {
				Type:         packageurl.TypeGithub,
				TokenFromEnv: "GITHUB_TOKEN",
			},
		},
		FetchPolicy: uses.FetchPolicyAlways,
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Setenv("HOME", "")
		configDir, err := config.DefaultDirectory()
		assert.Empty(t, configDir)
		require.EqualError(t, err, "$HOME is not defined")

		tmpDir := t.TempDir()
		err = os.Mkdir(filepath.Join(tmpDir, ".maru2"), 0o755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".maru2", config.DefaultFileName), []byte(configContent), 0o644)
		require.NoError(t, err)

		t.Setenv("HOME", tmpDir)
		configDir, err = config.DefaultDirectory()
		assert.Equal(t, filepath.Join(tmpDir, ".maru2"), configDir)
		require.NoError(t, err)
		f, err := os.Open(filepath.Join(configDir, config.DefaultFileName))
		require.NoError(t, err)
		t.Cleanup(func() {
			f.Close()
		})

		cfg, err := configv0.LoadConfig(f)
		require.NoError(t, err)
		assert.Equal(t, tcfg, cfg)

		cfg, err = configv0.LoadDefaultConfig()
		require.NoError(t, err)
		assert.Equal(t, tcfg, cfg)
	}
}
