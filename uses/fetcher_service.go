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
	resolver   AliasResolver
	fileSystem afero.Fs
	cache      map[string]Fetcher
	mu         sync.RWMutex
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
		resolver:   resolver,
		fileSystem: fs,
		cache:      make(map[string]Fetcher),
		mu:         sync.RWMutex{},
	}, nil
}

// GetFetcher returns a fetcher for the given URI and previous URI
func (s *FetcherService) GetFetcher(uri, previous *url.URL) (Fetcher, error) {
	cacheKey := uri.String() + "|" + previous.String()

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
	var err error

	switch uri.Scheme {
	case "http", "https":
		fetcher = NewHTTPFetcher()
	case "pkg":
		fetcher, err = s.handlePackageURL(uri)
	case "file":
		fetcher, err = s.handleFileScheme(previous)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", uri.Scheme)
	}

	if err != nil {
		return nil, err
	}

	s.cache[cacheKey] = fetcher
	return fetcher, nil
}

// handlePackageURL handles pkg: scheme URIs
func (s *FetcherService) handlePackageURL(uri *url.URL) (Fetcher, error) {
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
		return NewGitHubClient(base, tokenEnv)
	case packageurl.TypeGitlab:
		return NewGitLabClient(base, tokenEnv)
	default:
		return nil, fmt.Errorf("unsupported type: %q", pURL.Type)
	}
}

// handleFileScheme handles file: scheme URIs
func (s *FetcherService) handleFileScheme(previous *url.URL) (Fetcher, error) {
	switch previous.Scheme {
	case "file":
		return NewLocalFetcher(s.fileSystem), nil
	case "http", "https":
		return NewHTTPFetcher(), nil
	case "pkg":
		return s.handlePackageURL(previous)
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", previous.Scheme)
	}
}
