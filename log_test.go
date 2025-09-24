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

	"github.com/defenseunicorns/maru2/schema"
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
		logLevel log.Level
	}{
		{
			name:     "simple shell",
			script:   "echo hello",
			expected: "  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m hello\x1b[0m\n",
			color:    true,
			logLevel: log.InfoLevel,
		},
		{
			name:     "multiline",
			script:   "echo hello\necho world\n\necho !",
			expected: "  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m hello\x1b[0m\n  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m world\x1b[0m\n  \x1b[38;5;189m\x1b[0m\n  \x1b[38;5;150mecho\x1b[0m\x1b[38;5;189m !\x1b[0m\n",
			color:    true,
			logLevel: log.InfoLevel,
		},
		{
			name:     "simple shell",
			script:   "echo hello",
			expected: "echo hello\n",
			color:    false,
			logLevel: log.InfoLevel,
		},
		{
			name:     "multiline",
			script:   "echo hello\necho world\n\necho !",
			expected: "echo hello\necho world\n\necho !\n",
			color:    false,
			logLevel: log.InfoLevel,
		},
		{
			name:     "info level - should print",
			script:   "echo hello",
			expected: "echo hello\n",
			color:    false,
			logLevel: log.InfoLevel,
		},
		{
			name:     "debug level - should print",
			script:   "echo hello",
			expected: "echo hello\n",
			color:    false,
			logLevel: log.DebugLevel,
		},
		{
			name:     "warn level - should not print",
			script:   "echo hello",
			expected: "",
			color:    false,
			logLevel: log.WarnLevel,
		},
		{
			name:     "error level - should not print",
			script:   "echo hello",
			expected: "",
			color:    false,
			logLevel: log.ErrorLevel,
		},
		{
			name:     "fatal level - should not print",
			script:   "echo hello",
			expected: "",
			color:    false,
			logLevel: log.FatalLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.color {
				t.Setenv("NO_COLOR", "true")
			}
			var buf strings.Builder
			logger := log.New(&buf)
			logger.SetLevel(tc.logLevel)
			printScript(logger, "", tc.script)
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
		builtin  schema.With
		expected string
		color    bool
		logLevel log.Level
	}{
		{
			name:     "simple shell",
			builtin:  schema.With{"text": "hello"},
			expected: "\x1b[38;5;141mwith\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m\x1b[0m\n\x1b[38;5;189m  \x1b[0m\x1b[38;5;141mtext\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m \x1b[0m\x1b[38;5;189mhello\x1b[0m\x1b[38;5;189m\x1b[0m\n",
			color:    true,
			logLevel: log.InfoLevel,
		},
		{
			name:     "multiline",
			builtin:  schema.With{"text": "hello\nworld\n!"},
			expected: "\x1b[38;5;141mwith\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m\x1b[0m\n\x1b[38;5;189m  \x1b[0m\x1b[38;5;141mtext\x1b[0m\x1b[38;5;189m:\x1b[0m\x1b[38;5;189m \x1b[0m\x1b[38;5;189m|-\x1b[0m\x1b[38;5;240m\x1b[0m\n\x1b[38;5;240m    hello\x1b[0m\n\x1b[38;5;240m    world\x1b[0m\n\x1b[38;5;240m    !\x1b[0m\x1b[38;5;189m\x1b[0m\n",
			color:    true,
			logLevel: log.InfoLevel,
		},
		{
			name:    "simple shell",
			builtin: schema.With{"text": "hello"},
			expected: `with:
  text: hello
`,
			color:    false,
			logLevel: log.InfoLevel,
		},
		{
			name:    "multiline",
			builtin: schema.With{"text": "hello\nworld\n!"},
			expected: `with:
  text: |-
    hello
    world
    !
`,
			color:    false,
			logLevel: log.InfoLevel,
		},
		{
			name:    "info level - should print",
			builtin: schema.With{"text": "hello"},
			expected: `with:
  text: hello
`,
			color:    false,
			logLevel: log.InfoLevel,
		},
		{
			name:    "debug level - should print",
			builtin: schema.With{"text": "hello"},
			expected: `with:
  text: hello
`,
			color:    false,
			logLevel: log.DebugLevel,
		},
		{
			name:     "warn level - should not print",
			builtin:  schema.With{"text": "hello"},
			expected: "",
			color:    false,
			logLevel: log.WarnLevel,
		},
		{
			name:     "error level - should not print",
			builtin:  schema.With{"text": "hello"},
			expected: "",
			color:    false,
			logLevel: log.ErrorLevel,
		},
		{
			name:     "fatal level - should not print",
			builtin:  schema.With{"text": "hello"},
			expected: "",
			color:    false,
			logLevel: log.FatalLevel,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.color {
				t.Setenv("NO_COLOR", "true")
			}
			var buf strings.Builder
			logger := log.New(&buf)
			logger.SetLevel(tc.logLevel)
			printBuiltin(logger, tc.builtin)
			assert.Equal(t, tc.expected, buf.String())
		})
	}

	curr := lexers.Get("yaml")
	t.Cleanup(func() {
		lexers.Register(curr)
	})

	lexers.Register(&errLexer{name: "yaml"}) // overrides yaml lexer

	var buf strings.Builder
	printBuiltin(log.New(&buf), schema.With{"text": "echo hello"})
	assert.Equal(t, `with:
  text: echo hello
`, buf.String())
}

func TestPrintBuiltinMarshalError(t *testing.T) {
	var buf strings.Builder
	logger := log.New(&buf)
	logger.SetLevel(log.DebugLevel)

	builtin := schema.With{"func": func() {}}

	printBuiltin(logger, builtin)

	output := buf.String()
	assert.Contains(t, output, "failed to marshal builtin")
}

func TestPrintGroup(t *testing.T) {
	syncTrue := func() bool {
		return true
	}
	syncFalse := func() bool {
		return false
	}

	currIsGitHubActions := isGitHubActions
	currIsGitLabCI := isGitLabCI

	restore := func() {
		isGitHubActions = currIsGitHubActions
		isGitLabCI = currIsGitLabCI
	}

	// set both to false so that this runs the same local and in GitHub CI
	isGitHubActions = syncFalse
	isGitLabCI = syncFalse

	t.Cleanup(restore)

	t.Run("default", func(t *testing.T) {
		// no task name
		closeGroup := printGroup(nil, "", "")
		assert.NotNil(t, closeGroup)
		assert.NotPanics(t, closeGroup)

		closeGroup = printGroup(nil, "default", "")
		assert.NotNil(t, closeGroup)
		assert.NotPanics(t, closeGroup)

		var buf strings.Builder
		closeGroup = printGroup(&buf, "default", "")
		assert.Equal(t, "", buf.String())
		closeGroup()
		assert.Equal(t, "", buf.String())
	})

	t.Run("github", func(t *testing.T) {
		var buf strings.Builder

		isGitHubActions = syncTrue
		t.Cleanup(func() {
			isGitHubActions = syncFalse
		})

		// regular execution with header
		closeGroup := printGroup(&buf, "default", "description")
		assert.Equal(t, "::group::default: description\n", buf.String())
		closeGroup()
		assert.Equal(t, "::group::default: description\n::endgroup::\n", buf.String())

		buf.Reset()

		// execution without header
		closeGroup = printGroup(&buf, "default", "")
		assert.Equal(t, "::group::default\n", buf.String())
		closeGroup()
		assert.Equal(t, "::group::default\n::endgroup::\n", buf.String())

		buf.Reset()

		// does not error if a nil writer is provided
		closeGroup = printGroup(nil, "default", "description")
		assert.Equal(t, "", buf.String())
		closeGroup()
		assert.Equal(t, "", buf.String())
	})

	t.Run("gitlab", func(t *testing.T) {
		var buf strings.Builder

		isGitLabCI = syncTrue
		t.Cleanup(func() {
			isGitLabCI = syncFalse
		})

		// execution without header (header gets set to taskName)
		closeGroup := printGroup(&buf, "default", "")
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdefault$`, buf.String())
		closeGroup()
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdefault\\e\[0Ksection_end:\d+:default\\r\\e\[0K$`, buf.String())

		buf.Reset()

		// execution with header (header is not changed)
		closeGroup = printGroup(&buf, "default", "description")
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdescription$`, buf.String())
		closeGroup()
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdescription\\e\[0Ksection_end:\d+:default\\r\\e\[0K$`, buf.String())

		buf.Reset()

		// does not error if a nil writer is provided
		closeGroup = printGroup(nil, "default", "description")
		assert.Equal(t, "", buf.String())
		closeGroup()
		assert.Equal(t, "", buf.String())
	})
}
