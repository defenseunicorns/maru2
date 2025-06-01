// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		verifyPURL     func(*testing.T, packageurl.PackageURL)
		verifyStore    func(*testing.T, Fetcher)
	}{
		{
			name:         "new service with defaults",
			expectedType: nil,
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
			verifyStore: func(t *testing.T, f Fetcher) {
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
			verifyStore: func(t *testing.T, f Fetcher) {
				// Even with pkg scheme, should return the storage directly, not wrapped
				_, ok := f.(*mockStorage)
				require.True(t, ok, "Expected a direct storage instance, not wrapped in StoreFetcher")
				assert.NotEqual(t, "*uses.StoreFetcher", fmt.Sprintf("%T", f))
			},
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
			verifyStore: func(t *testing.T, f Fetcher) {
				storeFetcher, ok := f.(*StoreFetcher)
				require.True(t, ok)
				assert.IsType(t, &HTTPClient{}, storeFetcher.Source)
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
			verifyStore: func(t *testing.T, f Fetcher) {
				storeFetcher, ok := f.(*StoreFetcher)
				require.True(t, ok)
				assert.IsType(t, &HTTPClient{}, storeFetcher.Source)
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

			if tc.name == "new service with defaults" {
				assert.NotNil(t, service.PkgAliases())
				assert.Empty(t, service.PkgAliases())
				assert.NotNil(t, service.client)
				assert.NotNil(t, service.fsys)
				assert.NotNil(t, service.fetcherCache)
				assert.Nil(t, service.storage)
				assert.Equal(t, config.DefaultFetchPolicy, service.policy)
				return
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

			if tc.verifyStore != nil {
				tc.verifyStore(t, fetcher)
			}
		})
	}
}
