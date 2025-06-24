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
		inputExpr       string
		with            With
		previousOutputs CommandOutputs
		dry             bool
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
			name:      "failure",
			inputExpr: "failure()",
			expected:  false,
		},
		{
			name:      "failure after command failure",
			inputExpr: "failure()",
			hasFailed: true,
			expected:  true,
		},
		{
			name:      "always",
			inputExpr: "always()",
			expected:  true,
		},
		{
			name:      "always after failure",
			inputExpr: "always()",
			hasFailed: true,
			expected:  true,
		},
		{
			name:      "always wins",
			inputExpr: "always() and failure()",
			expected:  true,
		},
		{
			name:      "based upon with",
			inputExpr: `inputs.foo == "bar"`,
			with:      With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "complex boolean expression (true)",
			inputExpr: `(inputs.foo == "bar" && !failure()) || always()`,
			with:      With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "complex boolean expression (false)",
			inputExpr: `inputs.foo == "baz" && !failure()`,
			with:      With{"foo": "bar"},
			expected:  false,
		},
		{
			name:      "access nested map in inputs",
			inputExpr: `inputs.nested.value == "nested-value"`,
			with:      With{"nested": map[string]any{"value": "nested-value"}},
			expected:  true,
		},
		{
			name:      "access nested map in inputs (false)",
			inputExpr: `inputs.nested.value == "wrong-value"`,
			with:      With{"nested": map[string]any{"value": "nested-value"}},
			expected:  false,
		},
		{
			name:            "access from outputs",
			inputExpr:       `from.step1.output == "step1-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "access from outputs (false)",
			inputExpr:       `from.step1.output == "wrong-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        false,
		},
		{
			name:            "access nested from outputs",
			inputExpr:       `from.step1.nested.value == "nested-value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"nested": map[string]any{"value": "nested-value"}}},
			expected:        true,
		},
		{
			name:            "missing step in from",
			inputExpr:       `from.missing.output == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:            "missing output in step",
			inputExpr:       `from.step1.missing == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:      "numeric comparison (equal)",
			inputExpr: `inputs.num == 42`,
			with:      With{"num": 42},
			expected:  true,
		},
		{
			name:      "numeric comparison (not equal)",
			inputExpr: `inputs.num != 43`,
			with:      With{"num": 42},
			expected:  true,
		},
		{
			name:      "numeric comparison (greater than)",
			inputExpr: `inputs.num > 40`,
			with:      With{"num": 42},
			expected:  true,
		},
		{
			name:      "numeric comparison (less than)",
			inputExpr: `inputs.num < 50`,
			with:      With{"num": 42},
			expected:  true,
		},
		{
			name:      "boolean value in inputs",
			inputExpr: `inputs.enabled`,
			with:      With{"enabled": true},
			expected:  true,
		},
		{
			name:      "boolean value in inputs (false)",
			inputExpr: `!inputs.disabled`,
			with:      With{"disabled": false},
			expected:  true,
		},
		{
			name:      "mathematical operation",
			inputExpr: `(inputs.num + 8) == 50`,
			with:      With{"num": 42},
			expected:  true,
		},
		{
			name:        "syntax error",
			inputExpr:   `inputs.foo == `,
			with:        With{"foo": "bar"},
			expectedErr: "unexpected token EOF (1:14)\n | inputs.foo == \n | .............^",
		},
		{
			name:        "typo",
			inputExpr:   `nputs.foo == bar`,
			dry:         true,
			with:        With{"foo": "bar"},
			expectedErr: "unknown name nputs (1:1)\n | nputs.foo == bar\n | ^",
		},
		{
			name:      "dry run",
			dry:       true,
			inputExpr: "true",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := If(tt.inputExpr).ShouldRun(tt.hasFailed, tt.with, tt.previousOutputs, tt.dry)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				require.False(t, actual)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, actual)
			}
		})
	}
}
