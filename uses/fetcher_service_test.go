// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/config"
)

func TestFetcherService(t *testing.T) {
	// Helper function to create a mock storage for testing
	createMockStorage := func(content string) *mockStorage {
		return &mockStorage{
			fetchFunc: func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(content)), nil
			},
			existsFunc: func(_ *url.URL) (bool, error) {
				return true, nil
			},
			storeFunc: func(_ io.Reader, _ *url.URL) error {
				return nil
			},
		}
	}

	testCases := []struct {
		name           string
		opts           []FetcherServiceOption
		uri            string
		expectedType   any
		expectedErr    string
		checkSameCache bool
		verifyService  func(t *testing.T, s *FetcherService)
		verifyFetcher  func(t *testing.T, f Fetcher)
	}{
		{
			name:         "new service with defaults",
			uri:          "https://example.com",
			expectedType: &HTTPClient{},
			verifyService: func(t *testing.T, s *FetcherService) {
				assert.NotNil(t, s.PkgAliases())
				assert.Empty(t, s.PkgAliases())
				assert.NotNil(t, s.client)
				assert.NotNil(t, s.fsys)
				assert.NotNil(t, s.fetcherCache)
				assert.Nil(t, s.storage)
				assert.Equal(t, config.DefaultFetchPolicy, s.policy)
			},
		},
		{
			name:         "new service with fs",
			uri:          "https://example.com",
			expectedType: &HTTPClient{},
			opts: []FetcherServiceOption{
				WithFS(afero.NewMemMapFs()),
			},
			verifyService: func(t *testing.T, s *FetcherService) {
				assert.IsType(t, afero.NewMemMapFs(), s.fsys)
			},
		},
		{
			name: "new service with client",
			opts: []FetcherServiceOption{
				WithClient(&http.Client{Timeout: 10 * time.Second}),
			},
			uri:          "https://example.com",
			expectedType: &HTTPClient{},
			verifyService: func(t *testing.T, s *FetcherService) {
				assert.Equal(t, 10*time.Second, s.client.Timeout)
			},
			verifyFetcher: func(t *testing.T, f Fetcher) {
				assert.IsType(t, &HTTPClient{}, f)
				assert.Equal(t, 10*time.Second, f.(*HTTPClient).client.Timeout)
			},
		},
		{
			name: "new service with aliases",
			opts: []FetcherServiceOption{
				WithAliases(map[string]config.Alias{
					"test": {Type: "github", Base: "https://example.com"},
				}),
			},
			uri:         "pkg:test/defenseunicorns/maru2",
			expectedErr: "unsupported package type: \"test\"", // TODO: really gotta figure out when aliases should be applied
		},
		{
			name:         "get http fetcher",
			uri:          "https://example.com",
			expectedType: &HTTPClient{},
		},
		{
			name:         "get file fetcher",
			uri:          "file:///tmp",
			expectedType: &LocalFetcher{},
		},
		{
			name:         "get github fetcher",
			uri:          "pkg:github/defenseunicorns/maru2",
			expectedType: &GitHubClient{},
		},
		{
			name:         "get gitlab fetcher",
			uri:          "pkg:gitlab/noxsios/vai",
			expectedType: &GitLabClient{},
		},
		{
			name:           "caching",
			uri:            "https://example.com",
			expectedType:   &HTTPClient{},
			checkSameCache: true,
		},
		{
			name:        "unsupported scheme",
			uri:         "ftp://example.com",
			expectedErr: `unsupported scheme: "ftp"`,
		},
		{
			name:        "unsupported package type",
			uri:         "pkg:unsupported/noxsios/vai",
			expectedErr: `unsupported package type: "unsupported"`,
		},
		{
			name:        "with FetchPolicyNever without storage",
			opts:        []FetcherServiceOption{WithFetchPolicy(config.FetchPolicyNever)},
			uri:         "https://example.com",
			expectedErr: "store is not initialized",
		},
		{
			name: "with FetchPolicyNever with storage",
			opts: []FetcherServiceOption{
				WithFetchPolicy(config.FetchPolicyNever),
				WithStorage(createMockStorage("stored content")),
			},
			uri:          "https://example.com",
			expectedType: &mockStorage{},
			verifyFetcher: func(t *testing.T, f Fetcher) {
				store, ok := f.(*mockStorage)
				require.True(t, ok)

				// Test the mock storage directly
				uri, err := url.Parse("https://example.com")
				require.NoError(t, err)

				rc, err := store.Fetch(t.Context(), uri)
				require.NoError(t, err)

				content, err := io.ReadAll(rc)
				require.NoError(t, err)
				assert.Equal(t, "stored content", string(content))
			},
		},
		{
			name: "with FetchPolicyNever with storage - pkg scheme",
			opts: []FetcherServiceOption{
				WithFetchPolicy(config.FetchPolicyNever),
				WithStorage(createMockStorage("stored content")),
			},
			uri:          "pkg:github/defenseunicorns/maru2",
			expectedType: &mockStorage{},
		},
		{
			name: "with FetchPolicyAlways with storage - file scheme",
			opts: []FetcherServiceOption{
				WithFetchPolicy(config.FetchPolicyAlways),
				WithStorage(createMockStorage("stored content")),
			},
			uri:          "file:///tmp/example.txt",
			expectedType: &LocalFetcher{},
		},
		{
			name: "with FetchPolicyAlways with storage - http scheme",
			opts: []FetcherServiceOption{
				WithFetchPolicy(config.FetchPolicyAlways),
				WithStorage(createMockStorage("stored content")),
			},
			uri:          "https://example.com",
			expectedType: &StoreFetcher{},
			verifyFetcher: func(t *testing.T, f Fetcher) {
				storeFetcher, ok := f.(*StoreFetcher)
				require.True(t, ok)
				assert.IsType(t, &HTTPClient{}, storeFetcher.Source)
				assert.IsType(t, &mockStorage{}, storeFetcher.Store)
				assert.Equal(t, config.FetchPolicyAlways, storeFetcher.Policy)
			},
		},
		{
			name: "with FetchPolicyIfNotPresent with storage",
			opts: []FetcherServiceOption{
				WithFetchPolicy(config.FetchPolicyIfNotPresent),
				WithStorage(createMockStorage("stored content")),
			},
			uri:          "https://example.com",
			expectedType: &StoreFetcher{},
			verifyFetcher: func(t *testing.T, f Fetcher) {
				storeFetcher, ok := f.(*StoreFetcher)
				require.True(t, ok)
				assert.IsType(t, &HTTPClient{}, storeFetcher.Source)
				assert.IsType(t, &mockStorage{}, storeFetcher.Store)
				assert.Equal(t, config.FetchPolicyIfNotPresent, storeFetcher.Policy)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewFetcherService(tc.opts...)

			if tc.expectedErr != "" {
				if err == nil {
					// Try fetcher creation if service creation worked but should fail later
					if uri, parseErr := url.Parse(tc.uri); parseErr == nil {
						_, err = service.GetFetcher(uri)
					}
				}
				require.EqualError(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, service)

			if tc.verifyService != nil {
				tc.verifyService(t, service)
			}

			uri, err := url.Parse(tc.uri)
			require.NoError(t, err)

			fetcher, err := service.GetFetcher(uri)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.IsType(t, tc.expectedType, fetcher)

			if tc.checkSameCache {
				fetcher2, err := service.GetFetcher(uri)
				require.NoError(t, err)
				assert.Same(t, fetcher, fetcher2, "fetchers should be the same instance due to caching")
			}

			if tc.verifyFetcher != nil {
				tc.verifyFetcher(t, fetcher)
			}
		})
	}
}
