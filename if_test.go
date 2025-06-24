// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIf(t *testing.T) {
	tests := []struct {
		name                string
		inputExpr           string
		inputTemplate       string
		with                With
		previousOutputs     CommandOutputs
		dry                 bool
		hasFailed           bool
		expected            bool
		expectedExprErr     string
		expectedTemplateErr string
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
			name:          "failure",
			inputExpr:     "failure()",
			inputTemplate: "${{ failure }}",
			expected:      false,
		},
		{
			name:          "failure after command failure",
			inputExpr:     "failure()",
			inputTemplate: "${{ failure }}",
			hasFailed:     true,
			expected:      true,
		},
		{
			name:          "always",
			inputExpr:     "always()",
			inputTemplate: "${{ always }}",
			expected:      true,
		},
		{
			name:          "always after failure",
			inputExpr:     "always()",
			inputTemplate: "${{ always }}",
			hasFailed:     true,
			expected:      true,
		},
		{
			name:          "always wins",
			inputExpr:     "always() and failure()",
			inputTemplate: "${{ and always failure }}",
			expected:      true,
		},
		{
			name:          "always wins2 (template only)",
			inputTemplate: "${{ and failure always }}",
			expected:      false, // Logic note: 'always' doesn't short-circuit here because 'and' evaluates its first argument first
		},
		{
			name:          "always wins3 (template only)",
			inputTemplate: "${{if and always failure}}true${{end}}",
			expected:      true,
		},
		{
			name:          "based upon with",
			inputExpr:     `inputs.foo == "bar"`,
			inputTemplate: `${{ input "foo" | eq "bar" }}`,
			with:          With{"foo": "bar"},
			expected:      true,
		},
		{
			name:          "complex boolean expression (true)",
			inputExpr:     `(inputs.foo == "bar" && !failure()) || always()`,
			inputTemplate: `${{ or (and (input "foo" | eq "bar") (not failure)) always }}`,
			with:          With{"foo": "bar"},
			expected:      true,
		},
		{
			name:          "complex boolean expression (false)",
			inputExpr:     `inputs.foo == "baz" && !failure()`,
			inputTemplate: `${{ and (input "foo" | eq "baz") (not failure) }}`,
			with:          With{"foo": "bar"},
			expected:      false,
		},
		{
			name:          "access nested map in inputs",
			inputExpr:     `inputs.nested.value == "nested-value"`,
			inputTemplate: `${{ eq (index (input "nested") "value") "nested-value" }}`,
			with:          With{"nested": map[string]any{"value": "nested-value"}},
			expected:      true,
		},
		{
			name:          "access nested map in inputs (false)",
			inputExpr:     `inputs.nested.value == "wrong-value"`,
			inputTemplate: `${{ eq (index (input "nested") "value") "wrong-value" }}`,
			with:          With{"nested": map[string]any{"value": "nested-value"}},
			expected:      false,
		},
		{
			name:            "access from outputs",
			inputExpr:       `from.step1.output == "step1-output"`,
			inputTemplate:   `${{ from "step1" "output" | eq "step1-output" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "access from outputs (false)",
			inputExpr:       `from.step1.output == "wrong-output"`,
			inputTemplate:   `${{ from "step1" "output" | eq "wrong-output" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        false,
		},
		{
			name:            "access nested from outputs",
			inputExpr:       `from.step1.nested.value == "nested-value"`,
			inputTemplate:   `${{ eq (index (from "step1" "nested") "value") "nested-value" }}`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"nested": map[string]any{"value": "nested-value"}}},
			expected:        true,
		},
		{
			name:                "missing step in from",
			inputExpr:           `from.missing.output == "value"`,
			inputTemplate:       `${{ eq (from "missing" "output") "value" }}`,
			previousOutputs:     CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:            false,
			expectedTemplateErr: "template: should run:1:8: executing \"should run\" at <from \"missing\" \"output\">: error calling from: no outputs from step \"missing\"",
		},
		{
			name:                "missing output in step",
			inputExpr:           `from.step1.missing == "value"`,
			inputTemplate:       `${{ eq (from "step1" "missing") "value" }}`,
			previousOutputs:     CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:            false,
			expectedTemplateErr: "template: should run:1:8: executing \"should run\" at <from \"step1\" \"missing\">: error calling from: no output \"missing\" from step \"step1\"",
		},
		{
			name:          "numeric comparison (equal)",
			inputExpr:     `inputs.num == 42`,
			inputTemplate: `${{ input "num" | eq 42 }}`,
			with:          With{"num": 42},
			expected:      true,
		},
		{
			name:          "numeric comparison (not equal)",
			inputExpr:     `inputs.num != 43`,
			inputTemplate: `${{ input "num" | ne 43 }}`,
			with:          With{"num": 42},
			expected:      true,
		},
		{
			name:          "numeric comparison (greater than)",
			inputExpr:     `inputs.num > 40`,
			inputTemplate: `${{ gt (input "num") 40 }}`,
			with:          With{"num": 42},
			expected:      true,
		},
		{
			name:          "numeric comparison (less than)",
			inputExpr:     `inputs.num < 50`,
			inputTemplate: `${{ lt (input "num") 50 }}`,
			with:          With{"num": 42},
			expected:      true,
		},
		{
			name:          "boolean value in inputs",
			inputExpr:     `inputs.enabled`,
			inputTemplate: `${{ input "enabled" }}`,
			with:          With{"enabled": true},
			expected:      true,
		},
		{
			name:          "boolean value in inputs (false)",
			inputExpr:     `!inputs.disabled`,
			inputTemplate: `${{ input "disabled" | not }}`,
			with:          With{"disabled": false},
			expected:      true,
		},
		{
			name:          "mathematical operation",
			inputExpr:     `(inputs.num + 8) == 50`,
			inputTemplate: `${{ input "num" | add 8 | eq 50 }}`,
			with:          With{"num": 42},
			expected:      true,
		},
		{
			name:                "syntax error",
			inputExpr:           `inputs.foo == `,
			inputTemplate:       `${{ eq (input "foo") }}`,
			with:                With{"foo": "bar"},
			expectedExprErr:     "unexpected token EOF (1:14)\n | inputs.foo == \n | .............^",
			expectedTemplateErr: "template: should run:1:4: executing \"should run\" at <eq (input \"foo\")>: error calling eq: missing argument for comparison",
		},
		{
			name:                "typo",
			inputExpr:           `input.foo == bar`,
			dry:                 true,
			inputTemplate:       `${{ eq (inputs "foo") }}`,
			with:                With{"foo": "bar"},
			expectedExprErr:     "unknown name input (1:1)\n | input.foo == bar\n | ^",
			expectedTemplateErr: "template: should run:1: function \"inputs\" not defined",
		},
		{
			name:          "dry run",
			dry:           true,
			inputExpr:     "true",
			inputTemplate: "${{ true }}",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.inputExpr != "" || (tt.inputExpr == "" && tt.inputTemplate == "") {
				actual, err := If(tt.inputExpr).ShouldRun(tt.hasFailed, tt.with, tt.previousOutputs, tt.dry)

				if tt.expectedExprErr != "" {
					require.EqualError(t, err, tt.expectedExprErr)
					require.False(t, actual)
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.expected, actual)
				}
			}

			if tt.inputTemplate != "" {
				actual, err := If(tt.inputTemplate).ShouldRunTemplate(tt.hasFailed, tt.with, tt.previousOutputs, tt.dry)

				if tt.expectedTemplateErr != "" {
					require.EqualError(t, err, tt.expectedTemplateErr)
					require.False(t, actual)
				} else {
					require.NoError(t, err)
					require.Equal(t, tt.expected, actual)
				}
			}
		})
	}
}
