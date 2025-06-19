// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIf(t *testing.T) {
	tests := []struct {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := If(tt.input).ShouldRun(t.Context(), tt.hasFailed)

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
