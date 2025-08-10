// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"github.com/package-url/packageurl-go"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

// ResolveAlias resolves a package URL using the given aliases map
func ResolveAlias(pURL packageurl.PackageURL, aliases v0.AliasMap) (packageurl.PackageURL, bool) {
	aliasDef, ok := aliases[pURL.Type]
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
