// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	resolveFunc func(packageurl.PackageURL) (packageurl.PackageURL, bool)
}

func (m *mockResolver) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	return m.resolveFunc(pURL)
}

func TestFetcherService(t *testing.T) {
	testCases := []struct {
		name           string
		resolver       AliasResolver
		fs             afero.Fs
		uri            string
		expectedType   any
		expectedErr    string
		checkSameCache bool
	}{
		{
			name:         "new service with defaults",
			resolver:     nil,
			fs:           nil,
			expectedType: nil,
		},
		{
			name:         "new service with custom config",
			resolver:     &mockResolver{resolveFunc: func(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) { return pURL, false }},
			fs:           afero.NewMemMapFs(),
			expectedType: nil,
		},
		{
			name:         "get http fetcher",
			uri:          "https://example.com",
			expectedType: &HTTPFetcher{},
		},
		{
			name:         "get file fetcher",
			uri:          "file:///tmp",
			expectedType: &LocalFetcher{},
		},
		{
			name:         "get github fetcher",
			uri:          "pkg:github/noxsios/vai",
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
			expectedType:   &HTTPFetcher{},
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewFetcherService(
				WithFallbackResolver(tc.resolver),
				WithFS(tc.fs),
			)
			require.NoError(t, err)
			assert.NotNil(t, service)

			if tc.name == "new service with defaults" {
				require.NotNil(t, service.resolver)
				require.NotNil(t, service.fsys)
				return
			}

			if tc.name == "new service with custom config" {
				assert.Equal(t, tc.resolver, service.resolver)
				assert.Equal(t, tc.fs, service.fsys)
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
		})
	}
}
