// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

func TestPrintScript(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
	}{
		{
			name:     "simple shell",
			script:   "echo hello",
			expected: "$ echo hello\n",
		},
		{
			name:     "multiline",
			script:   "echo hello\necho world\n\necho !",
			expected: "$ echo hello\n$ echo world\n$ \n$ echo !\n",
		},
	}

	var buf strings.Builder

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			printScript(log.New(&buf), tc.script)
			require.Equal(t, tc.expected, ansi.Strip(buf.String()))
			buf.Reset()
		})
	}
}
