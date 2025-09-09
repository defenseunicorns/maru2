// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package v0 provides the schema for v0 of the system config file for maru2
//
// v0 allows for breaking changes without a major version increase
package v0

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
	"github.com/xeipuuv/gojsonschema"

	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

// SchemaVersion is the current schema version for configs
const SchemaVersion = "v0"

// Config is the system configuration file for maru2
type Config struct {
	SchemaVersion string           `json:"schema-version"`
	Aliases       v1.AliasMap      `json:"aliases"`
	FetchPolicy   uses.FetchPolicy `json:"fetch-policy"`
}

// the default config, matches flag defaults in cmd/root.go
func defaultConfig() *Config {
	return &Config{
		SchemaVersion: SchemaVersion,
		Aliases:       v1.AliasMap{},
		FetchPolicy:   uses.DefaultFetchPolicy,
	}
}

// JSONSchemaExtend extends the JSON schema for a workflow
func (Config) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schemaVersion, ok := schema.Properties.Get("schema-version"); ok && schemaVersion != nil {
		schemaVersion.Description = "Config schema version"
		schemaVersion.Enum = []any{SchemaVersion}
		schemaVersion.AdditionalProperties = jsonschema.FalseSchema
	}
}

// LoadConfig loads the configuration from the file system
//
// # It assumes the provided fs's base directory contains a valid configuration file
//
// If the configuration file does not exist, this function returns a default valid but "empty" config
func LoadConfig(r io.Reader) (*Config, error) {
	cfg := defaultConfig()

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var versioned schema.Versioned
	if err := yaml.Unmarshal(data, &versioned); err != nil {
		return nil, err
	}

	switch version := versioned.SchemaVersion; version {
	case SchemaVersion:
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
		if err := Validate(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	// See schema/v1/validate.go for an example on how auto migrations during loading/reading can work for when v1 of config is released
	default:
		return nil, fmt.Errorf("unsupported config schema version: expected %q, got %q", SchemaVersion, version)
	}
}

// LoadDefaultConfig loads the config from config.DefaultDirectory
// if this file does not exist, the default config is returned
func LoadDefaultConfig() (*Config, error) {
	configDir, err := config.DefaultDirectory()
	if err != nil {
		return nil, err
	}

	cfg := defaultConfig()

	f, err := os.Open(filepath.Join(configDir, config.DefaultFileName))
	if err != nil {
		if os.IsNotExist(err) { // default config is allowed to not exist
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	loaded, err := LoadConfig(f)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	return loaded, nil
}

// Since every validation operation leverages the same config, only calculate it once to save some compute cycles
//
// This also prevents any schema changes from occurring at runtime
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

// Schema generates the JSON schema for v0 configuration validation
//
// Returns a schema for IDE integration and automated validation
func Schema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{DoNotReference: true}
	return reflector.Reflect(&Config{})
}
