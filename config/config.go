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
	"github.com/spf13/pflag"
)

// DefaultFileName is the default file name for the config file
const DefaultFileName = "config.yaml"

// Config is the system configuration file for maru2
type Config struct {
	Aliases     map[string]Alias `yaml:"aliases"`
	FetchPolicy FetchPolicy      `yaml:"fetch-policy"`
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

// FetchPolicy defines the fetching behavior for the fetcher service
type FetchPolicy string

var _ pflag.Value = (*FetchPolicy)(nil)

// AvailablePolicies returns a list of available fetch policies
func AvailablePolicies() []string {
	return []string{
		string(FetchPolicyAlways),
		string(FetchPolicyIfNotPresent),
		string(FetchPolicyNever),
	}
}

const (
	// FetchPolicyAlways will always use the cache if available, never fetching from source
	FetchPolicyAlways FetchPolicy = "always"
	// FetchPolicyIfNotPresent will use the cache if available, otherwise fetch from source
	FetchPolicyIfNotPresent FetchPolicy = "if-not-present"
	// FetchPolicyNever will never use the cache, always fetching from source
	FetchPolicyNever FetchPolicy = "never"
	// DefaultFetchPolicy is the default fetch policy used when none is specified
	DefaultFetchPolicy FetchPolicy = FetchPolicyIfNotPresent
)

// String implements the pflag.Value and fmt.Stringer interfaces
func (f *FetchPolicy) String() string {
	return string(*f)
}

// Set implements the pflag.Value interface
func (f *FetchPolicy) Set(value string) error {
	switch value {
	case string(FetchPolicyAlways):
		*f = FetchPolicyAlways
	case string(FetchPolicyIfNotPresent):
		*f = FetchPolicyIfNotPresent
	case string(FetchPolicyNever):
		*f = FetchPolicyNever
	default:
		return fmt.Errorf("invalid fetch policy: %s", value)
	}
	return nil
}

// Type implements the pflag.Value interface
func (f *FetchPolicy) Type() string {
	return "string"
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

	return config, nil
}
