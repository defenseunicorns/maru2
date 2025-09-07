// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestExecuteBuiltin(t *testing.T) {
	testCases := []struct {
		name            string
		step            v1.Step
		with            schema.With
		previousOutputs CommandOutputs
		dry             bool
		expectedError   string
		expectedLog     string
		expected        map[string]any
	}{
		{
			name: "echo builtin",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Hello, World!",
				},
			},
			with:        schema.With{},
			expectedLog: "Hello, World!\n",
			expected:    map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "echo builtin dry run",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Hello, World!",
				},
			},
			with:        schema.With{},
			dry:         true,
			expectedLog: "dry run",
		},
		{
			name: "fetch builtin",
			step: v1.Step{
				Uses: "builtin:fetch",
				With: schema.With{
					"url":    "http://example.com",
					"method": "GET",
				},
			},
			with:        schema.With{},
			dry:         true, // Use dry run to avoid actual HTTP requests
			expectedLog: "dry run",
		},
		{
			name: "non-existent builtin",
			step: v1.Step{
				Uses: "builtin:nonexistent",
			},
			with:          schema.With{},
			expectedError: "builtin:nonexistent not found",
		},
		{
			name: "echo builtin with invalid with",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": make(chan int),
				},
			},
			with:          schema.With{},
			expectedError: "builtin:echo: decoding failed due to the following error(s):\n\n'Text' expected type 'string', got unconvertible type 'chan int'",
		},
		{
			name: "fetch builtin with invalid with",
			step: v1.Step{
				Uses: "builtin:fetch",
			},
			with:          schema.With{},
			expectedError: "builtin:fetch: error executing request: Get \"\": unsupported protocol scheme \"\"",
		},
		{
			name: "echo builtin with templated with",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ input \"greeting\" }}",
				},
			},
			with:        schema.With{"greeting": "Hello from template"},
			expectedLog: "Hello from template\n",
			expected:    map[string]any{"stdout": "Hello from template"},
		},
		{
			name: "echo builtin with broken structure",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": []string{"not", "a", "string"}, // Text should be a string, not an array
				},
			},
			with:          schema.With{},
			expectedError: "builtin:echo: decoding failed due to the following error(s):\n\n'Text' expected type 'string', got unconvertible type '[]string'",
		},
		{
			name: "echo builtin with previous step output",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ from \"previous-step\" \"message\" }}",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"previous-step": map[string]any{
					"message": "Hello from previous step",
				},
			},
			expectedLog: "Hello from previous step\n",
			expected:    map[string]any{"stdout": "Hello from previous step"},
		},
		{
			name: "echo builtin with multiple previous step outputs",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ from \"step1\" \"greeting\" }} ${{ from \"step2\" \"name\" }}!",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"step1": map[string]any{
					"greeting": "Hello",
				},
				"step2": map[string]any{
					"name": "World",
				},
			},
			expectedLog: "Hello World!\n",
			expected:    map[string]any{"stdout": "Hello World!"},
		},
		{
			name: "echo builtin with nested output structure",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ from \"step1\" \"nested.value\" }}",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"step1": map[string]any{
					"nested.value": "nested output",
				},
			},
			expectedLog: "nested output\n",
			expected:    map[string]any{"stdout": "nested output"},
		},
		{
			name: "echo builtin with numeric output from previous step",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Count: ${{ from \"counter\" \"value\" }}",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"counter": map[string]any{
					"value": 42,
				},
			},
			expectedLog: "Count: 42\n",
			expected:    map[string]any{"stdout": "Count: 42"},
		},
		{
			name: "echo builtin with boolean output from previous step",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Success: ${{ from \"checker\" \"success\" }}",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"checker": map[string]any{
					"success": true,
				},
			},
			expectedLog: "Success: true\n",
			expected:    map[string]any{"stdout": "Success: true"},
		},
		{
			name: "fetch builtin with URL from previous step output",
			step: v1.Step{
				Uses: "builtin:fetch",
				With: schema.With{
					"url":    "${{ from \"config-step\" \"api-url\" }}",
					"method": "GET",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"config-step": map[string]any{
					"api-url": "http://example.com/api",
				},
			},
			dry:         true, // Use dry run to avoid actual HTTP requests
			expectedLog: "dry run",
		},
		{
			name: "echo builtin with missing step in previous outputs",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ from \"missing-step\" \"value\" }}",
				},
			},
			with:            schema.With{},
			previousOutputs: CommandOutputs{}, // Empty outputs to test missing step error
			expectedError:   "builtin:echo: template: expression evaluator:1:4: executing \"expression evaluator\" at <from \"missing-step\" \"value\">: error calling from: no outputs from step \"missing-step\"",
		},
		{
			name: "echo builtin with missing key in previous outputs",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ from \"existing-step\" \"missing-key\" }}",
				},
			},
			with: schema.With{},
			previousOutputs: CommandOutputs{
				"existing-step": map[string]any{
					"existing-key": "value",
				},
			},
			expectedError: "builtin:echo: template: expression evaluator:1:4: executing \"expression evaluator\" at <from \"existing-step\" \"missing-key\">: error calling from: no output \"missing-key\" from step \"existing-step\"",
		},
		{
			name: "echo builtin combining input and previous outputs",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ input \"prefix\" }}: ${{ from \"data-step\" \"result\" }}",
				},
			},
			with: schema.With{
				"prefix": "Result",
			},
			previousOutputs: CommandOutputs{
				"data-step": map[string]any{
					"result": "success",
				},
			},
			expectedLog: "Result: success\n",
			expected:    map[string]any{"stdout": "Result: success"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			ctx := log.WithContext(t.Context(), log.New(&buf))

			result, err := ExecuteBuiltin(ctx, tc.step, tc.with, tc.previousOutputs, tc.dry)

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
