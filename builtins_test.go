// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteBuiltin(t *testing.T) {
	testCases := []struct {
		name          string
		step          Step
		with          With
		dry           bool
		expectedError string
		expectedLog   string
		expected      map[string]any
	}{
		{
			name: "echo builtin",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": "Hello, World!",
				},
			},
			with:          With{},
			dry:           false,
			expectedError: "",
			expectedLog:   "Hello, World!\n",
			expected:      map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "echo builtin dry run",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": "Hello, World!",
				},
			},
			with:          With{},
			dry:           true,
			expectedError: "",
			expectedLog:   "dry run",
			expected:      nil,
		},
		{
			name: "fetch builtin",
			step: Step{
				Uses: "builtin:fetch",
				With: With{
					"url":    "http://example.com",
					"method": "GET",
				},
			},
			with:          With{},
			dry:           true, // Use dry run to avoid actual HTTP requests
			expectedError: "",
			expectedLog:   "dry run",
			expected:      nil,
		},
		{
			name: "non-existent builtin",
			step: Step{
				Uses: "builtin:nonexistent",
			},
			with:          With{},
			dry:           false,
			expectedError: "builtin:nonexistent not found",
			expected:      nil,
		},
		{
			name: "echo builtin with invalid with",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": make(chan int),
				},
			},
			with:          With{},
			dry:           false,
			expectedError: "builtin:echo: decoding failed due to the following error(s):\n\n'Text' expected type 'string', got unconvertible type 'chan int', value:",
			expected:      nil,
		},
		{
			name: "fetch builtin with invalid with",
			step: Step{
				Uses: "builtin:fetch",
			},
			with:          With{},
			dry:           false,
			expectedError: "builtin:fetch: error executing request: Get \"\": unsupported protocol scheme \"\"",
			expected:      nil,
		},
		{
			name: "echo builtin with templated with",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": "${{ input \"greeting\" }}",
				},
			},
			with:          With{"greeting": "Hello from template"},
			dry:           false,
			expectedError: "",
			expectedLog:   "Hello from template\n",
			expected:      map[string]any{"stdout": "Hello from template"},
		},
		{
			name: "echo builtin with broken structure",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": []string{"not", "a", "string"}, // Text should be a string, not an array
				},
			},
			with:          With{},
			dry:           false,
			expectedError: "builtin:echo: decoding failed due to the following error(s):\n\n'Text' expected type 'string', got unconvertible type '[]string', value: '[not a string]'",
			expected:      nil,
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
				if tc.expected != nil {
					assert.Equal(t, tc.expected, result)
				}
			} else if strings.Contains(tc.expectedError, "got unconvertible type 'chan int'") {
				require.ErrorContains(t, err, tc.expectedError)
				assert.Nil(t, result)
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
