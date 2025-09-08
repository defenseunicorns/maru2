// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"github.com/package-url/packageurl-go"

	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

// ResolvePkgAlias transforms package URLs using configured aliases
//
// Maps short package URL types to full package URLs with authentication
// and base URL configuration. Returns the resolved package URL and whether
// an alias was expanded
func ResolvePkgAlias(pURL packageurl.PackageURL, aliases v1.AliasMap) (packageurl.PackageURL, bool) {
	aliasDef, ok := aliases[pURL.Type]
	if !ok {
		return pURL, false
	}

	qualifiers := pURL.Qualifiers.Map()

	if aliasDef.BaseURL != "" && qualifiers[QualifierBaseURL] == "" {
		qualifiers[QualifierBaseURL] = aliasDef.BaseURL
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
