// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/defenseunicorns/maru2/config"
)

// StoreFetcher is a fetcher that wraps another fetcher and caches the results
// in a store according to the cache policy.
type StoreFetcher struct {
	Source Fetcher
	Store  Storage
	Policy config.FetchPolicy
}

// Fetch implements the Fetcher interface
func (f *StoreFetcher) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	switch f.Policy {
	case config.FetchPolicyNever:
		return f.Store.Fetch(ctx, uri)
	case config.FetchPolicyIfNotPresent:
		exists, err := f.Store.Exists(uri)
		if err != nil {
			return nil, err
		}
		if exists {
			return f.Store.Fetch(ctx, uri)
		}
		fallthrough
	case config.FetchPolicyAlways:
		rc, err := f.Source.Fetch(ctx, uri)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		if err := f.Store.Store(rc, uri); err != nil {
			return nil, err
		}

		return f.Store.Fetch(ctx, uri)
	default:
		return nil, fmt.Errorf("unsupported fetch policy: %s", f.Policy)
	}
}
