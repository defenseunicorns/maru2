// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/log"
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
	ctx := context.Background()

	t.Run("basic workflow with tasks", func(t *testing.T) {
		wf := v1.Workflow{
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
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		// Render the table to check its content
		rendered := table.String()
		assert.Contains(t, rendered, "default")
		assert.Contains(t, rendered, "test")
		assert.Contains(t, rendered, "# Default task")
		assert.Contains(t, rendered, "# Test task")
	})

	t.Run("workflow with inputs", func(t *testing.T) {
		defaultVal := "default-value"
		required := true
		notRequired := false

		wf := v1.Workflow{
			Tasks: v1.TaskMap{
				"echo": v1.Task{
					Description: "Echo task with inputs",
					Inputs: v1.InputMap{
						"text": v1.InputParameter{
							Description: "Text to echo",
							Default:     &defaultVal,
						},
						"required-param": v1.InputParameter{
							Description: "Required parameter",
							Required:    &required,
						},
						"optional-param": v1.InputParameter{
							Description: "Optional parameter",
							Required:    &notRequired,
						},
					},
					Steps: []v1.Step{{Run: "echo ${{ input \"text\" }}"}},
				},
			},
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()
		assert.Contains(t, rendered, "echo")
		assert.Contains(t, rendered, "# Echo task with inputs")
		assert.Contains(t, rendered, "-w")
		assert.Contains(t, rendered, "text")
		assert.Contains(t, rendered, "required-param")
		// optional-param should not be rendered when required is false
		assert.NotContains(t, rendered, "optional-param")
	})

	t.Run("workflow with aliases and paths", func(t *testing.T) {
		wf := v1.Workflow{
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
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		// This should return an error because we can't actually fetch the alias
		table, err := DetailedTaskList(ctx, svc, origin, wf)

		// We expect an error when trying to resolve the relative path
		assert.Error(t, err)
		assert.Nil(t, table)
	})

	t.Run("empty workflow", func(t *testing.T) {
		wf := v1.Workflow{
			Tasks: v1.TaskMap{},
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()
		// Should be empty or minimal content
		assert.NotContains(t, rendered, "default")
	})

	t.Run("workflow with default from env", func(t *testing.T) {
		defaultVal := "fallback"

		wf := v1.Workflow{
			Tasks: v1.TaskMap{
				"env-task": v1.Task{
					Description: "Task with env default",
					Inputs: v1.InputMap{
						"value": v1.InputParameter{
							Description:    "Value from env",
							Default:        &defaultVal,
							DefaultFromEnv: "MY_ENV_VAR",
						},
					},
					Steps: []v1.Step{{Run: "echo ${{ input \"value\" }}"}},
				},
			},
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()
		assert.Contains(t, rendered, "env-task")
		assert.Contains(t, rendered, "-w")
		assert.Contains(t, rendered, "value")
		assert.Contains(t, rendered, "${MY_ENV_VAR:-fallback}")
	})
}

func TestRenderInputMap(t *testing.T) {
	testCases := []struct {
		name        string
		inputs      v1.InputMap
		expected    []string
		notExpected []string
	}{
		{
			name:        "empty input map",
			inputs:      v1.InputMap{},
			expected:    []string{},
			notExpected: []string{"-w"},
		},
		{
			name: "input with string default",
			inputs: v1.InputMap{
				"text": v1.InputParameter{
					Description: "Text parameter",
					Default:     "hello world",
				},
			},
			expected: []string{"-w", "text=", "'hello world'"},
		},
		{
			name: "input with boolean default",
			inputs: v1.InputMap{
				"enabled": v1.InputParameter{
					Description: "Boolean parameter",
					Default:     true,
				},
			},
			expected: []string{"-w", "enabled=", "'true'"},
		},
		{
			name: "input with integer default",
			inputs: v1.InputMap{
				"count": v1.InputParameter{
					Description: "Integer parameter",
					Default:     42,
				},
			},
			expected: []string{"-w", "count=", "'42'"},
		},
		{
			name: "input with default from env",
			inputs: v1.InputMap{
				"token": v1.InputParameter{
					Description:    "Token from environment",
					Default:        "default-token",
					DefaultFromEnv: "API_TOKEN",
				},
			},
			expected: []string{"-w", "token=", "${API_TOKEN:-default-token}"},
		},
		{
			name: "required input without default",
			inputs: v1.InputMap{
				"required": v1.InputParameter{
					Description: "Required parameter",
					Required:    func() *bool { b := true; return &b }(),
				},
			},
			expected: []string{"-w", "required=", "''"},
		},
		{
			name: "explicitly required input without default",
			inputs: v1.InputMap{
				"explicit": v1.InputParameter{
					Description: "Explicitly required parameter",
					Required:    func() *bool { b := true; return &b }(),
				},
			},
			expected: []string{"-w", "explicit=", "''"},
		},
		{
			name: "optional input without default",
			inputs: v1.InputMap{
				"optional": v1.InputParameter{
					Description: "Optional parameter",
					Required:    func() *bool { b := false; return &b }(),
				},
			},
			expected:    []string{},
			notExpected: []string{"optional"},
		},
		{
			name: "mixed inputs",
			inputs: v1.InputMap{
				"default-val": v1.InputParameter{
					Description: "Has default",
					Default:     "test",
				},
				"required-val": v1.InputParameter{
					Description: "Required",
					Required:    func() *bool { b := true; return &b }(),
				},
				"optional-val": v1.InputParameter{
					Description: "Optional",
					Required:    func() *bool { b := false; return &b }(),
				},
				"env-val": v1.InputParameter{
					Description:    "From env",
					Default:        "fallback",
					DefaultFromEnv: "ENV_VAR",
				},
			},
			expected: []string{
				"-w", "default-val=", "'test'",
				"-w", "required-val=", "''",
				"-w", "env-val=", "${ENV_VAR:-fallback}",
			},
			notExpected: []string{"optional-val"},
		},
		{
			name: "input with nil default (should be treated as required)",
			inputs: v1.InputMap{
				"nil-default": v1.InputParameter{
					Description: "Nil default parameter",
					Default:     nil,
				},
			},
			expected: []string{"-w", "nil-default=", "''"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf strings.Builder
			renderInputMap(&buf, tc.inputs)

			result := buf.String()

			for _, expected := range tc.expected {
				assert.Contains(t, result, expected, "Expected to find %q in result %q", expected, result)
			}

			for _, notExpected := range tc.notExpected {
				assert.NotContains(t, result, notExpected, "Did not expect to find %q in result %q", notExpected, result)
			}
		})
	}

	t.Run("multiple inputs rendered correctly", func(t *testing.T) {
		inputs := v1.InputMap{
			"z-param": v1.InputParameter{Default: "z"},
			"a-param": v1.InputParameter{Default: "a"},
			"m-param": v1.InputParameter{Default: "m"},
		}

		var buf strings.Builder
		renderInputMap(&buf, inputs)

		// Should contain all parameters (order may vary due to Go map iteration)
		result := buf.String()
		assert.Contains(t, result, "z-param='z'")
		assert.Contains(t, result, "a-param='a'")
		assert.Contains(t, result, "m-param='m'")
		// Should contain exactly 3 -w flags for the 3 parameters
		assert.Equal(t, 3, strings.Count(result, "-w"))
	})

	t.Run("special characters in default values", func(t *testing.T) {
		inputs := v1.InputMap{
			"special": v1.InputParameter{
				Description: "Special characters",
				Default:     "hello 'world' \"test\"",
			},
		}

		var buf strings.Builder
		renderInputMap(&buf, inputs)

		result := buf.String()
		assert.Contains(t, result, "special=")
		assert.Contains(t, result, "'hello 'world' \"test\"'")
	})

	t.Run("edge cases and nil values", func(t *testing.T) {
		testCases := []struct {
			name        string
			inputs      v1.InputMap
			expected    []string
			notExpected []string
		}{
			{
				name: "nil required field defaults to true",
				inputs: v1.InputMap{
					"implicit-required": v1.InputParameter{
						Description: "Implicitly required parameter",
						// Required is nil, should default to true behavior
					},
				},
				expected: []string{"-w", "implicit-required=", "''"},
			},
			{
				name: "default value of zero/false should still render",
				inputs: v1.InputMap{
					"zero-int": v1.InputParameter{
						Description: "Zero integer",
						Default:     0,
					},
					"false-bool": v1.InputParameter{
						Description: "False boolean",
						Default:     false,
					},
					"empty-string": v1.InputParameter{
						Description: "Empty string",
						Default:     "",
					},
				},
				expected: []string{
					"-w", "zero-int=", "'0'",
					"-w", "false-bool=", "'false'",
					"-w", "empty-string=", "''",
				},
			},
			{
				name: "default from env without default value",
				inputs: v1.InputMap{
					"env-only": v1.InputParameter{
						Description:    "Environment only",
						DefaultFromEnv: "MY_VAR",
						// No Default field set
					},
				},
				expected: []string{"-w", "env-only=", "''"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf strings.Builder
				renderInputMap(&buf, tc.inputs)

				result := buf.String()

				for _, expected := range tc.expected {
					assert.Contains(t, result, expected, "Expected to find %q in result %q", expected, result)
				}

				for _, notExpected := range tc.notExpected {
					assert.NotContains(t, result, notExpected, "Did not expect to find %q in result %q", notExpected, result)
				}
			})
		}
	})

	t.Run("stress test with many parameters", func(t *testing.T) {
		inputs := v1.InputMap{}

		// Add 50 parameters with various configurations
		for i := 0; i < 50; i++ {
			paramName := fmt.Sprintf("param-%d", i)
			param := v1.InputParameter{
				Description: fmt.Sprintf("Parameter %d", i),
			}

			switch i % 4 {
			case 0:
				param.Default = fmt.Sprintf("default-%d", i)
			case 1:
				required := true
				param.Required = &required
			case 2:
				required := false
				param.Required = &required
			case 3:
				param.Default = fmt.Sprintf("env-default-%d", i)
				param.DefaultFromEnv = fmt.Sprintf("ENV_%d", i)
			}

			inputs[paramName] = param
		}

		var buf strings.Builder
		renderInputMap(&buf, inputs)

		result := buf.String()

		// All parameters get a -w prefix, even optional ones that don't get fully rendered
		expectedCount := 50
		actualCount := strings.Count(result, "-w")
		assert.Equal(t, expectedCount, actualCount, "Expected %d -w flags but got %d", expectedCount, actualCount)

		// Should contain parameters with defaults or that are required
		assert.Contains(t, result, "param-0=") // Has default
		assert.Contains(t, result, "param-1=") // Required
		// param-2 is optional, so it has -w but no content after
		assert.Contains(t, result, "param-3=") // Has env default

		// Optional parameters should have -w but no parameter name= part
		// We can't easily test for this without complex parsing, so just ensure count is right
	})
}

func TestDetailedTaskListEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("workflow with complex task ordering", func(t *testing.T) {
		wf := v1.Workflow{
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
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()

		// Check that all tasks are present
		assert.Contains(t, rendered, "default")
		assert.Contains(t, rendered, "aaa-first")
		assert.Contains(t, rendered, "zzz-last")

		// Default should appear first due to OrderedTaskNames implementation
		defaultPos := strings.Index(rendered, "default")
		aaaPos := strings.Index(rendered, "aaa-first")
		zzzPos := strings.Index(rendered, "zzz-last")

		assert.True(t, defaultPos >= 0, "default task should be present")
		assert.True(t, aaaPos >= 0, "aaa-first task should be present")
		assert.True(t, zzzPos >= 0, "zzz-last task should be present")
		assert.True(t, defaultPos < aaaPos, "default should come before aaa-first")
	})

	t.Run("workflow with no description tasks", func(t *testing.T) {
		wf := v1.Workflow{
			Tasks: v1.TaskMap{
				"no-desc": v1.Task{
					Steps: []v1.Step{{Run: "echo no description"}},
					// Description is empty
				},
				"with-desc": v1.Task{
					Description: "Has description",
					Steps:       []v1.Step{{Run: "echo with description"}},
				},
			},
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()
		assert.Contains(t, rendered, "no-desc")
		assert.Contains(t, rendered, "with-desc")
		assert.Contains(t, rendered, "# Has description")
		// Task with no description should not have a comment
		assert.NotContains(t, rendered, "# no-desc")
	})

	t.Run("nil inputs handling", func(t *testing.T) {
		wf := v1.Workflow{
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
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()
		assert.Contains(t, rendered, "nil-inputs")
		assert.Contains(t, rendered, "empty-inputs")
		// Should not contain any -w flags since there are no inputs
		assert.NotContains(t, rendered, "-w")
	})

	t.Run("workflow with aliases without path resolution", func(t *testing.T) {
		wf := v1.Workflow{
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
		}

		svc, err := uses.NewFetcherService()
		require.NoError(t, err)
		origin, _ := url.Parse("file:///test/tasks.yaml")

		table, err := DetailedTaskList(ctx, svc, origin, wf)

		assert.NoError(t, err)
		assert.NotNil(t, table)

		rendered := table.String()
		assert.Contains(t, rendered, "main")
		assert.Contains(t, rendered, "# Main task")
		// Should only show the main task, not try to resolve remote alias
		assert.NotContains(t, rendered, "remote:")
	})
}
