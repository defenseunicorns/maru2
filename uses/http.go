// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
func (f *HTTPFetcher) Fetch(ctx context.Context, raw string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "maru2")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: %s", raw, resp.Status)
	}
	return resp.Body, nil
}
