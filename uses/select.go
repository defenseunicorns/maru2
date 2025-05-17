// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/url"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
)

// defaultResolver is the resolver used by SelectFetcher
var defaultResolver AliasResolver

// SelectFetcher returns a Fetcher based on the URI scheme and previous scheme.
func SelectFetcher(uri, previous *url.URL) (Fetcher, error) {
	// Initialize the resolver if needed
	if defaultResolver == nil {
		var err error
		defaultResolver, err = DefaultResolver()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize alias resolver: %w", err)
		}
	}

	switch uri.Scheme {
	case "http", "https":
		return NewHTTPFetcher(), nil
	case "pkg":
		pURL, err := packageurl.FromString(uri.String())
		if err != nil {
			return nil, err
		}

		resolvedPURL, isAlias := defaultResolver.ResolveAlias(pURL)
		if isAlias {
			pURL = resolvedPURL
		}

		qualifiers := pURL.Qualifiers.Map()
		tokenEnv := qualifiers["token-from-env"]
		base := qualifiers["base"]

		switch pURL.Type {
		case "github":
			return NewGitHubClient(base, tokenEnv)
		case "gitlab":
			return NewGitLabClient(base, tokenEnv)
		default:
			return nil, fmt.Errorf("unsupported type: %q", pURL.Type)
		}
	case "file":
		switch previous.Scheme {
		case "file":
			return NewLocalFetcher(afero.NewOsFs()), nil
		case "http", "https":
			return NewHTTPFetcher(), nil
		case "pkg":
			pURL, err := packageurl.FromString(previous.String())
			if err != nil {
				return nil, err
			}
			resolvedPURL, isAlias := defaultResolver.ResolveAlias(pURL)
			if isAlias {
				pURL = resolvedPURL
			}

			qualifiers := pURL.Qualifiers.Map()
			tokenEnv := qualifiers["token-from-env"]
			base := qualifiers["base"]

			switch pURL.Type {
			case "github":
				return NewGitHubClient(base, tokenEnv)
			case "gitlab":
				return NewGitLabClient(base, tokenEnv)
			default:
				return nil, fmt.Errorf("unsupported type: %q", pURL.Type)
			}
		default:
			return nil, fmt.Errorf("unsupported scheme: %q", previous.Scheme)
		}
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", uri.Scheme)
	}
}
