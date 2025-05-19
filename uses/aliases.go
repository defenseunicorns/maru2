// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
)

// ConfigBasedPackageAliasMapper maps package aliases based on a configuration
type ConfigBasedPackageAliasMapper struct {
	Config *config.Config
}

// NewConfigBasedPackageAliasMapper creates a new ConfigBasedPackageAliasMapper
func NewConfigBasedPackageAliasMapper(config *config.Config) *ConfigBasedPackageAliasMapper {
	return &ConfigBasedPackageAliasMapper{Config: config}
}

// ResolveAlias resolves a package URL if its type is an alias
func (r *ConfigBasedPackageAliasMapper) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	return MapBasedPackageAliasMapper(r.Config.Aliases).ResolveAlias(pURL)
}

// MapBasedPackageAliasMapper maps package aliases based on a map of aliases
type MapBasedPackageAliasMapper map[string]config.Alias

// ResolveAlias resolves a package URL if its type is an alias
func (r MapBasedPackageAliasMapper) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
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

// FallbackPackageAliasMapper maps package aliases using a list of mappers
type FallbackPackageAliasMapper struct {
	mappers []PackageAliasMapper
}

// ResolveAlias resolves a package URL if its type is an alias
func (m *FallbackPackageAliasMapper) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	for _, mapper := range m.mappers {
		if mapper == nil {
			continue
		}
		if resolvedPURL, isAlias := mapper.ResolveAlias(pURL); isAlias {
			return resolvedPURL, true
		}
	}
	return pURL, false
}

// NewFallbackPackageAliasMapper creates a new FallbackPackageAliasMapper
func NewFallbackPackageAliasMapper(mappers ...PackageAliasMapper) *FallbackPackageAliasMapper {
	return &FallbackPackageAliasMapper{mappers: mappers}
}
