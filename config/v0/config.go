// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package v0 provides the schema for v0 of the system config file for maru2
package v0

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/invopop/jsonschema"
	"github.com/spf13/afero"
	"github.com/xeipuuv/gojsonschema"

	"github.com/defenseunicorns/maru2/config"
	v0 "github.com/defenseunicorns/maru2/schema/v0"
	"github.com/defenseunicorns/maru2/uses"
)

// Config is the system configuration file for maru2
type Config struct {
	Aliases     map[string]v0.Alias `json:"aliases"`
	FetchPolicy uses.FetchPolicy    `json:"fetch-policy"`
}

// LoadConfig loads the configuration from the file system
func LoadConfig(fsys afero.Fs) (*Config, error) {
	cfg := &Config{
		Aliases:     map[string]v0.Alias{},
		FetchPolicy: uses.DefaultFetchPolicy,
	}

	f, err := fsys.Open(config.DefaultFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
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
