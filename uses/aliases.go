// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
)

// ConfigBasedResolver resolves aliases based on a configuration
type ConfigBasedResolver struct {
	Config *config.Config
}

// NewConfigBasedResolver creates a new ConfigBasedResolver
func NewConfigBasedResolver(config *config.Config) *ConfigBasedResolver {
	return &ConfigBasedResolver{Config: config}
}

// ResolveAlias resolves a package URL if its type is an alias
func (r *ConfigBasedResolver) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	return MapBasedResolver(r.Config.Aliases).ResolveAlias(pURL)
}

// MapBasedResolver resolves aliases based on a map of aliases
type MapBasedResolver map[string]config.Alias

// ResolveAlias resolves a package URL if its type is an alias
func (r MapBasedResolver) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	aliasDef, ok := r[pURL.Type]
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
