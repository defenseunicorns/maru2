// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/invopop/jsonschema"
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

	// cfg := &Config{
	// 	Aliases: map[string]Alias{
	// 		"gl": {
	// 			Type: packageurl.TypeGitlab,
	// 			Base: "https://gitlab.example.com",
	// 		},
	// 		"gh": {
	// 			Type: packageurl.TypeGithub,
	// 		},
	// 		"another": {
	// 			Type:         packageurl.TypeGithub,
	// 			TokenFromEnv: "GITHUB_TOKEN",
	// 		},
	// 	},
	// }

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
		configDir, err := DefaultDirectory()
		assert.Empty(t, configDir)
		require.EqualError(t, err, "$HOME is not defined")

		// 	tmpDir := t.TempDir()
		// 	err = os.Mkdir(filepath.Join(tmpDir, ".maru2"), 0755)
		// 	require.NoError(t, err)

		// 	err = os.WriteFile(filepath.Join(tmpDir, ".maru2", DefaultFileName), []byte(configContent), 0644)
		// 	require.NoError(t, err)

		t.Setenv("HOME", tmpDir)
		configDir, err = DefaultDirectory()
		require.NoError(t, err)
		loader = &FileSystemConfigLoader{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), configDir),
		}
		config, err = loader.LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, cfg.Aliases, config.Aliases)
	}

	t.Run("invalid config", func(t *testing.T) {
		fsys = afero.NewMemMapFs()
		err = afero.WriteFile(fsys, "invalid/config.yaml", []byte(`invalid: yaml: content`), 0644)
		require.NoError(t, err)
		loader := &FileSystemConfigLoader{
			Fs: afero.NewBasePathFs(fsys, "invalid"),
		}
		_, err = loader.LoadConfig()
		require.EqualError(t, err, "failed to parse config file: [1:10] mapping value is not allowed in this context\n>  1 | invalid: yaml: content\n                ^\n")
	})

	t.Run("nonexistent config", func(t *testing.T) {
		loader := &FileSystemConfigLoader{
			Fs: afero.NewBasePathFs(fsys, "nonexistent"),
		}
		config, err := loader.LoadConfig()
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Empty(t, config.Aliases)
	})

	t.Run("read error", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := filepath.Join(tmpDir, DefaultFileName)
		err = os.Mkdir(configDir, 0755)
		require.NoError(t, err)

		loader := &FileSystemConfigLoader{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}

		_, err := loader.LoadConfig()
		require.EqualError(t, err, fmt.Sprintf("failed to read config file: read %s: is a directory", configDir))
	})

	t.Run("open error", func(t *testing.T) {
		tmpDir := t.TempDir()

		configPath := filepath.Join(tmpDir, DefaultFileName)
		err = os.WriteFile(configPath, []byte(`valid: yaml`), 0000)
		require.NoError(t, err)

		loader := &FileSystemConfigLoader{
			Fs: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}
		_, err = loader.LoadConfig()
		require.EqualError(t, err, fmt.Sprintf("failed to open config file: open %s: permission denied", filepath.Join(tmpDir, DefaultFileName)))
	})
}

func TestConfigSchema(t *testing.T) {
	f, err := os.Open("../maru2.schema.json")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	curr := schema["$defs"].(map[string]any)["Alias"]
	b, err := json.Marshal(curr)
	require.NoError(t, err)

	reflector := jsonschema.Reflector{ExpandedStruct: true}
	aliasSchema := reflector.Reflect(&Alias{})
	aliasSchema.Version = ""
	aliasSchema.ID = ""
	b2, err := json.Marshal(aliasSchema)
	require.NoError(t, err)

	assert.JSONEq(t, string(b), string(b2))
}
