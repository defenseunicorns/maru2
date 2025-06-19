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
			name:        "based upon with failure map access",
			input:       `inputs.bar == "foo"`,
			with:        With{"foo": "bar"},
			expectedErr: "dne",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := If(tt.input).ShouldRun(t.Context(), tt.hasFailed, tt.with, tt.previousOutputs)

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
		name        string
		input       string
		hasFailed   bool
		expected    bool
		expectedErr string
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
			actual, err := If(tt.input).ShouldRunTemplate(t.Context(), tt.hasFailed)

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
