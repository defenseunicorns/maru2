// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIf(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		with            With
		previousOutputs CommandOutputs
		hasFailed       bool
		expected        bool
		expectedErr     string
	}{
		{
			name:     "empty",
			expected: true,
		},
		{
			name:      "empty after failure",
			hasFailed: true,
			expected:  false,
		},
		{
			name:     "failure()",
			input:    "failure()",
			expected: false,
		},
		{
			name:      "failure() after failure",
			input:     "failure()",
			hasFailed: true,
			expected:  true,
		},
		{
			name:     "always()",
			input:    "always()",
			expected: true,
		},
		{
			name:      "always() after failure",
			input:     "always()",
			hasFailed: true,
			expected:  true,
		},
		{
			name:     "always() always wins",
			input:    "always() and failure()",
			expected: true,
		},
		{
			name:     "based upon with",
			input:    `inputs.foo == "bar"`,
			with:     With{"foo": "bar"},
			expected: true,
		},
		{
			name:     "complex boolean expression (true)",
			input:    `(inputs.foo == "bar" && !failure()) || always()`,
			with:     With{"foo": "bar"},
			expected: true,
		},
		{
			name:     "complex boolean expression (false)",
			input:    `inputs.foo == "baz" && !failure()`,
			with:     With{"foo": "bar"},
			expected: false,
		},
		{
			name:     "access nested map in inputs",
			input:    `inputs.nested.value == "nested-value"`,
			with:     With{"nested": map[string]any{"value": "nested-value"}},
			expected: true,
		},
		{
			name:     "access nested map in inputs (false)",
			input:    `inputs.nested.value == "wrong-value"`,
			with:     With{"nested": map[string]any{"value": "nested-value"}},
			expected: false,
		},
		{
			name:            "access from outputs",
			input:           `from.step1.output == "step1-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "access from outputs (false)",
			input:           `from.step1.output == "wrong-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        false,
		},
		{
			name:            "access nested from outputs",
			input:           `from.step1.nested.value == "nested-value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"nested": map[string]any{"value": "nested-value"}}},
			expected:        true,
		},
		{
			name:            "missing step in from",
			input:           `from.missing.output == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:            "missing output in step",
			input:           `from.step1.missing == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:     "numeric comparison (equal)",
			input:    `inputs.num == 42`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "numeric comparison (not equal)",
			input:    `inputs.num != 43`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "numeric comparison (greater than)",
			input:    `inputs.num > 40`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "numeric comparison (less than)",
			input:    `inputs.num < 50`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "boolean value in inputs",
			input:    `inputs.enabled`,
			with:     With{"enabled": true},
			expected: true,
		},
		{
			name:     "boolean value in inputs (false)",
			input:    `!inputs.disabled`,
			with:     With{"disabled": false},
			expected: true,
		},
		{
			name:     "mathematical operation",
			input:    `(inputs.num + 8) == 50`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:        "syntax error in expression",
			input:       `inputs.foo == `,
			with:        With{"foo": "bar"},
			expectedErr: "unexpected token EOF (1:14)\n | inputs.foo == \n | .............^",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := If(tt.input).ShouldRun(tt.hasFailed, tt.with, tt.previousOutputs)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				require.False(t, actual)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}

	tests2 := []struct {
		name            string
		input           string
		with            With
		previousOutputs CommandOutputs
		hasFailed       bool
		expected        bool
		expectedErr     string
	}{
		{
			name:     "empty",
			expected: true,
		},
		{
			name:      "empty after failure",
			hasFailed: true,
			expected:  false,
		},
		{
			name:     "failure",
			input:    "${{ failure }}",
			expected: false,
		},
		{
			name:      "failure after command failure",
			input:     "${{ failure }}",
			hasFailed: true,
			expected:  true,
		},
		{
			name:     "always",
			input:    "${{ always }}",
			expected: true,
		},
		{
			name:      "always after failure",
			input:     "${{ always }}",
			hasFailed: true,
			expected:  true,
		},
		{
			name:     "always wins",
			input:    "${{ and always failure }}",
			expected: true,
		},
		{
			name:     "always wins2",
			input:    "${{ and failure always }}",
			expected: false, // what is this logic?
		},
		{
			name:     "always wins3",
			input:    "${{if and always failure}}true${{end}}", // this is so gross
			expected: true,
		},
	}

	for _, tt := range tests2 {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := If(tt.input).ShouldRunTemplate(tt.hasFailed, tt.with, tt.previousOutputs)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				require.False(t, actual)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}
}
