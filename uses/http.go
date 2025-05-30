// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPFetcher fetches a file from a remote HTTP server
type HTTPFetcher struct {
	client *http.Client
}

// NewHTTPFetcher returns a new HTTPFetcher
func NewHTTPFetcher(client *http.Client) *HTTPFetcher {
	return &HTTPFetcher{client: client}
}

// Fetch performs a GET request using the default HTTP client
// against the provided raw URL string and returns the request body
func (f *HTTPFetcher) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("url is nil")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "maru2")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %q: %s", uri.String(), resp.Status)
	}
	return resp.Body, nil
}
