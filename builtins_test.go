// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestExecuteBuiltin(t *testing.T) {
	testCases := []struct {
		name          string
		step          v1.Step
		with          v1.With
		dry           bool
		expectedError string
		expectedLog   string
		expected      map[string]any
	}{
		{
			name: "echo builtin",
			step: v1.Step{
				Uses: "builtin:echo",
				With: v1.With{
					"text": "Hello, World!",
				},
			},
			with:        v1.With{},
			expectedLog: "Hello, World!\n",
			expected:    map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "echo builtin dry run",
			step: v1.Step{
				Uses: "builtin:echo",
				With: v1.With{
					"text": "Hello, World!",
				},
			},
			with:        v1.With{},
			dry:         true,
			expectedLog: "dry run",
		},
		{
			name: "fetch builtin",
			step: v1.Step{
				Uses: "builtin:fetch",
				With: v1.With{
					"url":    "http://example.com",
					"method": "GET",
				},
			},
			with:        v1.With{},
			dry:         true, // Use dry run to avoid actual HTTP requests
			expectedLog: "dry run",
		},
		{
			name: "non-existent builtin",
			step: v1.Step{
				Uses: "builtin:nonexistent",
			},
			with:          v1.With{},
			expectedError: "builtin:nonexistent not found",
		},
		{
			name: "echo builtin with invalid with",
			step: v1.Step{
				Uses: "builtin:echo",
				With: v1.With{
					"text": make(chan int),
				},
			},
			with:          v1.With{},
			expectedError: "builtin:echo: decoding failed due to the following error(s):\n\n'Text' expected type 'string', got unconvertible type 'chan int'",
		},
		{
			name: "fetch builtin with invalid with",
			step: v1.Step{
				Uses: "builtin:fetch",
			},
			with:          v1.With{},
			expectedError: "builtin:fetch: error executing request: Get \"\": unsupported protocol scheme \"\"",
		},
		{
			name: "echo builtin with templated with",
			step: v1.Step{
				Uses: "builtin:echo",
				With: v1.With{
					"text": "${{ input \"greeting\" }}",
				},
			},
			with:        v1.With{"greeting": "Hello from template"},
			expectedLog: "Hello from template\n",
			expected:    map[string]any{"stdout": "Hello from template"},
		},
		{
			name: "echo builtin with broken structure",
			step: v1.Step{
				Uses: "builtin:echo",
				With: v1.With{
					"text": []string{"not", "a", "string"}, // Text should be a string, not an array
				},
			},
			with:          v1.With{},
			expectedError: "builtin:echo: decoding failed due to the following error(s):\n\n'Text' expected type 'string', got unconvertible type '[]string'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			ctx := log.WithContext(t.Context(), log.New(&buf))

			result, err := ExecuteBuiltin(ctx, tc.step, tc.with, CommandOutputs{}, tc.dry)

			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			} else {
				require.EqualError(t, err, tc.expectedError)
				assert.Nil(t, result)
			}

			if tc.expectedLog != "" {
				assert.Contains(t, buf.String(), tc.expectedLog)
			}
		})
	}
}
