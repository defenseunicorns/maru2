// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
)

// FetcherService creates and manages fetchers
type FetcherService struct {
	resolver AliasResolver
	client   *http.Client
	fsys     afero.Fs
	cache    map[string]Fetcher
	mu       sync.RWMutex
}

// FetcherServiceOption is a function that configures a FetcherService
type FetcherServiceOption func(*FetcherService)

// WithFallbackResolver sets the alias resolver to be used by the fetcher service
func WithFallbackResolver(resolver AliasResolver) FetcherServiceOption {
	return func(s *FetcherService) {
		s.resolver = resolver
	}
}

// WithFS sets the filesystem to be used by the fetcher service
func WithFS(fs afero.Fs) FetcherServiceOption {
	return func(s *FetcherService) {
		s.fsys = fs
	}
}

// WithClient sets the HTTP client to be used by the fetcher service
func WithClient(client *http.Client) FetcherServiceOption {
	return func(s *FetcherService) {
		s.client = client
	}
}

// NewFetcherService creates a new FetcherService with custom resolver and filesystem
func NewFetcherService(opts ...FetcherServiceOption) (*FetcherService, error) {
	svc := &FetcherService{
		cache:  make(map[string]Fetcher),
		mu:     sync.RWMutex{},
		client: &http.Client{},
	}

	for _, opt := range opts {
		opt(svc)
	}

	if svc.resolver == nil {
		loader, err := config.DefaultConfigLoader()
		if err != nil {
			return nil, err
		}

		config, err := loader.LoadConfig()
		if err != nil {
			return nil, err
		}

		svc.resolver = NewConfigBasedResolver(config)
	}

	if svc.fsys == nil {
		svc.fsys = afero.NewOsFs()
	}

	return svc, nil
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
		fetcher = NewHTTPFetcher(s.client)
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
		tokenEnv := qualifiers[QualifierTokenFromEnv]
		base := qualifiers[QualifierBaseURL]

		switch pURL.Type {
		case packageurl.TypeGithub:
			fetcher, err = NewGitHubClient(s.client, base, tokenEnv)
		case packageurl.TypeGitlab:
			fetcher, err = NewGitLabClient(s.client, base, tokenEnv)
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
