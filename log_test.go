// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

type errLexer struct {
	name string
}

var _ chroma.Lexer = (*errLexer)(nil)

func (l *errLexer) Config() *chroma.Config {
	return &chroma.Config{
		Name: l.name,
	}
}

func (l *errLexer) Tokenise(_ *chroma.TokeniseOptions, _ string) (chroma.Iterator, error) {
	return nil, fmt.Errorf("not implemented")
}

func (l *errLexer) SetRegistry(_ *chroma.LexerRegistry) chroma.Lexer {
	return l
}

func (l *errLexer) SetAnalyser(_ func(text string) float32) chroma.Lexer {
	return l
}

func (l *errLexer) AnalyseText(_ string) float32 {
	return 0.0
}

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
			expected: "  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m hello\x1b[0m\n",
			color:    true,
		},
		{
			name:     "multiline",
			script:   "echo hello\necho world\n\necho !",
			expected: "  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m hello\x1b[0m\n  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m world\x1b[0m\n  \x1b[38;5;189m\x1b[0m\n  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m !\x1b[0m\n",
			color:    true,
		},
		{
			name:     "simple shell",
			script:   "echo hello",
			expected: "echo hello\n",
			color:    false,
		},
		{
			name:     "multiline",
			script:   "echo hello\necho world\n\necho !",
			expected: "echo hello\necho world\n\necho !\n",
			color:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.color {
				t.Setenv("NO_COLOR", "true")
			}
			var buf strings.Builder
			printScript(log.New(&buf), "", tc.script)
			assert.Equal(t, tc.expected, buf.String(), "this test fails when run w/ `go test`, run w/ `make test` instead as that will use maru2, which uses a true shell env")
		})
	}

	curr := lexers.Get("shell")
	t.Cleanup(func() {
		lexers.Register(curr)
	})

	lexers.Register(&errLexer{name: "shell"}) // overrides shell lexer

	var buf strings.Builder
	printScript(log.New(&buf), "", "echo hello")
	assert.Equal(t, "  echo hello\n", buf.String())
}

func TestPrintBuiltin(t *testing.T) {
	testCases := []struct {
		name     string
		builtin  v0.With
		expected string
		color    bool
	}{
		{
			name:     "simple shell",
			builtin:  v0.With{"text": "hello"},
			expected: "\x1b[38;5;141mwith\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m\x1b[0m\n\x1b[38;5;189m  \x1b[0m\x1b[38;5;141mtext\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m \x1b[0m\x1b[38;5;189mhello\x1b[0m\x1b[38;5;189m\x1b[0m\n",
			color:    true,
		},
		{
			name:     "multiline",
			builtin:  v0.With{"text": "hello\nworld\n!"},
			expected: "\x1b[38;5;141mwith\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m\x1b[0m\n\x1b[38;5;189m  \x1b[0m\x1b[38;5;141mtext\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m \x1b[0m\x1b[38;5;189m|-\x1b[0m\x1b[38;5;240m\x1b[0m\n\x1b[38;5;240m    hello\x1b[0m\n\x1b[38;5;240m    world\x1b[0m\n\x1b[38;5;240m    !\x1b[0m\x1b[38;5;189m\x1b[0m\n",
			color:    true,
		},
		{
			name:    "simple shell",
			builtin: v0.With{"text": "hello"},
			expected: `with:
  text: hello
`,
			color: false,
		},
		{
			name:    "multiline",
			builtin: v0.With{"text": "hello\nworld\n!"},
			expected: `with:
  text: |-
    hello
    world
    !
`,
			color: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.color {
				t.Setenv("NO_COLOR", "true")
			}
			var buf strings.Builder
			printBuiltin(log.New(&buf), tc.builtin)
			assert.Equal(t, tc.expected, buf.String())
		})
	}

	curr := lexers.Get("yaml")
	t.Cleanup(func() {
		lexers.Register(curr)
	})

	lexers.Register(&errLexer{name: "yaml"}) // overrides yaml lexer

	var buf strings.Builder
	printBuiltin(log.New(&buf), v0.With{"text": "echo hello"})
	assert.Equal(t, `with:
  text: echo hello

`, buf.String())
}
