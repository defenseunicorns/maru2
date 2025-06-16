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
)

// DefaultFileName is the default file name for the config file
const DefaultFileName = "config.yaml"

// Config is the system configuration file for maru2
type Config struct {
	Aliases     map[string]Alias `json:"aliases"`
	FetchPolicy FetchPolicy      `json:"fetch-policy"`
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
		Aliases:     map[string]Alias{},
		FetchPolicy: DefaultFetchPolicy,
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

var _schema string
var _schemaOnce sync.Once
var _schemaOnceErr error

// Validate checks if a config adheres to the JSON schema
func Validate(config *Config) error {
	_schemaOnce.Do(func() {
		s := Schema()
		b, err := json.Marshal(s)
		if err != nil {
			_schemaOnceErr = err
		}
		_schema = string(b)
	})

	if _schemaOnceErr != nil {
		return _schemaOnceErr
	}

	schemaLoader := gojsonschema.NewStringLoader(_schema)

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
