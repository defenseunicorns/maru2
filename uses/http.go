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

// HTTPClient fetches a file from a remote HTTP server
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a client for fetching workflows over HTTP/HTTPS
//
// Provides a simple HTTP fetcher with proper user agent and context support
func NewHTTPClient(client *http.Client) *HTTPClient {
	return &HTTPClient{client: client}
}

// Fetch downloads workflow content from HTTP/HTTPS URLs
//
// Sets a maru2 user agent and handles standard HTTP error responses.
// Returns the response body as a ReadCloser for streaming
func (f *HTTPClient) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("uri is nil")
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
