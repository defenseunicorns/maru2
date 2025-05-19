// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinsMap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		builtinName string
		expectFound bool
	}{
		{
			name:        "echo builtin exists",
			builtinName: "echo",
			expectFound: true,
		},
		{
			name:        "fetch builtin exists",
			builtinName: "fetch",
			expectFound: true,
		},
		{
			name:        "non-existent builtin",
			builtinName: "nonexistent",
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			builtin, found := Builtins[tc.builtinName]
			assert.Equal(t, tc.expectFound, found)

			if tc.expectFound {
				assert.NotNil(t, builtin)
			}
		})
	}
}

func TestBuiltinEcho(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "simple text",
			text:     "Hello, World!",
			expected: "Hello, World!\n",
		},
		{
			name:     "empty text",
			text:     "",
			expected: "\n",
		},
		{
			name:     "special characters",
			text:     "!@#$%^&*()",
			expected: "!@#$%^&*()\n",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			var buf bytes.Buffer
			logger := log.New(&buf)
			ctx := log.WithContext(t.Context(), logger)

			echo := echo{Text: tc.text}
			result, err := echo.Execute(ctx)

			require.NoError(t, err)
			assert.Equal(t, tc.text, result["stdout"])
			assert.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestBuiltinFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"success"}`))
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("plain text response"))
		case "/headers":
			for k, v := range r.Header {
				if k == "X-Custom-Header" && len(v) > 0 {
					_, _ = w.Write([]byte(v[0]))
					return
				}
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	testCases := []struct {
		name          string
		fetch         fetch
		expectedError bool
	}{
		{
			name: "fetch json",
			fetch: fetch{
				URL:    server.URL + "/json",
				Method: "GET",
			},
			expectedError: false,
		},
		{
			name: "fetch text",
			fetch: fetch{
				URL:    server.URL + "/text",
				Method: "GET",
			},
			expectedError: false,
		},
		{
			name: "default method",
			fetch: fetch{
				URL: server.URL + "/text",
			},
			expectedError: false,
		},
		{
			name: "with headers",
			fetch: fetch{
				URL:    server.URL + "/headers",
				Method: "GET",
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
			expectedError: false,
		},
		{
			name: "with timeout",
			fetch: fetch{
				URL:     server.URL + "/text",
				Method:  "GET",
				Timeout: "1s",
			},
			expectedError: false,
		},
		{
			name: "invalid url",
			fetch: fetch{
				URL:    "http://invalid-url-that-does-not-exist.example",
				Method: "GET",
			},
			expectedError: true,
		},
		{
			name: "invalid timeout format",
			fetch: fetch{
				URL:     server.URL + "/text",
				Method:  "GET",
				Timeout: "invalid",
			},
			expectedError: true,
		},
		{
			name: "empty timeout",
			fetch: fetch{
				URL:     server.URL + "/text",
				Method:  "GET",
				Timeout: "",
			},
			expectedError: false,
		},
		{
			name: "complex timeout",
			fetch: fetch{
				URL:     server.URL + "/text",
				Method:  "GET",
				Timeout: "1m30s",
			},
			expectedError: false,
		},
		{
			name: "invalid request",
			fetch: fetch{
				URL:    string([]byte{0x7f}),
				Method: "GET",
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			logger := log.New(io.Discard)
			ctx := log.WithContext(t.Context(), logger)

			result, err := tc.fetch.Execute(ctx)

			if tc.expectedError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Contains(t, result, "body")
			}
		})
	}
}
