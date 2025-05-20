// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
)

func TestPrintScript(t *testing.T) {
	testCases := []struct {
		name     string
		script   string
		expected string
		color    bool
	}{
		{
			name:     "simple shell",
			script:   "echo hello",
			expected: "$ \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m hello\x1b[0m\n",
			color:    true,
		},
		{
			name:     "multiline",
			script:   "echo hello\necho world\n\necho !",
			expected: "$ \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m hello\x1b[0m\n$ \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m world\x1b[0m\n$ \x1b[38;5;189m\x1b[0m\n$ \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m !\x1b[0m\n",
			color:    true,
		},
		{
			name:     "simple shell",
			script:   "echo hello",
			expected: "$ echo hello\n",
			color:    false,
		},
		{
			name:   "multiline",
			script: "echo hello\necho world\n\necho !",
			expected: `$ echo hello
$ echo world
$ 
$ echo !
`,
			color: false,
		},
	}

	var buf strings.Builder

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.color {
				t.Setenv("NO_COLOR", "true")
			}
			printScript(log.New(&buf), tc.script)
			assert.Equal(t, tc.expected, buf.String())
			buf.Reset()
		})
	}
}

func TestPrintBuiltin(t *testing.T) {
	testCases := []struct {
		name     string
		builtin  With
		expected string
		color    bool
	}{
		{
			name:     "simple shell",
			builtin:  With{"text": "hello"},
			expected: "\x1b[38;5;141mwith\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m\x1b[0m\n\x1b[38;5;189m  \x1b[0m\x1b[38;5;141mtext\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m \x1b[0m\x1b[38;5;189mhello\x1b[0m\x1b[38;5;189m\x1b[0m\n",
			color:    true,
		},
		{
			name:     "multiline",
			builtin:  With{"text": "hello\nworld\n!"},
			expected: "\x1b[38;5;141mwith\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m\x1b[0m\n\x1b[38;5;189m  \x1b[0m\x1b[38;5;141mtext\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m \x1b[0m\x1b[38;5;189m|-\x1b[0m\x1b[38;5;240m\x1b[0m\n\x1b[38;5;240m    hello\x1b[0m\n\x1b[38;5;240m    world\x1b[0m\n\x1b[38;5;240m    !\x1b[0m\x1b[38;5;189m\x1b[0m\n",
			color:    true,
		},
		{
			name:    "simple shell",
			builtin: With{"text": "hello"},
			expected: `with:
  text: hello
`,
			color: false,
		},
		{
			name:    "multiline",
			builtin: With{"text": "hello\nworld\n!"},
			expected: `with:
  text: |-
    hello
    world
    !
`,
			color: false,
		},
	}

	var buf strings.Builder

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.color {
				t.Setenv("NO_COLOR", "true")
			}
			printBuiltin(log.New(&buf), tc.builtin)
			assert.Equal(t, tc.expected, buf.String())
			buf.Reset()
		})
	}
}
