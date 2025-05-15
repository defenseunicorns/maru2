// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/spf13/afero"
)

// LocalFetcher fetches a file from the local filesystem.
type LocalFetcher struct {
	fs afero.Fs
}

// NewLocalFetcher creates a new local fetcher
func NewLocalFetcher(fs afero.Fs) *LocalFetcher {
	return &LocalFetcher{fs}
}

// Fetch opens a file handle at the given location
func (f *LocalFetcher) Fetch(_ context.Context, uses string) (io.ReadCloser, error) {
	uri, err := url.Parse(uses)
	if err != nil {
		return nil, err
	}

	if uri.Scheme != "file" {
		return nil, fmt.Errorf("scheme is not \"file\"")
	}

	p := uri.Opaque
	return f.fs.Open(p)
}
