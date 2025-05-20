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
	aliases map[string]config.Alias
	client  *http.Client
	fsys    afero.Fs
	cache   map[string]Fetcher
	mu      sync.RWMutex
}

// FetcherServiceOption is a function that configures a FetcherService
type FetcherServiceOption func(*FetcherService)

// WithAliases sets the aliases to be used by the fetcher service
func WithAliases(aliases map[string]config.Alias) FetcherServiceOption {
	return func(s *FetcherService) {
		s.aliases = aliases
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
		mu:    sync.RWMutex{},
		cache: make(map[string]Fetcher),
	}

	for _, opt := range opts {
		opt(svc)
	}

	if svc.aliases == nil {
		loader, err := config.DefaultConfigLoader()
		if err != nil {
			return nil, err
		}

		config, err := loader.LoadConfig()
		if err != nil {
			return nil, err
		}

		svc.aliases = config.Aliases
	}

	if svc.fsys == nil {
		svc.fsys = afero.NewOsFs()
	}

	if svc.client == nil {
		svc.client = &http.Client{}
	}

	return svc, nil
}

// PkgAliases returns the aliases used by the fetcher service
func (s *FetcherService) PkgAliases() map[string]config.Alias {
	return s.aliases
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
