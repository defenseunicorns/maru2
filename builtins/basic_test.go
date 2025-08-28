// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/iotest"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinsMap(t *testing.T) {
	names := Names()

	assert.Len(t, names, len(_registrations))

	for _, name := range names {
		builtin := Get(name)
		assert.NotNil(t, builtin)
		assert.Implements(t, (*Builtin)(nil), builtin)
	}

	//nolint:testifylint
	assert.NotSame(t, Get("echo"), Get("echo"))

	assert.Nil(t, Get(""))
}

func TestBuiltinEcho(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
		case "/invalid-json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid": json}`))
		case "/partial-read-failure":
			w.Header().Set("Content-Type", "application/json")
			fr := iotest.ErrReader(fmt.Errorf("failed to read"))
			w.WriteHeader(http.StatusOK)
			_, _ = io.Copy(w, fr)
		case "/timeout":
			d, _ := time.ParseDuration(r.URL.Query().Get("in"))
			time.Sleep(d + time.Millisecond*100)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("plain text response"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	testCases := []struct {
		name          string
		fetch         fetch
		body          string
		expectedError string
	}{
		{
			name: "default method",
			fetch: fetch{
				URL: server.URL + "/text",
			},
			body: "plain text response",
		},
		{
			name: "404",
			fetch: fetch{
				URL: server.URL + "/404",
			},
			expectedError: "expected status code 200 got 404",
		},
		{
			name: "fetch json",
			fetch: fetch{
				URL:    server.URL + "/json",
				Method: http.MethodGet,
			},
			body: `{"message":"success"}`,
		},
		{
			name: "fetch text",
			fetch: fetch{
				URL:    server.URL + "/text",
				Method: http.MethodGet,
			},
			body: "plain text response",
		},
		{
			name: "fetch invalid json",
			fetch: fetch{
				URL:    server.URL + "/invalid-json",
				Method: http.MethodGet,
			},
			body: `{"invalid": json}`,
		},
		{
			name: "fail on partial body read",
			fetch: fetch{
				URL:    server.URL + "/partial-read-failure",
				Method: http.MethodGet,
			},
			expectedError: "partial",
		},
		{
			name: "with headers",
			fetch: fetch{
				URL:    server.URL + "/headers",
				Method: http.MethodGet,
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
		},
		{
			name: "with timeout",
			fetch: fetch{
				URL:     server.URL + "/timeout?in=1s",
				Method:  http.MethodGet,
				Timeout: "1s",
			},
			expectedError: "context deadline exceeded",
		},
		{
			name: "url does not exist",
			fetch: fetch{
				URL:    "http://localhost:123456",
				Method: http.MethodGet,
			},
			expectedError: "dial tcp: address 123456: invalid port",
		},
		{
			name: "invalid timeout format",
			fetch: fetch{
				URL:     server.URL + "/text",
				Method:  "GET",
				Timeout: "invalid",
			},
			expectedError: `invalid timeout: time: invalid duration "invalid"`,
		},
		{
			name: "empty timeout",
			fetch: fetch{
				URL:     server.URL + "/text",
				Method:  "GET",
				Timeout: "",
			},
			body: "plain text response",
		},
		{
			name: "invalid request",
			fetch: fetch{
				URL:    string([]byte{0x7f}),
				Method: "GET",
			},
			expectedError: `error creating request: parse "\x7f": net/url: invalid control character in URL`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			result, err := tc.fetch.Execute(ctx)

			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.body, result["body"])
			}
		})
	}
}

func TestBuiltinWackyStructs(t *testing.T) {
	wacky := Get("wacky-structs")
	assert.Implements(t, (*Builtin)(nil), wacky)

	out, err := wacky.Execute(t.Context())
	assert.Nil(t, out)
	require.Error(t, err)
}
