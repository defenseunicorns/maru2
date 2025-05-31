// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"io"
	"net/url"
)

// StoreFetcher is a fetcher that wraps another fetcher and caches the results
// in a store according to the cache policy.
type StoreFetcher struct {
	Source Fetcher
	Store  *Store
}

// Fetch implements the Fetcher interface
func (f *StoreFetcher) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	key := uri.String()

	if exists, err := f.Store.Exists(key); err == nil && exists {
		rc, err := f.Store.Fetch(ctx, key)
		if err == nil {
			return rc, nil
		}
	}

	rc, err := f.Source.Fetch(ctx, uri)
	if err != nil {
		return nil, err
	}

	if err := f.Store.Store(rc, key); err != nil {
		return nil, err
	}

	return f.Store.Fetch(ctx, key)
}
