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

// FileSystemConfigLoader loads configuration from the file system
type FileSystemConfigLoader struct {
	Fs afero.Fs
}

// DefaultConfigDirectory returns the default directory for maru2 configuration
func DefaultConfigDirectory() (string, error) {
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
