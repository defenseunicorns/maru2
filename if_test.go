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
			name:      "input dne returns nil",
			inputExpr: `input("dne") == nil`,
			with:      schema.With{"bar": "baz"},
			expected:  true,
		},
		{
			name:      "input dne in comparison returns false",
			inputExpr: `input("dne") == "foo"`,
			with:      schema.With{"bar": "baz"},
			expected:  false,
		},
		{
			name:      "input dne can be checked for nil",
			inputExpr: `input("dne") != nil`,
			with:      schema.With{"bar": "baz"},
			expected:  false,
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
			name:            "missing step in from returns nil",
			inputExpr:       `from("missing", "output") == nil`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        true,
		},
		{
			name:            "missing step in from comparison returns false",
			inputExpr:       `from("missing", "output") == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:            "missing output in step returns nil",
			inputExpr:       `from("step1", "missing") == nil`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        true,
		},
		{
			name:            "missing output in step comparison returns false",
			inputExpr:       `from("step1", "missing") == "value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:            "can check missing step output for nil",
			inputExpr:       `from("missing", "output") != nil`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
		},
		{
			name:            "can check missing output for nil",
			inputExpr:       `from("step1", "missing") != nil`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        false,
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
		{
			name:      "nil input in logical expression with explicit nil check",
			inputExpr: `input("missing") == nil || input("exists") == "value"`,
			with:      schema.With{"exists": "value"},
			expected:  true,
		},
		{
			name:            "nil input and nil output both nil",
			inputExpr:       `input("missing") == nil && from("missing", "output") == nil`,
			with:            schema.With{"exists": "value"},
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        true,
		},
		{
			name:            "nil input with existing output",
			inputExpr:       `input("missing") == nil && from("step1", "output") == "value"`,
			with:            schema.With{"exists": "value"},
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        true,
		},
		{
			name:            "complex expression with nil values",
			inputExpr:       `(input("missing") == nil && from("missing", "output") == nil) || always()`,
			with:            schema.With{"exists": "value"},
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "value"}},
			expected:        true,
		},
		{
			name:            "nil input with default fallback",
			inputExpr:       `input("missing") == nil && input("default") == "fallback"`,
			with:            schema.With{"default": "fallback"},
			previousOutputs: CommandOutputs{},
			expected:        true,
		},
		{
			name:            "mixed nil and non-nil inputs",
			inputExpr:       `input("exists") == "value" && input("missing") == nil`,
			with:            schema.With{"exists": "value"},
			previousOutputs: CommandOutputs{},
			expected:        true,
		},
		{
			name:            "nil output with step that exists but missing key",
			inputExpr:       `from("step1", "missing") == nil && from("step1", "exists") == "value"`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{"step1": map[string]any{"exists": "value"}},
			expected:        true,
		},
		{
			name:            "using nil in arithmetic expression should fail gracefully",
			inputExpr:       `input("missing") == nil || input("num") + 5 == 47`,
			with:            schema.With{"num": 42},
			previousOutputs: CommandOutputs{},
			expected:        true,
		},
		{
			name:            "chaining nil checks with logical operators",
			inputExpr:       `input("a") == nil && input("b") == nil && input("c") == "exists"`,
			with:            schema.With{"c": "exists"},
			previousOutputs: CommandOutputs{},
			expected:        true,
		},
		{
			name:            "nil from with failure condition",
			inputExpr:       `failure() || from("missing", "key") == nil`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{},
			err:             fmt.Errorf("previous step failed"),
			expected:        true,
		},
		{
			name:            "nil input overrides failure when always is used",
			inputExpr:       `input("missing") == nil && always()`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{},
			err:             fmt.Errorf("previous step failed"),
			expected:        true,
		},
		{
			name:            "complex nested nil checks",
			inputExpr:       `(input("missing1") == nil && input("missing2") == nil) || (from("step1", "missing") == nil && from("step2", "missing") == nil)`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{"step1": map[string]any{"exists": "value"}},
			expected:        true,
		},
		{
			name:            "nil check with string operations",
			inputExpr:       `input("missing") == nil && len(input("text")) > 0`,
			with:            schema.With{"text": "hello"},
			previousOutputs: CommandOutputs{},
			expected:        true,
		},
		{
			name:            "expression evaluates to nil should return false",
			inputExpr:       `input("missing")`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{},
			expected:        false,
		},
		{
			name:            "from expression evaluates to nil should return false",
			inputExpr:       `from("missing", "output")`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{},
			expected:        false,
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
