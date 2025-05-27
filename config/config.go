// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package config provides system-level configuration for maru2
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/invopop/jsonschema"
	"github.com/spf13/afero"
)

// DefaultFileName is the default file name for the config file
const DefaultFileName = "config.yaml"

// Config is the system configuration file for maru2
type Config struct {
	Aliases map[string]Alias `yaml:"aliases"`
}

// Alias defines how an alias should be resolved
type Alias struct {
	Type         string `json:"type"`
	Base         string `json:"base,omitempty"`
	TokenFromEnv string `json:"token-from-env,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for an alias
func (Alias) JSONSchemaExtend(schema *jsonschema.Schema) {
	if typ, ok := schema.Properties.Get("type"); ok && typ != nil {
		typ.Description = "Type of the alias, maps to a package URL type"
		typ.Enum = []any{"github", "gitlab"}
	}

	if base, ok := schema.Properties.Get("base"); ok && base != nil {
		base.Description = "Base URL for the underlying client (e.g. https://mygitlab.com )"
	}

	if tokenFromEnv, ok := schema.Properties.Get("token-from-env"); ok && tokenFromEnv != nil {
		tokenFromEnv.Description = "Environment variable containing the token for authentication"
		tokenFromEnv.Pattern = "^[a-zA-Z_]+[a-zA-Z0-9_]*$" // EnvVariablePattern.String(), a little bit of copying never hurt anyone
	}
}

// FileSystemConfigLoader loads configuration from the file system
type FileSystemConfigLoader struct {
	Fs afero.Fs
}

// DefaultDirectory returns the default directory for maru2 configuration
func DefaultDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".maru2"), nil
}

// LoadConfig loads the configuration from the file system
func (l *FileSystemConfigLoader) LoadConfig() (*Config, error) {
	config := &Config{
		Aliases: map[string]Alias{},
	}

	f, err := l.Fs.Open(DefaultFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}
