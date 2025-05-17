// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type mockResolver struct {
	resolveFunc func(packageurl.PackageURL) (packageurl.PackageURL, bool)
}

func (m *mockResolver) ResolveAlias(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
	return m.resolveFunc(pURL)
}

func TestFetcherService(t *testing.T) {
	t.Run("new service with defaults", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)
		require.NotNil(t, service)
		require.NotNil(t, service.resolver)
		require.NotNil(t, service.fileSystem)
	})

	t.Run("new service with custom config", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		resolver := &mockResolver{
			resolveFunc: func(pURL packageurl.PackageURL) (packageurl.PackageURL, bool) {
				return pURL, false
			},
		}

		service, err := NewFetcherService(resolver, fs)

		require.NoError(t, err)
		require.NotNil(t, service)
		require.Equal(t, resolver, service.resolver)
		require.Equal(t, fs, service.fileSystem)
	})

	t.Run("get http fetcher", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("https://example.com")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		fetcher, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)
		require.IsType(t, &HTTPFetcher{}, fetcher)
	})

	t.Run("get file fetcher", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		fetcher, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)
		require.IsType(t, &LocalFetcher{}, fetcher)
	})

	t.Run("get github fetcher", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("pkg:github/noxsios/vai")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		fetcher, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)
		require.IsType(t, &GitHubClient{}, fetcher)
	})

	t.Run("get gitlab fetcher", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("pkg:gitlab/noxsios/vai")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		fetcher, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)
		require.IsType(t, &GitLabClient{}, fetcher)
	})

	t.Run("get fetcher from previous URI", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		prev, err := url.Parse("pkg:github/noxsios/vai")
		require.NoError(t, err)

		fetcher, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)
		require.IsType(t, &GitHubClient{}, fetcher)
	})

	t.Run("caching", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("https://example.com")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		fetcher1, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)

		fetcher2, err := service.GetFetcher(uri, prev)
		require.NoError(t, err)

		require.Same(t, fetcher1, fetcher2, "fetchers should be the same instance due to caching")
	})

	t.Run("unsupported scheme", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("ftp://example.com")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		_, err = service.GetFetcher(uri, prev)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported scheme")
	})

	t.Run("unsupported package type", func(t *testing.T) {
		service, err := NewFetcherService(nil, nil)
		require.NoError(t, err)

		uri, err := url.Parse("pkg:unsupported/noxsios/vai")
		require.NoError(t, err)

		prev, err := url.Parse("file:///tmp")
		require.NoError(t, err)

		_, err = service.GetFetcher(uri, prev)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported type")
	})
}
