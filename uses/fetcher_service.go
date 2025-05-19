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
	mapper PackageAliasMapper
	client *http.Client
	fsys   afero.Fs
	cache  map[string]Fetcher
	mu     sync.RWMutex
}

// FetcherServiceOption is a function that configures a FetcherService
type FetcherServiceOption func(*FetcherService)

// WithPackageAliasMapper sets the package alias mapper to be used by the fetcher service
func WithPackageAliasMapper(mapper PackageAliasMapper) FetcherServiceOption {
	return func(s *FetcherService) {
		s.mapper = mapper
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

	if svc.mapper == nil {
		loader, err := config.DefaultConfigLoader()
		if err != nil {
			return nil, err
		}

		config, err := loader.LoadConfig()
		if err != nil {
			return nil, err
		}

		svc.mapper = NewConfigBasedPackageAliasMapper(config)
	}

	if svc.fsys == nil {
		svc.fsys = afero.NewOsFs()
	}

	return svc, nil
}

// FallbackAliasMapper returns the package alias mapper used by the fetcher service
func (s *FetcherService) FallbackAliasMapper(mapper PackageAliasMapper) *FallbackPackageAliasMapper {
	return NewFallbackPackageAliasMapper(mapper, s.mapper)
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
