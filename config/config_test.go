// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	cfg := &Config{
		Aliases: map[string]Alias{
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

	assert.Len(t, config.Aliases, 3)

	glAlias, ok := config.Aliases["gl"]
	assert.True(t, ok)
	assert.Equal(t, packageurl.TypeGitlab, glAlias.Type)
	assert.Equal(t, "https://gitlab.example.com", glAlias.Base)

	ghAlias, ok := config.Aliases["gh"]
	assert.True(t, ok)
	assert.Equal(t, packageurl.TypeGithub, ghAlias.Type)
	assert.Empty(t, ghAlias.Base)

	loader = NewFileSystemConfigLoader(fsys, "nonexistent-file.yaml")
	config, err = loader.LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.Aliases)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Setenv("HOME", "")
		loader, err = DefaultConfigLoader()
		assert.Nil(t, loader)
		require.EqualError(t, err, "$HOME is not defined")

		tmpDir := t.TempDir()
		err = os.Mkdir(filepath.Join(tmpDir, ".maru2"), 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".maru2", DefaultFileName), []byte(configContent), 0644)
		require.NoError(t, err)

		t.Setenv("HOME", tmpDir)
		loader, err = DefaultConfigLoader()
		require.NoError(t, err)
		config, err = loader.LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, cfg.Aliases, config.Aliases)
	}
}

func TestConfigLoaderWithInvalidYAML(t *testing.T) {
	fsys := afero.NewMemMapFs()
	err := afero.WriteFile(fsys, "invalid.yaml", []byte(`invalid: yaml: content`), 0644)
	require.NoError(t, err)

	loader := NewFileSystemConfigLoader(fsys, "invalid.yaml")
	_, err = loader.LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}
