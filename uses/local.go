// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

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
func (f *LocalFetcher) Fetch(_ context.Context, uri *url.URL) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("url is nil")
	}

	clone := *uri

	if clone.Scheme != "" && clone.Scheme != "file" {
		return nil, fmt.Errorf("scheme is not \"file\" or empty")
	}

	clone.Scheme = ""
	clone.RawQuery = ""
	p := clone.String()
	p = filepath.Clean(p)

	fileInfo, err := f.fs.Stat(p)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("read %s: is a directory", p)
	}

	return f.fs.Open(p)
}
