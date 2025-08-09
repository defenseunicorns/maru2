// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/config"
	configv0 "github.com/defenseunicorns/maru2/config/v0"
	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

func TestDefaultDirectory(t *testing.T) {
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

	tcfg := &configv0.Config{
		SchemaVersion: v0.SchemaVersion,
		Aliases: map[string]v0.Alias{
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

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Setenv("HOME", "")
		configDir, err := config.DefaultDirectory()
		assert.Empty(t, configDir)
		require.EqualError(t, err, "$HOME is not defined")

		tmpDir := t.TempDir()
		err = os.Mkdir(filepath.Join(tmpDir, ".maru2"), 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".maru2", config.DefaultFileName), []byte(configContent), 0644)
		require.NoError(t, err)

		t.Setenv("HOME", tmpDir)
		configDir, err = config.DefaultDirectory()
		assert.Equal(t, filepath.Join(tmpDir, ".maru2"), configDir)
		require.NoError(t, err)
		cfg, err := configv0.LoadConfig(afero.NewBasePathFs(afero.NewOsFs(), configDir))
		require.NoError(t, err)
		assert.Equal(t, tcfg.Aliases, cfg.Aliases)
	}
}
