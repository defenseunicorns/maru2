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

	// Tests using the template-based ShouldRunTemplate function
	templateTests := []struct {
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
			expected: false, // Logic note: 'always' doesn't short-circuit here because 'and' evaluates its first argument first
		},
		{
			name:     "always wins3",
			input:    "${{if and always failure}}true${{end}}",
			expected: true,
		},
		{
			name:     "based upon with",
			input:    `${{ input "foo" | eq "bar" }}`,
			with:     With{"foo": "bar"},
			expected: true,
		},
		{
			name:     "complex boolean expression (true)",
			input:    `${{ or (and (input "foo" | eq "bar") (not failure)) always }}`,
			with:     With{"foo": "bar"},
			expected: true,
		},
		{
			name:     "complex boolean expression (false)",
			input:    `${{ and (input "foo" | eq "baz") (not failure) }}`,
			with:     With{"foo": "bar"},
			expected: false,
		},
		{
			name:     "access nested map in inputs",
			input:    `${{ eq (index (input "nested") "value") "nested-value" }}`,
			with:     With{"nested": map[string]any{"value": "nested-value"}},
			expected: true,
		},
		{
			name:     "access nested map in inputs (false)",
			input:    `${{ eq (index (input "nested") "value") "wrong-value" }}`,
			with:     With{"nested": map[string]any{"value": "nested-value"}},
			expected: false,
		},
		{
			name:            "access from outputs",
			input:           `${{ from "step1" "output" | eq "step1-output" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "access from outputs (false)",
			input:           `${{ from "step1" "output" | eq "wrong-output" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        false,
		},
		{
			name:            "access nested from outputs",
			input:           `${{ eq (index (from "step1" "nested") "value") "nested-value" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"nested": map[string]any{"value": "nested-value"}}},
			expected:        true,
		},
		{
			name:            "missing step in from",
			input:           `${{ eq (from "missing" "output") "value" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expectedErr:     "template: should run:1:8: executing \"should run\" at <from \"missing\" \"output\">: error calling from: no outputs from step \"missing\"",
		},
		{
			name:            "missing output in step",
			input:           `${{ eq (from "step1" "missing") "value" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expectedErr:     "template: should run:1:8: executing \"should run\" at <from \"step1\" \"missing\">: error calling from: no output \"missing\" from step \"step1\"",
		},
		{
			name:     "numeric comparison (equal)",
			input:    `${{ input "num" | eq 42 }}`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "numeric comparison (not equal)",
			input:    `${{ input "num" | ne 43 }}`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "numeric comparison (greater than)",
			input:    `${{ gt (input "num") 40 }}`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "numeric comparison (less than)",
			input:    `${{ lt (input "num") 50 }}`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:     "boolean value in inputs",
			input:    `${{ input "enabled" }}`,
			with:     With{"enabled": true},
			expected: true,
		},
		{
			name:     "boolean value in inputs (false)",
			input:    `${{ input "disabled" | not }}`,
			with:     With{"disabled": false},
			expected: true,
		},
		{
			name:     "mathematical operation",
			input:    `${{ input "num" | add 8 | eq 50 }}`,
			with:     With{"num": 42},
			expected: true,
		},
		{
			name:        "syntax error in expression",
			input:       `${{ eq (input "foo") }}`,
			with:        With{"foo": "bar"},
			expectedErr: "template: should run:1:4: executing \"should run\" at <eq (input \"foo\")>: error calling eq: missing argument for comparison",
		},
	}

	for _, tt := range templateTests {
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
