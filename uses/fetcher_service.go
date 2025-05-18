// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
)

// FetcherService creates and manages fetchers
type FetcherService struct {
	resolver AliasResolver
	fsys     afero.Fs
	cache    map[string]Fetcher
	mu       sync.RWMutex
}

// NewFetcherService creates a new FetcherService with custom resolver and filesystem
func NewFetcherService(resolver AliasResolver, fs afero.Fs) (*FetcherService, error) {
	if resolver == nil {
		var err error
		resolver, err = DefaultResolver()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize default resolver: %w", err)
		}
	}

	if fs == nil {
		fs = afero.NewOsFs()
	}

	return &FetcherService{
		resolver: resolver,
		fsys:     fs,
		cache:    make(map[string]Fetcher),
		mu:       sync.RWMutex{},
	}, nil
}

// GetFetcher returns a fetcher for the given URI
func (s *FetcherService) GetFetcher(uri *url.URL) (Fetcher, error) {
	cacheKey := uri.String()

	s.mu.RLock()
	if fetcher, ok := s.cache[cacheKey]; ok {
		s.mu.RUnlock()
		return fetcher, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if fetcher, ok := s.cache[cacheKey]; ok {
		return fetcher, nil
	}

	var fetcher Fetcher

	switch uri.Scheme {
	case "http", "https":
		fetcher = NewHTTPFetcher()
	case "pkg":
		pURL, err := packageurl.FromString(uri.String())
		if err != nil {
			return nil, err
		}

		resolvedPURL, isAlias := s.resolver.ResolveAlias(pURL)
		if isAlias {
			pURL = resolvedPURL
		}

		qualifiers := pURL.Qualifiers.Map()
		tokenEnv := qualifiers["token-from-env"]
		base := qualifiers["base"]

		switch pURL.Type {
		case packageurl.TypeGithub:
			fetcher, err = NewGitHubClient(base, tokenEnv)
		case packageurl.TypeGitlab:
			fetcher, err = NewGitLabClient(base, tokenEnv)
		default:
			return nil, fmt.Errorf("unsupported package type: %q", pURL.Type)
		}

		if err != nil {
			return nil, err
		}

	case "file":
		fetcher = NewLocalFetcher(s.fsys)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", uri.Scheme)
	}

	s.cache[cacheKey] = fetcher
	return fetcher, nil
}
