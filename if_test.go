// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/charmbracelet/log"
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
			name:     "empty expression with no error",
			expected: true,
		},
		{
			name:     "empty expression after failure",
			err:      fmt.Errorf("command failed"),
			expected: false,
		},
		{
			name:      "failure() returns false when no error",
			inputExpr: "failure()",
			expected:  false,
		},
		{
			name:      "failure() returns true after error",
			inputExpr: "failure()",
			err:       fmt.Errorf("command failed"),
			expected:  true,
		},
		{
			name:      "always() returns true",
			inputExpr: "always()",
			expected:  true,
		},
		{
			name:      "always() overrides failure",
			inputExpr: "always()",
			err:       fmt.Errorf("command failed"),
			expected:  true,
		},
		{
			name:      "always() short circuits other logic",
			inputExpr: "always() and failure()",
			expected:  true,
		},
		{
			name:      "cancelled() with cancelled context",
			inputExpr: "cancelled()",
			ctx:       cancelledContext(),
			expected:  true,
		},
		{
			name:      "cancelled() with active context",
			inputExpr: "cancelled()",
			ctx:       context.Background(),
			expected:  false,
		},
		{
			name:      "input() string comparison",
			inputExpr: `input("foo") == "bar"`,
			with:      schema.With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "input() numeric comparison",
			inputExpr: `input("num") == 42 && input("num") > 40 && input("num") < 50`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:      "input() boolean values",
			inputExpr: `input("enabled") && !input("disabled")`,
			with:      schema.With{"enabled": true, "disabled": false},
			expected:  true,
		},
		{
			name:      "input() missing key returns nil",
			inputExpr: `input("missing") == nil`,
			with:      schema.With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "input() missing key in comparison",
			inputExpr: `input("missing") == "value"`,
			with:      schema.With{"foo": "bar"},
			expected:  false,
		},
		{
			name:            "from() existing output",
			inputExpr:       `from("step1", "output") == "step1-output"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "from() missing step returns nil",
			inputExpr:       `from("missing", "output") == nil`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:            "from() missing key returns nil",
			inputExpr:       `from("step1", "missing") == nil`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"output": "step1-output"}},
			expected:        true,
		},
		{
			name:      "runtime environment variables",
			inputExpr: `len(arch) > 0 && len(os) > 0 && indexOf(platform, "/") > 0`,
			expected:  true,
		},
		{
			name:      "complex boolean with inputs",
			inputExpr: `(input("foo") == "bar" && !failure()) || always()`,
			with:      schema.With{"foo": "bar"},
			expected:  true,
		},
		{
			name:      "mathematical operations",
			inputExpr: `(input("num") + 8) == 50`,
			with:      schema.With{"num": 42},
			expected:  true,
		},
		{
			name:            "mixed nil checks and logic",
			inputExpr:       `input("missing") == nil && from("step1", "exists") == "value"`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{"step1": map[string]any{"exists": "value"}},
			expected:        true,
		},
		{
			name:        "syntax error",
			inputExpr:   `input.foo == `,
			with:        schema.With{"foo": "bar"},
			expectedErr: "unexpected token EOF (1:13)\n | input.foo == \n | ............^",
		},
		{
			name:      "invalid function call",
			inputExpr: `input("nested", "value") == "wrong-value"`,
			with:      schema.With{"nested": map[string]any{"value": "nested-value"}},
			expectedErr: `too many arguments to call input (1:1)
 | input("nested", "value") == "wrong-value"
 | ^`,
		},
		{
			name:            "invalid from function call",
			inputExpr:       `from("step1", "nested", "value") == "nested-value"`,
			previousOutputs: CommandOutputs{"step1": map[string]any{"nested": map[string]any{"value": "nested-value"}}},
			expectedErr: `too many arguments to call from (1:1)
 | from("step1", "nested", "value") == "nested-value"
 | ^`,
		},
		{
			name:      "dry run with true expression returns true",
			dry:       true,
			inputExpr: "true",
			expected:  true,
		},
		{
			name:      "dry run with false expression returns true (override)",
			dry:       true,
			inputExpr: "false",
			expected:  true,
		},
		{
			name:      "dry run with always() returns true",
			dry:       true,
			inputExpr: "always()",
			expected:  true,
		},
		{
			name:      "dry run with failure() after error returns true (override)",
			dry:       true,
			inputExpr: "failure()",
			err:       fmt.Errorf("command failed"),
			expected:  true,
		},
		{
			name:     "dry run with empty expression and no error returns true",
			dry:      true,
			expected: true,
		},
		{
			name:     "dry run with empty expression after failure returns false",
			dry:      true,
			err:      fmt.Errorf("command failed"),
			expected: false,
		},
		{
			name:            "expression evaluating to nil returns error",
			inputExpr:       `input("missing")`,
			with:            schema.With{},
			previousOutputs: CommandOutputs{},
			expectedErr:     "expression did not evaluate to a boolean",
		},
		{
			name:      "complex array operations",
			inputExpr: `len([1,2,3]) > 0`,
			with:      schema.With{},
			expected:  true,
		},
		{
			name:      "runtime environment variable operations",
			inputExpr: `len(platform) > 0 and len(os) > 0 and len(arch) > 0`,
			with:      schema.With{},
			expected:  true,
		},
		{
			name:            "complex nested operations with input and from",
			inputExpr:       `input("test") in ["a", "b", "c"] and from("step", "key") != nil`,
			with:            schema.With{"test": "a"},
			previousOutputs: CommandOutputs{"step": map[string]any{"key": "value"}},
			expected:        true,
		},
		{
			name:      "nil context with cancelled function",
			inputExpr: "cancelled()",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			if tt.ctx == nil {
				ctx = log.WithContext(t.Context(), log.New(io.Discard))
			}

			actual, err := ShouldRun(ctx, tt.inputExpr, tt.err, tt.with, tt.previousOutputs, tt.dry)

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
