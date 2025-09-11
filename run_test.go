// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name          string
		workflow      v1.Workflow
		taskName      string
		with          schema.With
		dry           bool
		expectedError string
		expectedOut   map[string]any
	}{
		{
			name: "simple task execution",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo hello >/dev/null",
							},
						},
					},
				},
			},
			taskName:    "test",
			with:        schema.With{},
			expectedOut: nil,
		},
		{
			name: "task with output",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo \"result=success\" >> $MARU2_OUTPUT",
								ID:  "step1",
							},
						},
					},
				},
			},
			taskName:    "test",
			with:        schema.With{},
			expectedOut: map[string]any{"result": "success"},
		},
		{
			name: "task not found",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{},
			},
			taskName:      "nonexistent",
			with:          schema.With{},
			expectedError: "task \"nonexistent\" not found",
			expectedOut:   nil,
		},
		{
			name: "uses step",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Uses: "builtin:echo",
								With: schema.With{
									"text": "Hello, World!",
								},
								ID: "echo-step",
							},
						},
					},
				},
			},
			taskName:    "test",
			with:        schema.With{},
			expectedOut: map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "conditional step execution - success path",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo step1 >/dev/null",
								ID:  "step1",
							},
							{
								Run: "echo step2 >/dev/null",
								ID:  "step2",
								If:  "",
							},
							{
								Run: "echo failure step",
								ID:  "failure-step",
								If:  "failure()",
							},
						},
					},
				},
			},
			taskName:    "test",
			with:        schema.With{},
			expectedOut: nil,
		},
		{
			name: "conditional step execution - failure path",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "exit 1",
								ID:  "step1",
							},
							{
								Run: "echo normal step",
								ID:  "normal-step",
								If:  "",
							},
							{
								Run: "echo \"result=handled\" >> $MARU2_OUTPUT",
								ID:  "failure-step",
								If:  "failure()",
							},
						},
					},
				},
			},
			taskName:      "test",
			with:          schema.With{},
			expectedError: "exit status 1",
			expectedOut:   map[string]any{"result": "handled"},
		},
		{
			name: "failed to parse duration",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"sleep": v1.Task{
						Steps: []v1.Step{
							{
								Run:     "sleep 3",
								Timeout: "1",
							},
						},
					},
				},
			},
			taskName:      "sleep",
			expectedError: "time: missing unit in duration \"1\"",
		},
		{
			name: "step with timeout",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					schema.DefaultTaskName: {
						Inputs: v1.InputMap{},
						Steps: []v1.Step{
							{
								Run:     "sleep 0.1",
								Timeout: "50ms",
							},
						},
					},
				},
			},
			taskName:      schema.DefaultTaskName,
			with:          schema.With{},
			expectedError: "signal: killed",
		},
		{
			name: "ShouldRun with missing input returns false",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo hello",
								If:  "input(\"nonexistent\")",
							},
						},
					},
				},
			},
			taskName: "test",
			with:     schema.With{},
		},
		{
			name: "ShouldRun error with prior error (logs but continues)",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "exit 1",
								ID:  "failing-step",
							},
							{
								Run: "echo \"result=handled\" >> $MARU2_OUTPUT",
								If:  "input(\"nonexistent\")",
								ID:  "error-step",
							},
							{
								Run: "echo \"final=done\" >> $MARU2_OUTPUT",
								If:  "failure()",
								ID:  "cleanup-step",
							},
						},
					},
				},
			},
			taskName:      "test",
			with:          schema.With{},
			expectedError: "exit status 1",
			expectedOut:   map[string]any{"final": "done"},
		},
		{
			name: "ShouldRun syntax error in if expression",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo hello",
								If:  "invalid syntax (",
							},
						},
					},
				},
			},
			taskName:      "test",
			with:          schema.With{},
			expectedError: "unexpected token Identifier(\"syntax\") (1:9)\n | invalid syntax (\n | ........^",
		},
		{
			name: "ShouldRun runtime error with prior error (logs but continues)",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "exit 1",
								ID:  "failing-step",
							},
							{
								Run: "echo \"result=handled\" >> $MARU2_OUTPUT",
								If:  "from(\"nonexistent\", \"key\")",
								ID:  "error-step",
							},
							{
								Run: "echo \"final=done\" >> $MARU2_OUTPUT",
								If:  "failure()",
								ID:  "cleanup-step",
							},
						},
					},
				},
			},
			taskName:      "test",
			with:          schema.With{},
			expectedError: "exit status 1",
			expectedOut:   map[string]any{"final": "done"},
		},
		{
			name: "ShouldRun error during context cancellation path",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "exit 1",
								ID:  "failing-step",
							},
							{
								Run: "echo skipped",
								If:  "from(\"badstep\", \"missing\")",
								ID:  "error-in-cancelled-context",
							},
							{
								Run: "echo \"result=cleanup\" >> $MARU2_OUTPUT",
								If:  "always()",
								ID:  "cleanup-step",
							},
						},
					},
				},
			},
			taskName:      "test",
			with:          schema.With{},
			expectedError: "exit status 1",
			expectedOut:   map[string]any{"result": "cleanup"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			svc, err := uses.NewFetcherService()
			require.NoError(t, err)

			result, err := Run(ctx, svc, tc.workflow, tc.taskName, tc.with, nil, "", nil, tc.dry)

			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}

			assert.Equal(t, tc.expectedOut, result)
		})
	}
}

func TestRunContext(t *testing.T) {
	discardLogCtx := log.WithContext(context.Background(), log.New(io.Discard))

	tests := []struct {
		name                 string
		workflow             v1.Workflow
		taskName             string
		setupContext         func() (context.Context, context.CancelFunc)
		cancelAfter          time.Duration
		expectedError        string
		expectedOutput       map[string]any
		expectedContextError error
	}{
		{
			name: "context timeout cancellation",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"sleep": v1.Task{
						Steps: []v1.Step{
							{
								Run: "sleep 5",
								ID:  "sleep-step",
							},
							{
								Run: "echo \"result=timeout-handled\" >> $MARU2_OUTPUT",
								ID:  "timeout-step",
								If:  "always()",
							},
						},
					},
				},
			},
			taskName: "sleep",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(discardLogCtx, 100*time.Millisecond)
			},
			expectedError: "signal: killed",
			expectedOutput: map[string]any{
				"result": "timeout-handled",
			},
			expectedContextError: context.DeadlineExceeded,
		},
		{
			name: "manual cancellation (simulating SIGINT)",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"sleep": v1.Task{
						Steps: []v1.Step{
							{
								Run: "sleep 5",
								ID:  "sleep-step",
							},
							{
								Run: "echo \"result=cancelled\" >> $MARU2_OUTPUT",
								ID:  "cancel-step",
								If:  "cancelled()",
							},
						},
					},
				},
			},
			taskName: "sleep",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(discardLogCtx)
			},
			cancelAfter:          100 * time.Millisecond,
			expectedError:        "signal: killed",
			expectedContextError: context.Canceled,
			expectedOutput:       nil,
		},
		{
			name: "context with cause cancellation",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"sleep": v1.Task{
						Steps: []v1.Step{
							{
								Run: "sleep 5",
								ID:  "sleep-step",
							},
							{
								Run: "echo \"result=caused\" >> $MARU2_OUTPUT",
								ID:  "cause-step",
								If:  "always()",
							},
						},
					},
				},
			},
			taskName: "sleep",
			setupContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(discardLogCtx)
				return ctx, func() {
					cancel(errors.New("custom cancellation cause"))
				}
			},
			cancelAfter:          100 * time.Millisecond,
			expectedError:        "signal: killed",
			expectedContextError: context.Canceled,
			expectedOutput:       nil,
		},
		{
			name: "successful completion without cancellation",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"quick": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo \"result=success\" >> $MARU2_OUTPUT",
								ID:  "quick-step",
							},
						},
					},
				},
			},
			taskName: "quick",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(discardLogCtx, 5*time.Second)
			},
			expectedOutput: map[string]any{
				"result": "success",
			},
			expectedContextError: nil,
		},
		{
			name: "step timeout with context still valid",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"timeout-step": v1.Task{
						Steps: []v1.Step{
							{
								Run:     "sleep 5",
								Timeout: "50ms",
								ID:      "timeout-step",
							},
							{
								Run: "echo \"result=timeout-recovered\" >> $MARU2_OUTPUT",
								ID:  "recovery-step",
								If:  "always()",
							},
						},
					},
				},
			},
			taskName: "timeout-step",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(discardLogCtx, 5*time.Second)
			},
			expectedError: "signal: killed",
			expectedOutput: map[string]any{
				"result": "timeout-recovered",
			},
			expectedContextError: nil,
		},
		{
			name: "timeout should NOT trigger cancelled()",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"timeout-test": v1.Task{
						Steps: []v1.Step{
							{
								Run: "sleep 5",
								ID:  "sleep-step",
							},
							{
								Run: "echo \"result=cancelled-step\" >> $MARU2_OUTPUT",
								ID:  "cancelled-step",
								If:  "cancelled()",
							},
							{
								Run: "echo \"result=always-step\" >> $MARU2_OUTPUT",
								ID:  "timeout-handled-step",
								If:  "always()",
							},
						},
					},
				},
			},
			taskName: "timeout-test",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(discardLogCtx, 100*time.Millisecond)
			},
			expectedError: "signal: killed",
			expectedOutput: map[string]any{
				"result": "always-step", // Only always() should run, not cancelled()
			},
			expectedContextError: context.DeadlineExceeded,
		},
		{
			name: "step timeout should NOT trigger cancelled() on parent context",
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"step-timeout-test": v1.Task{
						Steps: []v1.Step{
							{
								Run:     "sleep 5",
								Timeout: "50ms",
								ID:      "timeout-step",
							},
							{
								Run: "echo \"result=cancelled-step\" >> $MARU2_OUTPUT",
								ID:  "cancelled-step",
								If:  "cancelled()",
							},
							{
								Run: "echo \"result=always-step\" >> $MARU2_OUTPUT",
								ID:  "timeout-handled-step",
								If:  "always()",
							},
						},
					},
				},
			},
			taskName: "step-timeout-test",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(discardLogCtx, 5*time.Second)
			},
			expectedError: "signal: killed",
			expectedOutput: map[string]any{
				"result": "always-step", // Only always() should run, not cancelled()
			},
			expectedContextError: nil, // Parent context should still be valid
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := uses.NewFetcherService()
			require.NoError(t, err)

			testCtx, cancel := tc.setupContext()
			defer cancel()

			// If we need to cancel after a delay, do it in a goroutine
			if tc.cancelAfter > 0 {
				go func() {
					time.Sleep(tc.cancelAfter)
					cancel()
				}()
			}

			out, err := Run(testCtx, svc, tc.workflow, tc.taskName, schema.With{}, nil, "", nil, false)

			if tc.expectedError != "" {
				require.ErrorContains(t, err, tc.expectedError)

				require.ErrorIs(t, testCtx.Err(), tc.expectedContextError)

				// Special handling for context with cause cancellation
				if tc.name == "context with cause cancellation" {
					assert.Contains(t, context.Cause(testCtx).Error(), "custom cancellation cause")
				}
			} else {
				require.NoError(t, err)
				require.NoError(t, testCtx.Err())
			}

			assert.Equal(t, tc.expectedOutput, out)
		})
	}
}

func TestToEnvVar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"input", "INPUT"},
		{"input-name", "INPUT_NAME"},
		{"input_name", "INPUT_NAME"},
		{"inputName", "INPUTNAME"},
		{"input-name-with-dashes", "INPUT_NAME_WITH_DASHES"},
		{"", ""},
		{"-", "_"},
		{"--", "__"},
		{"_", "_"},
		{"__", "__"},
		{"-_", "__"},
		{"_-", "__"},
		{"Input-Name", "INPUT_NAME"},
		{"INPUT-NAME", "INPUT_NAME"},
		{"mixed_Case-Name", "MIXED_CASE_NAME"},
		{"CamelCase-kebab_snake", "CAMELCASE_KEBAB_SNAKE"},
		{"input1", "INPUT1"},
		{"input-1", "INPUT_1"},
		{"input-name-2", "INPUT_NAME_2"},
		{"v1-beta", "V1_BETA"},
		{"api-v2-endpoint", "API_V2_ENDPOINT"},
		{"input--name", "INPUT__NAME"},
		{"input---name", "INPUT___NAME"},
		{"input-name--with-multiple", "INPUT_NAME__WITH_MULTIPLE"},
		{"-input", "_INPUT"},
		{"input-", "INPUT_"},
		{"-input-", "_INPUT_"},
		{"--input--", "__INPUT__"},
		{"a", "A"},
		{"z", "Z"},
		{"1", "1"},
		{"very-long-input-name-with-many-dashes", "VERY_LONG_INPUT_NAME_WITH_MANY_DASHES"},
		{"very_long_input_name_with_many_underscores", "VERY_LONG_INPUT_NAME_WITH_MANY_UNDERSCORES"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			result := toEnvVar(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPrepareEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		startingEnv     []string
		withDefaults    schema.With
		stepEnv         schema.Env
		expectedEnvVars []string
		expectedError   string
	}{
		{
			name:            "empty inputs and step env",
			withDefaults:    schema.With{},
			stepEnv:         nil,
			expectedEnvVars: []string{},
		},
		{
			name: "string input value",
			withDefaults: schema.With{
				"test-input": "test-value",
			},
			stepEnv: nil,
			expectedEnvVars: []string{
				"INPUT_TEST_INPUT=test-value",
			},
		},
		{
			name: "integer input value",
			withDefaults: schema.With{
				"number": 42,
			},
			stepEnv: nil,
			expectedEnvVars: []string{
				"INPUT_NUMBER=42",
			},
		},
		{
			name: "boolean input value",
			withDefaults: schema.With{
				"flag": true,
			},
			stepEnv: nil,
			expectedEnvVars: []string{
				"INPUT_FLAG=true",
			},
		},
		{
			name: "no step env",
			withDefaults: schema.With{
				"test-input": "test-value",
			},
			stepEnv: nil,
			expectedEnvVars: []string{
				"INPUT_TEST_INPUT=test-value",
			},
		},
		{
			name:         "step env with string",
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"CUSTOM_VAR": "custom-value",
			},
			expectedEnvVars: []string{
				"CUSTOM_VAR=custom-value",
			},
		},
		{
			name:         "step env with different types",
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"STRING_VAR": "hello",
				"INT_VAR":    42,
				"BOOL_VAR":   true,
			},
			expectedEnvVars: []string{
				"STRING_VAR=hello",
				"INT_VAR=42",
				"BOOL_VAR=true",
			},
		},
		{
			name: "both input and step env",
			withDefaults: schema.With{
				"input-var": "input-value",
			},
			stepEnv: schema.Env{
				"CUSTOM_VAR": "custom-value",
			},
			expectedEnvVars: []string{
				"INPUT_INPUT_VAR=input-value",
				"CUSTOM_VAR=custom-value",
			},
		},
		{
			name:            "empty step env map",
			withDefaults:    schema.With{},
			stepEnv:         schema.Env{},
			expectedEnvVars: []string{},
		},
		{
			name:         "step env overrides existing env",
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"PATH": "/custom/path",
			},
			expectedEnvVars: []string{
				"PATH=/custom/path",
			},
		},
		{
			name:         "complex values in step env",
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"JSON_VAR":   `{"key": "value", "number": 42}`,
				"SPACES_VAR": "value with spaces",
				"EMPTY_VAR":  "",
			},
			expectedEnvVars: []string{
				`JSON_VAR="{\"key\": \"value\", \"number\": 42}"`,
				`SPACES_VAR="value with spaces"`,
				"EMPTY_VAR=",
			},
		},
		{
			name:         "PWD variable should be rejected",
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"PWD": "/some/path",
			},
			expectedError: "setting PWD environment variable is not allowed",
		},
		{
			name: "invalid input type conversion",
			withDefaults: schema.With{
				"bad-input": make(chan int), // channels can't be converted to string
			},
			stepEnv:       schema.Env{},
			expectedError: "failed to convert input \"bad-input\" to string",
		},
		{
			name:         "invalid env var type conversion",
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"BAD_VAR": make(chan int), // channels can't be converted to string
			},
			expectedError: "failed to convert env var \"BAD_VAR\" to string",
		},
		{
			name: "starting env with basic variables",
			startingEnv: []string{
				"PATH=/usr/bin:/bin",
				"HOME=/home/user",
				"USER=testuser",
			},
			withDefaults: schema.With{},
			stepEnv:      schema.Env{},
			expectedEnvVars: []string{
				"PATH=/usr/bin:/bin",
				"HOME=/home/user",
				"USER=testuser",
			},
		},
		{
			name: "starting env with inputs added",
			startingEnv: []string{
				"PATH=/usr/bin:/bin",
				"HOME=/home/user",
			},
			withDefaults: schema.With{
				"test-input": "test-value",
			},
			stepEnv: schema.Env{},
			expectedEnvVars: []string{
				"PATH=/usr/bin:/bin",
				"HOME=/home/user",
				"INPUT_TEST_INPUT=test-value",
			},
		},
		{
			name: "starting env with step env override",
			startingEnv: []string{
				"PATH=/usr/bin:/bin",
				"HOME=/home/user",
				"EXISTING_VAR=original",
			},
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"EXISTING_VAR": "overridden",
				"NEW_VAR":      "new-value",
			},
			expectedEnvVars: []string{
				"PATH=/usr/bin:/bin",
				"HOME=/home/user",
				"EXISTING_VAR=original",   // starting env is preserved as-is
				"EXISTING_VAR=overridden", // step env appends new variables
				"NEW_VAR=new-value",
			},
		},
		{
			name: "starting env with inputs and step env",
			startingEnv: []string{
				"PATH=/usr/bin:/bin",
				"SHELL=/bin/bash",
			},
			withDefaults: schema.With{
				"name":    "test",
				"version": 123,
			},
			stepEnv: schema.Env{
				"CUSTOM_VAR": "custom",
				"DEBUG":      true,
			},
			expectedEnvVars: []string{
				"PATH=/usr/bin:/bin",
				"SHELL=/bin/bash",
				"INPUT_NAME=test",
				"INPUT_VERSION=123",
				"CUSTOM_VAR=custom",
				"DEBUG=true",
			},
		},
		{
			name:        "empty starting env vs nil starting env",
			startingEnv: []string{},
			withDefaults: schema.With{
				"test": "value",
			},
			stepEnv: schema.Env{
				"STEP_VAR": "step-value",
			},
			expectedEnvVars: []string{
				"INPUT_TEST=value",
				"STEP_VAR=step-value",
			},
		},
		{
			name:         "nil withDefaults with empty outFileName (uses.go pattern)",
			startingEnv:  []string{"PATH=/usr/bin"},
			withDefaults: nil,
			stepEnv: schema.Env{
				"CUSTOM_VAR": "value",
			},
			expectedEnvVars: []string{
				"PATH=/usr/bin",
				"CUSTOM_VAR=value",
			},
		},
		{
			name:         "nil withDefaults and nil stepEnv with empty outFileName",
			startingEnv:  []string{"HOME=/home/user"},
			withDefaults: nil,
			stepEnv:      nil,
			expectedEnvVars: []string{
				"HOME=/home/user",
			},
		},
		{
			name:         "empty withDefaults with empty outFileName",
			startingEnv:  []string{"USER=testuser"},
			withDefaults: schema.With{},
			stepEnv: schema.Env{
				"STEP_VAR": "step-value",
			},
			expectedEnvVars: []string{
				"USER=testuser",
				"STEP_VAR=step-value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			outFilePath := filepath.Join(tempDir, "output.txt")

			// Use empty outFileName for specific test cases that match uses.go usage pattern
			actualOutFileName := outFilePath
			if strings.Contains(tc.name, "empty outFileName") {
				actualOutFileName = ""
			}

			env, err := prepareEnvironment(tc.startingEnv, tc.withDefaults, actualOutFileName, tc.stepEnv)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}

			require.NoError(t, err)

			if actualOutFileName != "" {
				outputEnv := "MARU2_OUTPUT=" + actualOutFileName
				assert.Contains(t, env, outputEnv, "MARU2_OUTPUT environment variable not set correctly")
			} else {
				for _, envVar := range env {
					assert.NotContains(t, envVar, "MARU2_OUTPUT=", "MARU2_OUTPUT should not be set when outFileName is empty")
				}
			}

			for _, expectedEnv := range tc.expectedEnvVars {
				assert.Contains(t, env, expectedEnv, "Expected environment variable not found: %s", expectedEnv)
			}
		})
	}
}

func TestHandleRunStep(t *testing.T) {
	tests := []struct {
		name          string
		step          v1.Step
		withDefaults  schema.With
		dry           bool
		expectedError string
		expectedOut   map[string]any
		expectedLog   string
	}{
		{
			name: "simple command",
			step: v1.Step{
				Run: "echo hello",
			},
			withDefaults: schema.With{},
			expectedLog:  "echo hello\n",
		},
		{
			name: "command with output",
			step: v1.Step{
				Run: "echo \"result=success\" >> $MARU2_OUTPUT",
				ID:  "step1",
			},
			withDefaults: schema.With{},
			expectedOut:  map[string]any{"result": "success"},
			expectedLog:  "echo \"result=success\" >> $MARU2_OUTPUT\n",
		},
		{
			name: "command with template",
			step: v1.Step{
				Run: "echo ${{ input \"text\" }}",
			},
			withDefaults: schema.With{"text": "hello world"},
			expectedLog:  "echo hello world\n",
		},
		{
			name: "bash array works",
			step: v1.Step{
				Run:   `arr=(a b c); echo "${arr[1]}"`,
				Shell: "bash",
			},
			withDefaults: schema.With{},
			expectedLog:  "arr=(a b c); echo \"${arr[1]}\"\n",
		},
		{
			name: "[[ ... ]] works in bash",
			step: v1.Step{
				Run:   `if [[ "foo" == "foo" ]]; then echo "match"; fi`,
				Shell: "bash",
			},
			withDefaults: schema.With{},
			expectedLog:  "if [[ \"foo\" == \"foo\" ]]; then echo \"match\"; fi\n",
		},
		{
			name: "unsupported shell",
			step: v1.Step{
				Run:   "echo foo",
				Shell: "fish",
			},
			withDefaults:  schema.With{},
			expectedLog:   "echo foo\n",
			expectedError: "unsupported shell: fish",
		},
		{
			name: "dry run",
			step: v1.Step{
				Run: "echo hello",
				ID:  "step1",
			},
			withDefaults: schema.With{},
			dry:          true,
			expectedLog:  "echo hello\n",
		},
		{
			name: "command error",
			step: v1.Step{
				Run: "exit 1",
			},
			withDefaults:  schema.With{},
			expectedError: "exit status 1",
			expectedLog:   "exit 1\n",
		},
		{
			name: "muted command",
			step: v1.Step{
				Run:  "echo 'This should not appear in output'",
				Mute: true,
			},
			withDefaults: schema.With{},
			expectedLog:  "echo 'This should not appear in output'\n",
		},
		{
			name: "muted command still can send outputs",
			step: v1.Step{
				Run:  "echo 'foo=bar' >> $MARU2_OUTPUT",
				Mute: true,
			},
			withDefaults: schema.With{},
			expectedLog:  "echo 'foo=bar' >> $MARU2_OUTPUT\n",
			expectedOut:  map[string]any{"foo": "bar"},
		},
		{
			name: "step with environment variables",
			step: v1.Step{
				Run: "echo \"MY_VAR=$MY_VAR\" && echo \"TEMPLATED_VAR=$TEMPLATED_VAR\"",
				Env: schema.Env{
					"MY_VAR":        "static-value",
					"TEMPLATED_VAR": "${{ input \"name\" }}",
				},
			},
			withDefaults: schema.With{"name": "world"},
			expectedLog:  "echo \"MY_VAR=$MY_VAR\" && echo \"TEMPLATED_VAR=$TEMPLATED_VAR\"\n",
		},
		{
			name: "dry run with environment variables",
			step: v1.Step{
				Run: "echo \"TEST_VAR=$TEST_VAR\"",
				Env: schema.Env{
					"TEST_VAR": "${{ input \"value\" }}",
				},
			},
			withDefaults: schema.With{"value": "dry-run-test"},
			dry:          true,
			expectedLog:  "echo \"TEST_VAR=$TEST_VAR\"\n",
		},
		{
			name: "step with env templating error",
			step: v1.Step{
				Run: "echo test",
				Env: schema.Env{
					"BAD_VAR": "${{ input \"nonexistent\" }}",
				},
			},
			withDefaults:  schema.With{},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <input "nonexistent">: error calling input: input "nonexistent" does not exist in []`,
			expectedLog:   "echo test\n",
		},
		{
			name: "step with empty env map",
			step: v1.Step{
				Run: "echo \"Empty env map test completed\"",
				Env: schema.Env{},
			},
			withDefaults: schema.With{},
			expectedLog:  "echo \"Empty env map test completed\"\n",
		},
		{
			name: "step with run templating error in dry mode",
			step: v1.Step{
				Run: "echo ${{ invalid syntax }}",
			},
			withDefaults:  schema.With{},
			dry:           true,
			expectedError: `template: dry-run expression evaluator:1: function "invalid" not defined`,
			expectedLog:   "\n",
		},
		{
			name: "step with PWD in env should fail",
			step: v1.Step{
				Run: "echo test",
				Env: schema.Env{
					"PWD": "/some/path",
				},
			},
			withDefaults:  schema.With{},
			expectedError: "setting PWD environment variable is not allowed",
			expectedLog:   "echo test\n",
		},
		{
			name: "step with invalid input type should fail",
			step: v1.Step{
				Run: "echo test",
			},
			withDefaults: schema.With{
				"bad-input": complex(1, 2), // complex numbers can't be converted to string
			},
			expectedError: "failed to convert input \"bad-input\" to string: unable to cast (1+2i) of type complex128 to string",
			expectedLog:   "echo test\n",
		},
		{
			name: "step with invalid env var type should fail",
			step: v1.Step{
				Run: "echo test",
				Env: schema.Env{
					"BAD_VAR": complex(1, 2), // complex numbers can't be converted to string
				},
			},
			withDefaults:  schema.With{},
			expectedError: "failed to convert env var \"BAD_VAR\" to string: unable to cast (1+2i) of type complex128 to string",
			expectedLog:   "echo test\n",
		},
		{
			name: "unset environment variable in sh shell",
			step: v1.Step{
				Run:   "echo \"UNSET_VAR value: '$UNSET_VAR'\" >/dev/null",
				Shell: "sh",
			},
			withDefaults: schema.With{},
			expectedLog:  "echo \"UNSET_VAR value: '$UNSET_VAR'\" >/dev/null\n",
		},
		{
			name: "unset environment variable in bash shell",
			step: v1.Step{
				Run:   "echo \"UNSET_VAR value: '$UNSET_VAR'\" >/dev/null",
				Shell: "bash",
			},
			withDefaults: schema.With{},
			expectedLog:  "echo \"UNSET_VAR value: '$UNSET_VAR'\" >/dev/null\n",
		},
		{
			name: "unset environment variable with default shell",
			step: v1.Step{
				Run: "echo \"UNSET_VAR value: '$UNSET_VAR'\" >/dev/null",
			},
			withDefaults: schema.With{},
			expectedLog:  "echo \"UNSET_VAR value: '$UNSET_VAR'\" >/dev/null\n",
		},
		{
			name: "mixed set and unset environment variables",
			step: v1.Step{
				Run: "echo \"SET_VAR: '$SET_VAR', UNSET_VAR: '$UNSET_VAR'\" >/dev/null",
				Env: schema.Env{
					"SET_VAR": "defined_value",
				},
			},
			withDefaults: schema.With{},
			expectedLog:  "echo \"SET_VAR: '$SET_VAR', UNSET_VAR: '$UNSET_VAR'\" >/dev/null\n",
		},
	}

	t.Setenv("NO_COLOR", "true")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			ctx := log.WithContext(t.Context(), log.NewWithOptions(&buf, log.Options{
				Level: log.InfoLevel,
			}))

			result, err := handleRunStep(ctx, tc.step, tc.withDefaults, nil, "", nil, tc.dry)

			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}

			assert.Equal(t, tc.expectedOut, result)

			assert.Equal(t, tc.expectedLog, buf.String())
		})
	}
}

func TestHandleUsesStep(t *testing.T) {
	tests := []struct {
		name          string
		step          v1.Step
		workflow      v1.Workflow
		withDefaults  schema.With
		origin        string
		dry           bool
		expectedError string
		expectedOut   map[string]any
	}{
		{
			name: "builtin echo",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Hello, World!",
				},
			},
			workflow:     v1.Workflow{},
			withDefaults: schema.With{},
			expectedOut:  map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "dry run builtin",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Hello, World!",
				},
			},
			workflow:     v1.Workflow{},
			withDefaults: schema.With{},
			dry:          true,
			expectedOut:  nil,
		},
		{
			name: "uses with template",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "Hello from template",
				},
			},
			workflow:     v1.Workflow{},
			withDefaults: schema.With{},
			expectedOut:  map[string]any{"stdout": "Hello from template"},
		},
		{
			name: "nonexistent builtin",
			step: v1.Step{
				Uses: "builtin:nonexistent",
			},
			workflow:      v1.Workflow{},
			withDefaults:  schema.With{},
			expectedError: "builtin:nonexistent not found",
			expectedOut:   nil,
		},
		{
			name: "template error in step.With",
			step: v1.Step{
				Uses: "builtin:echo",
				With: schema.With{
					"text": "${{ input \"nonexistent\" }}",
				},
			},
			workflow:      v1.Workflow{},
			withDefaults:  schema.With{},
			expectedError: `builtin:echo: template: expression evaluator:1:4: executing "expression evaluator" at <input "nonexistent">: error calling input: input "nonexistent" does not exist in []`,
			expectedOut:   nil,
		},
		{
			name: "template error in local task step.With",
			step: v1.Step{
				Uses: "test-task",
				With: schema.With{
					"input": "${{ input \"nonexistent\" }}",
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo ${{ input \"input\" }}",
							},
						},
					},
				},
			},
			withDefaults:  schema.With{},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <input "nonexistent">: error calling input: input "nonexistent" does not exist in []`,
			expectedOut:   nil,
		},
		{
			name: "template error in local task step.Env",
			step: v1.Step{
				Uses: "test-task",
				With: schema.With{
					"input": "hello",
				},
				Env: schema.Env{
					"TEST_VAR": "${{ input \"nonexistent\" }}",
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo ${{ input \"input\" }}",
							},
						},
					},
				},
			},
			withDefaults:  schema.With{},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <input "nonexistent">: error calling input: input "nonexistent" does not exist in []`,
			expectedOut:   nil,
		},
		{
			name: "PWD in local task step.Env should fail",
			step: v1.Step{
				Uses: "test-task",
				With: schema.With{
					"input": "hello",
				},
				Env: schema.Env{
					"PWD": "/some/path",
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo ${{ input \"input\" }}",
							},
						},
					},
				},
			},
			withDefaults:  schema.With{},
			expectedError: "setting PWD environment variable is not allowed",
			expectedOut:   nil,
		},
		{
			name: "invalid type in local task step.Env should fail",
			step: v1.Step{
				Uses: "test-task",
				With: schema.With{
					"input": "hello",
				},
				Env: schema.Env{
					"BAD_VAR": complex(1, 2),
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo ${{ input \"input\" }}",
							},
						},
					},
				},
			},
			withDefaults:  schema.With{},
			expectedError: "failed to convert env var \"BAD_VAR\" to string: unable to cast (1+2i) of type complex128 to string",
			expectedOut:   nil,
		},
		{
			name: "template error in local task step.Env",
			step: v1.Step{
				Uses: "test-task",
				With: schema.With{
					"input": "valid-input",
				},
				Env: schema.Env{
					"TEST_VAR": "${{ input \"nonexistent_env_var\" }}",
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo ${{ input \"input\" }}",
							},
						},
					},
				},
			},
			withDefaults: schema.With{
				"input": "provided-input",
			},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <input "nonexistent_env_var">: error calling input: input "nonexistent_env_var" does not exist in [input]`,
			expectedOut:   nil,
		},
		{
			name: "successful local task execution",
			step: v1.Step{
				Uses: "test-task",
				With: schema.With{
					"input": "hello world",
				},
				Env: schema.Env{
					"TEST_VAR": "test-value",
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"test-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo ${{ input \"input\" }}",
							},
						},
					},
				},
			},
			withDefaults: schema.With{},
			dry:          true, // Use dry run to avoid actual command execution
			expectedOut:  nil,  // Dry run returns nil
		},
		{
			name: "successful local task execution with output",
			step: v1.Step{
				Uses: "output-task",
				With: schema.With{
					"message": "test output",
				},
			},
			workflow: v1.Workflow{
				Tasks: v1.TaskMap{
					"output-task": v1.Task{
						Steps: []v1.Step{
							{
								Run: "echo \"result=${{ input \"message\" }}\" >> $MARU2_OUTPUT",
							},
						},
					},
				},
			},
			withDefaults: schema.With{},
			expectedOut:  map[string]any{"result": "test output"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			svc, err := uses.NewFetcherService()
			require.NoError(t, err)

			origin, err := url.Parse(tc.origin)
			require.NoError(t, err)

			result, err := handleUsesStep(ctx, svc, tc.step, tc.workflow, tc.withDefaults, nil, origin, "", nil, tc.dry)

			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}

			assert.Equal(t, tc.expectedOut, result)
		})
	}
}

func TestTraceError(t *testing.T) {
	t.Run("TraceError methods", func(t *testing.T) {
		tests := []struct {
			name           string
			err            error
			expectedMsg    string
			expectedUnwrap error
		}{
			{
				name:           "simple error",
				err:            errors.New("test error"),
				expectedMsg:    "test error",
				expectedUnwrap: errors.New("test error"),
			},
			{
				name:           "wrapped error",
				err:            fmt.Errorf("wrapped: %w", errors.New("inner error")),
				expectedMsg:    "wrapped: inner error",
				expectedUnwrap: fmt.Errorf("wrapped: %w", errors.New("inner error")),
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				traceErr := &TraceError{
					err:   tc.err,
					Trace: []string{"frame1", "frame2"},
				}

				assert.Equal(t, tc.expectedMsg, traceErr.Error())
				require.EqualError(t, traceErr.Unwrap(), tc.expectedUnwrap.Error())
				assert.Len(t, traceErr.Trace, 2)
				assert.Equal(t, "frame1", traceErr.Trace[0])
				assert.Equal(t, "frame2", traceErr.Trace[1])
			})
		}
	})

	t.Run("addTrace function", func(t *testing.T) {
		tests := []struct {
			name          string
			err           error
			frames        []string
			expectedTrace []string
		}{
			{
				name:          "new trace error",
				err:           errors.New("base error"),
				frames:        []string{"frame1"},
				expectedTrace: []string{"frame1"},
			},
			{
				name:          "append to existing trace",
				err:           &TraceError{err: errors.New("base error"), Trace: []string{"existing"}},
				frames:        []string{"frame1"},
				expectedTrace: []string{"frame1", "existing"},
			},
			{
				name:          "multiple frames",
				err:           errors.New("base error"),
				frames:        []string{"frame3", "frame2", "frame1"},
				expectedTrace: []string{"frame3", "frame2", "frame1"},
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				resultErr := tc.err
				// Apply frames in reverse to simulate the call stack
				for i := len(tc.frames) - 1; i >= 0; i-- {
					resultErr = addTrace(resultErr, tc.frames[i])
				}

				var traceErr *TraceError
				require.ErrorAs(t, resultErr, &traceErr)
				assert.Equal(t, tc.expectedTrace, traceErr.Trace)
				assert.Equal(t, tc.err.Error(), traceErr.Error())
			})
		}
	})
}
