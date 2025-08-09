// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package config provides system-level configuration for maru2
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/invopop/jsonschema"
	"github.com/spf13/afero"
	"github.com/xeipuuv/gojsonschema"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	"github.com/defenseunicorns/maru2/uses"
)

// DefaultFileName is the default file name for the config file
const DefaultFileName = "config.yaml"

// Config is the system configuration file for maru2
type Config struct {
	Aliases     map[string]v0.Alias `json:"aliases"`
	FetchPolicy uses.FetchPolicy    `json:"fetch-policy"`
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
		Aliases:     map[string]v0.Alias{},
		FetchPolicy: uses.DefaultFetchPolicy,
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

	if err := Validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

var schemaOnce = sync.OnceValues(func() (string, error) {
	s := Schema()
	b, err := json.Marshal(s)
	return string(b), err
})

// Validate checks if a config adheres to the JSON schema
func Validate(config *Config) error {
	schema, err := schemaOnce()
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)

	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewGoLoader(config))
	if err != nil {
		return err
	}

	if result.Valid() {
		return nil
	}

	var resErr error
	for _, err := range result.Errors() {
		resErr = errors.Join(resErr, errors.New(err.String()))
	}

	return resErr
}

// Schema returns the JSON schema for the Config type
func Schema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	return reflector.Reflect(&Config{})
}
