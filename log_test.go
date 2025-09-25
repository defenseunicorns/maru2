// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
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

	// reset state of checks to be "blank" after tests are done, these functions must EXACTLY match their counterparts
	t.Cleanup(func() {
		isGitHubActions = sync.OnceValue(func() bool {
			return os.Getenv(GitHubActionsEnvVar) == "true"
		})
		isGitLabCI = sync.OnceValue(func() bool {
			return os.Getenv(GitLabCIEnvVar) == "true"
		})
	})

	t.Run("env vars", func(t *testing.T) {
		t.Setenv(GitHubActionsEnvVar, "true")
		assert.True(t, isGitHubActions())
		t.Setenv(GitHubActionsEnvVar, "false")
		assert.True(t, isGitHubActions())

		t.Setenv(GitLabCIEnvVar, "true")
		assert.True(t, isGitLabCI())
		t.Setenv(GitLabCIEnvVar, "false")
		assert.True(t, isGitLabCI())
	})

	// set both to false so that this runs the same local and in GitHub CI
	isGitHubActions = syncFalse
	isGitLabCI = syncFalse

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
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdefault\n$`, buf.String())
		closeGroup()
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdefault\n\\e\[0Ksection_end:\d+:default\\r\\e\[0K\n$`, buf.String())

		buf.Reset()

		// execution with header (header is not changed)
		closeGroup = printGroup(&buf, "default", "description")
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdescription\n$`, buf.String())
		closeGroup()
		assert.Regexp(t, `^\\e\[0Ksection_start:\d+:default\[collapsed=true\]\\r\\e\[0Kdescription\n\\e\[0Ksection_end:\d+:default\\r\\e\[0K\n$`, buf.String())

		buf.Reset()

		// does not error if a nil writer is provided
		closeGroup = printGroup(nil, "default", "description")
		assert.Equal(t, "", buf.String())
		closeGroup()
		assert.Equal(t, "", buf.String())
	})
}

func TestDetailedTaskList(t *testing.T) {
	t.Setenv("NO_COLOR", "true") // format matters more than ensuring colors are correct

	testCases := []struct {
		name     string
		workflow v1.Workflow
		expected []string
	}{
		{
			name: "basic workflow with tasks",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Description: "Default task",
						Steps:       []v1.Step{{Run: "echo hello"}},
					},
					"test": v1.Task{
						Description: "Test task",
						Steps:       []v1.Step{{Run: "echo test"}},
					},
				},
			},
			expected: []string{
				"   default # Default task",
				"   test    # Test task   ",
			},
		},
		{
			name: "workflow with inputs",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"echo": v1.Task{
						Description: "Echo task with inputs",
						Inputs: v1.InputMap{
							"text": v1.InputParameter{
								Default: "default-value",
							},
							"required-param": v1.InputParameter{
								Required: func() *bool { b := true; return &b }(),
							},
							"optional-param": v1.InputParameter{
								Required: func() *bool { b := false; return &b }(),
							},
						},
						Steps: []v1.Step{{Run: "echo ${{ input \"text\" }}"}},
					},
				},
			},
			expected: []string{
				"   echo -w optional-param= -w required-param= -w text='default-value' # Echo task with inputs",
			},
		},
		{
			name: "empty workflow",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{},
			},
			expected: []string{},
		},
		{
			name: "workflow with env defaults",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"env-task": v1.Task{
						Description: "Task with env default",
						Inputs: v1.InputMap{
							"value": v1.InputParameter{
								Default:        "fallback",
								DefaultFromEnv: "MY_ENV_VAR",
							},
						},
						Steps: []v1.Step{{Run: "echo ${{ input \"value\" }}"}},
					},
				},
			},
			expected: []string{
				"   env-task -w value=\"${MY_ENV_VAR:-fallback}\" # Task with env default",
			},
		},
		{
			name: "task ordering",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"zzz-last": v1.Task{
						Description: "Should appear after default",
						Steps:       []v1.Step{{Run: "echo last"}},
					},
					"default": v1.Task{
						Description: "Default task should be first",
						Steps:       []v1.Step{{Run: "echo default"}},
					},
					"aaa-first": v1.Task{
						Description: "Should appear after default but before zzz",
						Steps:       []v1.Step{{Run: "echo first"}},
					},
				},
			},
			expected: []string{
				"   default   # Default task should be first              ",
				"   aaa-first # Should appear after default but before zzz",
				"   zzz-last  # Should appear after default               ",
			},
		},
		{
			name: "tasks without descriptions",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"no-desc": v1.Task{
						Steps: []v1.Step{{Run: "echo no description"}},
					},
					"with-desc": v1.Task{
						Description: "Has description",
						Steps:       []v1.Step{{Run: "echo with description"}},
					},
				},
			},
			expected: []string{
				"   no-desc                    ",
				"   with-desc # Has description",
			},
		},
		{
			name: "nil and empty inputs",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"nil-inputs": v1.Task{
						Description: "Task with nil inputs",
						Inputs:      nil,
						Steps:       []v1.Step{{Run: "echo test"}},
					},
					"empty-inputs": v1.Task{
						Description: "Task with empty inputs",
						Inputs:      v1.InputMap{},
						Steps:       []v1.Step{{Run: "echo test"}},
					},
				},
			},
			expected: []string{
				"   empty-inputs # Task with empty inputs",
				"   nil-inputs   # Task with nil inputs  ",
			},
		},
		{
			name: "input ordering",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"ordered-test": v1.Task{
						Description: "Test ordering",
						Inputs: v1.InputMap{
							"zebra": v1.InputParameter{Default: "z"},
							"alpha": v1.InputParameter{Default: "a"},
							"beta":  v1.InputParameter{Default: "b"},
						},
						Steps: []v1.Step{{Run: "echo test"}},
					},
				},
			},
			expected: []string{
				"   ordered-test -w alpha='a' -w beta='b' -w zebra='z' # Test ordering",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			table, err := DetailedTaskList(ctx, nil, nil, tc.workflow)

			require.NoError(t, err)
			assert.NotNil(t, table)

			assert.Equal(t, strings.Join(tc.expected, "\n"), table.String())
		})
	}
}

func TestDetailedTaskListWithAliases(t *testing.T) {
	t.Setenv("NO_COLOR", "true") // format matters more than ensuring colors are correct

	testCases := []struct {
		name      string
		workflow  v1.Workflow
		files     map[string][]byte
		expectErr string
		expected  []string
	}{
		{
			name: "alias path does not exist",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Description: "Main task",
						Steps:       []v1.Step{{Run: "echo main"}},
					},
				},
				Aliases: v1.AliasMap{
					"local": v1.Alias{
						Path: "other-tasks.yaml",
					},
				},
			},
			expectErr: "file does not exist",
		},
		{
			name: "non-path aliases",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"main": v1.Task{
						Description: "Main task",
						Steps:       []v1.Step{{Run: "echo main"}},
					},
				},
				Aliases: v1.AliasMap{
					"remote": v1.Alias{
						Type: "github",
					},
				},
			},
			expected: []string{
				"   main # Main task",
			},
		},
		{
			name: "multiple non-path aliases",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"local-task": v1.Task{
						Description: "Local task",
						Steps:       []v1.Step{{Run: "echo local"}},
					},
					"another": v1.Task{
						Description: "Another task",
						Steps:       []v1.Step{{Run: "echo another"}},
					},
				},
				Aliases: v1.AliasMap{
					"github-alias": v1.Alias{
						Type: "github",
					},
					"gitlab-alias": v1.Alias{
						Type: "gitlab",
					},
				},
			},
			expected: []string{
				"   another    # Another task",
				"   local-task # Local task  ",
			},
		},
		{
			name: "no aliases",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"standalone": v1.Task{
						Description: "Standalone task",
						Steps:       []v1.Step{{Run: "echo standalone"}},
					},
				},
			},
			expected: []string{
				"   standalone # Standalone task",
			},
		},
		{
			name: "empty workflow with aliases",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{},
				Aliases: v1.AliasMap{
					"remote": v1.Alias{
						Type: "github",
					},
				},
			},
			expected: []string{},
		},
		{
			name: "alias path exists with tasks",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"main": v1.Task{
						Description: "Main task",
						Steps:       []v1.Step{{Run: "echo main"}},
					},
				},
				Aliases: v1.AliasMap{
					"external": v1.Alias{
						Path: "external-tasks.yaml",
					},
				},
			},
			files: map[string][]byte{
				"external-tasks.yaml": []byte(`schema-version: v1
tasks:
  build:
    description: "Build the project"
    steps:
      - run: echo building
  test:
    description: "Test the project"
    steps:
      - run: echo testing
`),
			},
			expected: []string{
				"   main           # Main task        ",
				"   external:build # Build the project",
				"   external:test  # Test the project ",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			for name, content := range tc.files {
				err := afero.WriteFile(fs, name, content, 0o644)
				require.NoError(t, err)
			}

			svc, err := uses.NewFetcherService(uses.WithFS(fs))
			require.NoError(t, err)

			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			table, err := DetailedTaskList(ctx, svc, nil, tc.workflow)

			if tc.expectErr != "" {
				require.ErrorContains(t, err, tc.expectErr)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, table)

			assert.Equal(t, strings.Join(tc.expected, "\n"), table.String())
		})
	}
}

func TestRenderInputMap(t *testing.T) {
	testCases := []struct {
		name     string
		inputs   v1.InputMap
		expected string
	}{
		{
			name:     "empty input map",
			inputs:   v1.InputMap{},
			expected: "",
		},
		{
			name: "string default",
			inputs: v1.InputMap{
				"text": v1.InputParameter{Default: "hello world"},
			},
			expected: " -w text='hello world'",
		},
		{
			name: "boolean default",
			inputs: v1.InputMap{
				"enabled": v1.InputParameter{Default: true},
			},
			expected: " -w enabled='true'",
		},
		{
			name: "integer default",
			inputs: v1.InputMap{
				"count": v1.InputParameter{Default: 42},
			},
			expected: " -w count='42'",
		},
		{
			name: "default from env",
			inputs: v1.InputMap{
				"token": v1.InputParameter{
					Default:        "default-token",
					DefaultFromEnv: "API_TOKEN",
				},
			},
			expected: " -w token=\"${API_TOKEN:-default-token}\"",
		},
		{
			name: "required without default",
			inputs: v1.InputMap{
				"required": v1.InputParameter{
					Required: func() *bool { b := true; return &b }(),
				},
			},
			expected: " -w required=",
		},
		{
			name: "optional without default",
			inputs: v1.InputMap{
				"optional": v1.InputParameter{
					Required: func() *bool { b := false; return &b }(),
				},
			},
			expected: " -w optional=",
		},
		{
			name: "nil default treated as required",
			inputs: v1.InputMap{
				"nil-default": v1.InputParameter{Default: nil},
			},
			expected: " -w nil-default=",
		},
		{
			name: "zero and false values render",
			inputs: v1.InputMap{
				"zero":  v1.InputParameter{Default: 0},
				"false": v1.InputParameter{Default: false},
			},
			expected: " -w false='false' -w zero='0'",
		},
		{
			name: "env var without default",
			inputs: v1.InputMap{
				"env-only": v1.InputParameter{DefaultFromEnv: "MY_VAR"},
			},
			expected: " -w env-only=",
		},
		{
			name: "alphabetical ordering",
			inputs: v1.InputMap{
				"z-param": v1.InputParameter{Default: "z"},
				"a-param": v1.InputParameter{Default: "a"},
				"m-param": v1.InputParameter{Default: "m"},
			},
			expected: " -w a-param='a' -w m-param='m' -w z-param='z'",
		},
		{
			name: "special characters and whitespace",
			inputs: v1.InputMap{
				"quotes": v1.InputParameter{Default: "hello 'world' \"test\""},
				"empty":  v1.InputParameter{Default: ""},
				"spaces": v1.InputParameter{Default: "   "},
			},
			expected: " -w empty='' -w quotes='hello 'world' \"test\"' -w spaces='   '",
		},
		{
			name: "mixed input types",
			inputs: v1.InputMap{
				"default-val": v1.InputParameter{Default: "test"},
				"required-val": v1.InputParameter{
					Required: func() *bool { b := true; return &b }(),
				},
				"optional-val": v1.InputParameter{
					Required: func() *bool { b := false; return &b }(),
				},
				"env-val": v1.InputParameter{
					Default:        "fallback",
					DefaultFromEnv: "ENV_VAR",
				},
			},
			expected: " -w default-val='test' -w env-val=\"${ENV_VAR:-fallback}\" -w optional-val= -w required-val=",
		},
		{
			name: "comprehensive combinations",
			inputs: v1.InputMap{
				"z-string": v1.InputParameter{Default: "value"},
				"y-int":    v1.InputParameter{Default: 42},
				"x-bool":   v1.InputParameter{Default: false},
				"w-env": v1.InputParameter{
					Default:        "fallback",
					DefaultFromEnv: "TEST_VAR",
				},
				"v-required": v1.InputParameter{
					Required: func() *bool { b := true; return &b }(),
				},
				"u-optional": v1.InputParameter{
					Required: func() *bool { b := false; return &b }(),
				},
			},
			expected: " -w u-optional= -w v-required= -w w-env=\"${TEST_VAR:-fallback}\" -w x-bool='false' -w y-int='42' -w z-string='value'",
		},
		{
			name: "ordering consistency check",
			inputs: v1.InputMap{
				"third":  v1.InputParameter{Default: 3},
				"first":  v1.InputParameter{Default: 1},
				"second": v1.InputParameter{Default: 2},
			},
			expected: " -w first='1' -w second='2' -w third='3'",
		},
		{
			name: "numeric string sorting",
			inputs: v1.InputMap{
				"param-10": v1.InputParameter{Default: "ten"},
				"param-2":  v1.InputParameter{Default: "two"},
				"param-1":  v1.InputParameter{Default: "one"},
			},
			expected: " -w param-1='one' -w param-10='ten' -w param-2='two'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			renderInputMap(&buf, tc.inputs)
			assert.Equal(t, tc.expected, buf.String())
		})
	}
}
