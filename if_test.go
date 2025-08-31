// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
)

// cancelledContext returns a context that is already cancelled
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestIf(t *testing.T) {
	tests := []struct {
		name            string
		inputExpr       string
		with            schema.With
		previousOutputs CommandOutputs
		dry             bool
		err             error
		ctx             context.Context
		expected        bool
		expectedErr     string
	}{
		{
			name:     "empty",
			expected: true,
		},
		{
			name:     "empty after failure",
			err:      fmt.Errorf("i had a failure"),
			expected: false,
		},
		{
			name:      "failure",
			inputExpr: "failure()",
			expected:  false,
		},
		{
			name:      "failure after command failure",
			inputExpr: "failure()",
			err:       fmt.Errorf("the previous command failed"),
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
			err:       fmt.Errorf("the previous command failed"),
			expected:  true,
		},
		{
			name:      "always wins",
			inputExpr: "always() and failure()",
			expected:  true,
		},
		{
			name:      "based upon with",
			inputExpr: `input("foo") == "bar"`,
			with:      schema.With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "presets",
			inputExpr: `len(arch) > 0 && len(os) > 0 && indexOf(platform, "/") > 0`,
			expected:  true,
		},
		{
			name:      "complex boolean expression (true)",
			inputExpr: `(input("foo") == "bar" && !failure()) || always()`,
			with:      schema.With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "complex boolean expression (false)",
			inputExpr: `input("foo") == "baz" && !failure()`,
			with:      schema.With{"foo": "bar"},
			expected:  false,
		},
		{
			name:      "access nested map in inputs",
			inputExpr: `input("nested", "value") == "wrong-value"`,
			with:      schema.With{"nested": map[string]any{"value": "nested-value"}},
			expectedErr: `too many arguments to call input (1:1)
 | input("nested", "value") == "wrong-value"
 | ^`,
		},
		{
			name:      "input dne",
			inputExpr: `input("dne") == "foo"`,
			with:      schema.With{"bar": "baz"},
			expectedErr: `input "dne" does not exist in [bar] (1:1)
 | input("dne") == "foo"
 | ^`,
		},
		{
			name:            "access from outputs",
			inputExpr:       `from("step1", "output") == "step1-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "access from outputs (false)",
			inputExpr:       `from("step1", "output") == "wrong-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        false,
		},
		{
			name:            "access nested from outputs",
			inputExpr:       `from("step1", "nested", "value") == "nested-value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"nested": map[string]any{"value": "nested-value"}}},
			expectedErr: `too many arguments to call from (1:1)
 | from("step1", "nested", "value") == "nested-value"
 | ^`,
		},
		{
			name:            "missing step in from",
			inputExpr:       `from("missing", "output") == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expectedErr: `no outputs from step "missing" (1:1)
 | from("missing", "output") == "value"
 | ^`,
		},
		{
			name:            "missing output in step",
			inputExpr:       `from("step1", "missing") == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expectedErr: `no output "missing" from step "step1" (1:1)
 | from("step1", "missing") == "value"
 | ^`,
		},
		{
			name:      "numeric comparison (equal)",
			inputExpr: `input("num") == 42`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:      "numeric comparison (not equal)",
			inputExpr: `input("num") != 43`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:      "numeric comparison (greater than)",
			inputExpr: `input("num") > 40`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:      "numeric comparison (less than)",
			inputExpr: `input("num") < 50`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:      "boolean value in inputs",
			inputExpr: `input("enabled")`,
			with:      schema.With{"enabled": true},
			expected:  true,
		},
		{
			name:      "boolean value in inputs (false)",
			inputExpr: `!input("disabled")`,
			with:      schema.With{"disabled": false},
			expected:  true,
		},
		{
			name:      "mathematical operation",
			inputExpr: `(input("num") + 8) == 50`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:        "syntax error",
			inputExpr:   `input.foo == `,
			with:        schema.With{"foo": "bar"},
			expectedErr: "unexpected token EOF (1:13)\n | input.foo == \n | ............^",
		},
		{
			name:        "typo",
			inputExpr:   `nputs.foo == bar`,
			dry:         true,
			with:        schema.With{"foo": "bar"},
			expectedErr: "unknown name nputs (1:1)\n | nputs.foo == bar\n | ^",
		},
		{
			name:      "dry run",
			dry:       true,
			inputExpr: "true",
			expected:  false,
		},
		{
			name:      "cancelled",
			inputExpr: "cancelled()",
			ctx:       cancelledContext(),
			expected:  true,
		},
		{
			name:      "not cancelled",
			inputExpr: "cancelled()",
			ctx:       context.Background(),
			expected:  false,
		},
		{
			name:      "cancelled with failure",
			inputExpr: "cancelled() && failure()",
			ctx:       cancelledContext(),
			err:       fmt.Errorf("previous command failed"),
			expected:  true,
		},
		{
			name:      "cancelled with always",
			inputExpr: "cancelled() && always()",
			ctx:       cancelledContext(),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ShouldRun(tt.ctx, tt.inputExpr, tt.err, tt.with, tt.previousOutputs, tt.dry)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.False(t, actual)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}
