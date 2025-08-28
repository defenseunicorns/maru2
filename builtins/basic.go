// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package builtins provides built-in functions for maru2
package builtins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

// echo is a sample builtin to MVP execution
type echo struct {
	Text string `json:"text" jsonschema:"description=Text to echo"`
}

// Execute the builtin
func (b *echo) Execute(ctx context.Context) (map[string]any, error) {
	logger := log.FromContext(ctx)

	logger.Print(b.Text)
	return map[string]any{"stdout": b.Text}, nil
}

// fetch is a sample builtin to showcase configuration and schema gen
type fetch struct {
	URL     string            `json:"url"               jsonschema:"description=URL to fetch"`
	Method  string            `json:"method,omitempty"  jsonschema:"description=HTTP method to use"`
	Timeout string            `json:"timeout,omitempty" jsonschema:"description=Timeout for the request"`
	Headers map[string]string `json:"headers,omitempty" jsonschema:"description=HTTP headers to send"`

	parsedTimeout time.Duration
}

func (b *fetch) setDefaults() error {
	if b.Method == "" {
		b.Method = "GET"
	}

	b.parsedTimeout = 30 * time.Second
	if b.Timeout != "" {
		parsedTimeout, err := time.ParseDuration(b.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		b.parsedTimeout = parsedTimeout
	}
	return nil
}

// Execute the builtin
func (b *fetch) Execute(ctx context.Context) (map[string]any, error) {
	logger := log.FromContext(ctx)

	if err := b.setDefaults(); err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: b.parsedTimeout,
	}

	logger.Printf("Headers: %s", b.Headers)

	req, err := http.NewRequestWithContext(ctx, b.Method, b.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code %d got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	logger.Printf("Status: %s", resp.Status)
	logger.Printf("Content-Type: %s", resp.Header.Get("Content-Type"))
	logger.Printf("Content-Length: %d", len(body))

	if resp.Header.Get("Content-Type") == "application/json" {
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
			logger.Print("Response Body:")
			logger.Print(prettyJSON.String())
			return map[string]any{"body": string(body)}, nil
		}
	}

	logger.Print("Response Body:")
	logger.Print(string(body))

	return map[string]any{"body": string(body)}, nil
}
