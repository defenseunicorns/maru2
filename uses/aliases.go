// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/package-url/packageurl-go"
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

// AliasResolver handles resolving package URL aliases
type AliasResolver interface {
	ResolveAlias(packageurl.PackageURL) (packageurl.PackageURL, bool)
}

// ConfigLoader loads configuration from a source
type ConfigLoader interface {
	LoadConfig() (*AliasConfig, error)
}

// FileSystemConfigLoader loads configuration from the file system
type FileSystemConfigLoader struct {
	FS         fs.FS
	ConfigPath string
}

// NewFileSystemConfigLoader creates a new FileSystemConfigLoader
func NewFileSystemConfigLoader(fsys fs.FS, configPath string) *FileSystemConfigLoader {
	return &FileSystemConfigLoader{
		FS:         fsys,
		ConfigPath: configPath,
	}
}

// DefaultConfigLoader returns a config loader that uses the default locations
func DefaultConfigLoader() (*FileSystemConfigLoader, error) {
	// Try to find config in standard locations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Use ~/.maru2/aliases.yaml as the default path
	configPath := filepath.Join(homeDir, ".maru2", "aliases.yaml")
	return NewFileSystemConfigLoader(os.DirFS("/"), configPath), nil
}

// LoadConfig loads the alias configuration from the file system
func (l *FileSystemConfigLoader) LoadConfig() (*AliasConfig, error) {
	// Start with empty config
	config := &AliasConfig{
		Aliases: map[string]AliasDefinition{},
	}

	// Make the path relative to the FS root
	relPath := l.ConfigPath
	if filepath.IsAbs(relPath) {
		// For os.DirFS("/"), we need to remove the leading slash
		relPath = relPath[1:]
	}

	// Try to open the file
	f, err := l.FS.Open(relPath)
	if err != nil {
		// If the file doesn't exist, return empty config
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to open alias config file: %w", err)
	}
	defer f.Close()

	// Read the file
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read alias config file: %w", err)
	}

	// Parse the YAML
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

	qualifiers := packageurl.QualifiersFromMap(pURL.Qualifiers.Map())
	qualifierMap := pURL.Qualifiers.Map()

	if aliasDef.Base != "" && qualifierMap["base"] == "" {
		qualifiers = append(qualifiers, packageurl.Qualifier{Key: "base", Value: aliasDef.Base})
	}

	if aliasDef.TokenFromEnv != "" && qualifierMap["token-from-env"] == "" {
		qualifiers = append(qualifiers, packageurl.Qualifier{Key: "token-from-env", Value: aliasDef.TokenFromEnv})
	}

	return packageurl.PackageURL{
		Type:       aliasDef.Type,
		Namespace:  pURL.Namespace,
		Name:       pURL.Name,
		Version:    pURL.Version,
		Qualifiers: qualifiers,
		Subpath:    pURL.Subpath,
	}, true
}

// DefaultResolver returns a resolver with the default configuration
func DefaultResolver() (AliasResolver, error) {
	// Get the default config loader
	loader, err := DefaultConfigLoader()
	if err != nil {
		return nil, err
	}

	// Load the configuration
	config, err := loader.LoadConfig()
	if err != nil {
		return nil, err
	}

	// Create a resolver with the loaded config
	return NewConfigBasedResolver(config), nil
}
