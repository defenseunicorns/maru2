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
	err := afero.WriteFile(fsys, "etc/maru2/config.yaml", []byte(configContent), 0644)
	require.NoError(t, err)

	loader := &FileSystemConfigLoader{
		Fs: afero.NewBasePathFs(fsys, "etc/maru2"),
	}
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

	loader = &FileSystemConfigLoader{
		Fs: afero.NewBasePathFs(fsys, "nonexistent-dir"),
	}
	config, err = loader.LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Empty(t, config.Aliases)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Setenv("HOME", "")
		configDir, err := DefaultConfigDirectory()
		assert.Empty(t, configDir)
		require.EqualError(t, err, "$HOME is not defined")

		tmpDir := t.TempDir()
		err = os.Mkdir(filepath.Join(tmpDir, ".maru2"), 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, ".maru2", DefaultFileName), []byte(configContent), 0644)
		require.NoError(t, err)

		t.Setenv("HOME", tmpDir)
		configDir, err = DefaultConfigDirectory()
		require.NoError(t, err)
		loader = &FileSystemConfigLoader{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), configDir),
		}
		config, err = loader.LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, cfg.Aliases, config.Aliases)
	}
}

func TestConfigLoaderWithInvalidYAML(t *testing.T) {
	fsys := afero.NewMemMapFs()
	err := afero.WriteFile(fsys, "invalid/config.yaml", []byte(`invalid: yaml: content`), 0644)
	require.NoError(t, err)

	loader := &FileSystemConfigLoader{
		Fs: afero.NewBasePathFs(fsys, "invalid"),
	}
	_, err = loader.LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}
