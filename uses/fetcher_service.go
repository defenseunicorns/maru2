// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
)

// FetcherService creates and manages fetchers
type FetcherService struct {
	client       *http.Client
	fsys         afero.Fs
	fetcherCache map[string]Fetcher
	storage      Storage
	policy       FetchPolicy
	mu           sync.RWMutex
}

// FetcherServiceOption is a function that configures a FetcherService
type FetcherServiceOption func(*FetcherService)

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

// WithStorage sets the store to be used by the fetcher service
func WithStorage(store Storage) FetcherServiceOption {
	return func(s *FetcherService) {
		s.storage = store
	}
}

// WithFetchPolicy sets the fetch policy to be used by the fetcher service
func WithFetchPolicy(policy FetchPolicy) FetcherServiceOption {
	return func(s *FetcherService) {
		s.policy = policy
	}
}

// NewFetcherService creates a new FetcherService with custom resolver and filesystem
func NewFetcherService(opts ...FetcherServiceOption) (*FetcherService, error) {
	svc := &FetcherService{
		fetcherCache: make(map[string]Fetcher),
		policy:       DefaultFetchPolicy,
	}

	for _, opt := range opts {
		opt(svc)
	}

	if svc.fsys == nil {
		svc.fsys = afero.NewOsFs()
	}

	if svc.client == nil {
		svc.client = &http.Client{}
	}

	if svc.policy == FetchPolicyNever && svc.storage == nil {
		return nil, fmt.Errorf("store is not initialized")
	}

	// check the policy is valid
	if err := svc.policy.Set(svc.policy.String()); err != nil {
		return nil, err
	}

	return svc, nil
}

// GetFetcher returns a fetcher for the given URL
func (s *FetcherService) GetFetcher(uri *url.URL) (Fetcher, error) {
	if uri == nil {
		return nil, fmt.Errorf("uri cannot be nil")
	}

	if s.policy == FetchPolicyNever {
		return s.storage, nil
	}

	s.mu.RLock()
	fetcher, exists := s.fetcherCache[uri.String()]
	s.mu.RUnlock()
	if exists && fetcher != nil {
		return fetcher, nil
	}

	fetcher, err := s.createFetcher(uri)
	if err != nil {
		return nil, err
	}

	if s.storage != nil && uri.Scheme != "file" {
		fetcher = &StoreFetcher{
			Source: fetcher,
			Store:  s.storage,
			Policy: s.policy,
		}
	}

	s.mu.Lock()
	s.fetcherCache[uri.String()] = fetcher
	s.mu.Unlock()

	return fetcher, nil
}

// createFetcher creates a new fetcher for the given URI
func (s *FetcherService) createFetcher(uri *url.URL) (Fetcher, error) {
	var fetcher Fetcher

	switch uri.Scheme {
	case "http", "https":
		fetcher = NewHTTPClient(s.client)
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
	case "oci":
		var err error
		insecureSkipTLSVerify := uri.Query().Get(OCIQueryParamInsecureSkipTLSVerify) == "true"
		plainHTTP := uri.Query().Get(OCIQueryParamPlainHTTP) == "true"
		fetcher, err = NewOCIClient(s.client, insecureSkipTLSVerify, plainHTTP)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported scheme: %q", uri.Scheme)
	}

	return fetcher, nil
}
