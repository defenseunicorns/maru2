// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
)

// AliasConfig represents the configuration for package URL aliases
type AliasConfig struct {
	Aliases map[string]AliasDefinition `yaml:"aliases"`
}

// AliasDefinition defines how an alias should be resolved
type AliasDefinition struct {
	Type         string `yaml:"type"`
	Base         string `yaml:"base,omitempty"`
	TokenFromEnv string `yaml:"token-from-env,omitempty"`
}

// FileSystemConfigLoader loads configuration from the file system
type FileSystemConfigLoader struct {
	fs         afero.Fs
	configPath string
}

// NewFileSystemConfigLoader creates a new FileSystemConfigLoader
func NewFileSystemConfigLoader(fsys afero.Fs, configPath string) *FileSystemConfigLoader {
	return &FileSystemConfigLoader{
		fs:         fsys,
		configPath: configPath,
	}
}

// DefaultConfigLoader returns a config loader that uses the default locations
func DefaultConfigLoader() (*FileSystemConfigLoader, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".maru2", "aliases.yaml")
	return NewFileSystemConfigLoader(afero.NewOsFs(), configPath), nil
}

// LoadConfig loads the alias configuration from the file system
func (l *FileSystemConfigLoader) LoadConfig() (*AliasConfig, error) {
	config := &AliasConfig{
		Aliases: map[string]AliasDefinition{},
	}

	f, err := l.fs.Open(l.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to open alias config file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read alias config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse alias config file: %w", err)
	}

	return config, nil
}

// ConfigBasedResolver resolves aliases based on a configuration
type ConfigBasedResolver struct {
	Config *AliasConfig
}

// NewConfigBasedResolver creates a new ConfigBasedResolver
func NewConfigBasedResolver(config *AliasConfig) *ConfigBasedResolver {
	return &ConfigBasedResolver{Config: config}
}

// ResolveAlias resolves a package URL if its type is an alias
func (r *ConfigBasedResolver) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	aliasDef, ok := r.Config.Aliases[pURL.Type]
	if !ok {
		return pURL, false
	}

	qualifiers := pURL.Qualifiers.Map()

	if aliasDef.Base != "" && qualifiers[QualifierBaseURL] == "" {
		qualifiers[QualifierBaseURL] = aliasDef.Base
	}

	if aliasDef.TokenFromEnv != "" && qualifiers[QualifierTokenFromEnv] == "" {
		qualifiers[QualifierTokenFromEnv] = aliasDef.TokenFromEnv
	}

	return packageurl.PackageURL{
		Type:       aliasDef.Type,
		Namespace:  pURL.Namespace,
		Name:       pURL.Name,
		Version:    pURL.Version,
		Qualifiers: packageurl.QualifiersFromMap(qualifiers),
		Subpath:    pURL.Subpath,
	}, true
}

// DefaultResolver returns a resolver with the default configuration
func DefaultResolver() (AliasResolver, error) {
	loader, err := DefaultConfigLoader()
	if err != nil {
		return nil, err
	}

	config, err := loader.LoadConfig()
	if err != nil {
		return nil, err
	}

	return NewConfigBasedResolver(config), nil
}
