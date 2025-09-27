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
	fsys afero.Fs
}

// NewLocalFetcher creates a new local fetcher
func NewLocalFetcher(fsys afero.Fs) *LocalFetcher {
	return &LocalFetcher{fsys}
}

// Fetch opens a file handle at the given location
func (f *LocalFetcher) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("uri is nil")
	}

	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	clone := *uri

	if clone.Scheme != "" && clone.Scheme != "file" {
		return nil, fmt.Errorf("scheme is not \"file\" or empty")
	}

	clone.Scheme = ""
	clone.RawQuery = ""
	p := clone.String()
	p = filepath.Clean(p)

	fileInfo, err := f.fsys.Stat(p)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("read %s: is a directory", p)
	}

	return f.fsys.Open(p)
}
