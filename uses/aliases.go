package uses

import (
	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
)

func ResolveAlias(pURL packageurl.PackageURL, aliases map[string]config.Alias) (packageurl.PackageURL, bool) {
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
