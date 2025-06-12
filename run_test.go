// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/uses"
)

func TestRunExtended(t *testing.T) {
	tests := []struct {
		name          string
		workflow      Workflow
		taskName      string
		with          With
		dry           bool
		expectedError string
		expectedOut   map[string]any
	}{
		{
			name: "simple task execution",
			workflow: Workflow{
				Tasks: TaskMap{
					"test": []Step{
						{
							Run: "echo hello",
						},
					},
				},
			},
			taskName:    "test",
			with:        With{},
			expectedOut: nil,
		},
		{
			name: "task with output",
			workflow: Workflow{
				Tasks: TaskMap{
					"test": []Step{
						{
							Run: "echo \"result=success\" >> $MARU2_OUTPUT",
							ID:  "step1",
						},
					},
				},
			},
			taskName:    "test",
			with:        With{},
			expectedOut: map[string]any{"result": "success"},
		},
		{
			name: "task not found",
			workflow: Workflow{
				Tasks: TaskMap{},
			},
			taskName:      "nonexistent",
			with:          With{},
			expectedError: "task \"nonexistent\" not found",
			expectedOut:   nil,
		},
		{
			name: "uses step",
			workflow: Workflow{
				Tasks: TaskMap{
					"test": []Step{
						{
							Uses: "builtin:echo",
							With: With{
								"text": "Hello, World!",
							},
							ID: "echo-step",
						},
					},
				},
			},
			taskName:    "test",
			with:        With{},
			expectedOut: map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "conditional step execution - success path",
			workflow: Workflow{
				Tasks: TaskMap{
					"test": []Step{
						{
							Run: "echo step1",
							ID:  "step1",
						},
						{
							Run: "echo step2",
							ID:  "step2",
							If:  "",
						},
						{
							Run: "echo failure step",
							ID:  "failure-step",
							If:  "failure",
						},
					},
				},
			},
			taskName:    "test",
			with:        With{},
			expectedOut: nil,
		},
		{
			name: "conditional step execution - failure path",
			workflow: Workflow{
				Tasks: TaskMap{
					"test": []Step{
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
							If:  "failure",
						},
					},
				},
			},
			taskName:      "test",
			with:          With{},
			expectedError: "exit status 1",
			expectedOut:   map[string]any{"result": "handled"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			svc, err := uses.NewFetcherService()
			require.NoError(t, err)

			result, err := Run(ctx, svc, tc.workflow, tc.taskName, tc.with, nil, tc.dry)

			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}

			assert.Equal(t, tc.expectedOut, result)
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
	tempDir := t.TempDir()
	outFilePath := filepath.Join(tempDir, "output.txt")

	with := With{}
	env := prepareEnvironment(with, outFilePath)

	outputEnv := "MARU2_OUTPUT=" + outFilePath
	assert.Contains(t, env, outputEnv, "MARU2_OUTPUT environment variable not set correctly")

	tests := []struct {
		name           string
		with           With
		expectedEnvVar string
		expectedValue  string
	}{
		{
			name: "string value",
			with: With{
				"test-input": "test-value",
			},
			expectedEnvVar: "INPUT_TEST_INPUT",
			expectedValue:  "test-value",
		},
		{
			name: "integer value",
			with: With{
				"number": 42,
			},
			expectedEnvVar: "INPUT_NUMBER",
			expectedValue:  "42",
		},
		{
			name: "boolean value",
			with: With{
				"flag": true,
			},
			expectedEnvVar: "INPUT_FLAG",
			expectedValue:  "true",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			outFilePath := filepath.Join(tempDir, "output.txt")

			env := prepareEnvironment(tc.with, outFilePath)

			expectedEnv := tc.expectedEnvVar + "=" + tc.expectedValue
			assert.Contains(t, env, expectedEnv, "Expected environment variable not found")
		})
	}
}

func TestHandleRunStep(t *testing.T) {
	tests := []struct {
		name          string
		step          Step
		withDefaults  With
		dry           bool
		expectedError string
		expectedOut   map[string]any
		expectedLog   string
	}{
		{
			name: "simple command",
			step: Step{
				Run: "echo hello",
			},
			withDefaults: With{},
			expectedLog:  "$ echo hello\n",
		},
		{
			name: "command with output",
			step: Step{
				Run: "echo \"result=success\" >> $MARU2_OUTPUT",
				ID:  "step1",
			},
			withDefaults: With{},
			expectedOut:  map[string]any{"result": "success"},
			expectedLog:  "$ echo \"result=success\" >> $MARU2_OUTPUT\n",
		},
		{
			name: "command with template",
			step: Step{
				Run: "echo ${{ input \"text\" }}",
			},
			withDefaults: With{"text": "hello world"},
			expectedLog:  "$ echo hello world\n",
		},
		{
			name: "dry run",
			step: Step{
				Run: "echo hello",
				ID:  "step1",
			},
			withDefaults: With{},
			dry:          true,
			expectedLog:  "$ echo hello\n",
		},
		{
			name: "command error",
			step: Step{
				Run: "exit 1",
			},
			withDefaults:  With{},
			expectedError: "exit status 1",
			expectedLog:   "$ exit 1\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", "true")

			var buf bytes.Buffer

			ctx := log.WithContext(t.Context(), log.NewWithOptions(&buf, log.Options{
				Level: log.InfoLevel,
			}))

			result, err := handleRunStep(ctx, tc.step, tc.withDefaults, nil, tc.dry)

			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}

			if tc.expectedOut != nil {
				assert.Equal(t, tc.expectedOut, result)
			}

			assert.Equal(t, tc.expectedLog, buf.String())
		})
	}
}

func TestHandleUsesStep(t *testing.T) {
	tests := []struct {
		name          string
		step          Step
		workflow      Workflow
		withDefaults  With
		origin        string
		dry           bool
		expectedError string
		expectedOut   map[string]any
	}{
		{
			name: "builtin echo",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": "Hello, World!",
				},
			},
			workflow:     Workflow{},
			withDefaults: With{},
			expectedOut:  map[string]any{"stdout": "Hello, World!"},
		},
		{
			name: "dry run builtin",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": "Hello, World!",
				},
			},
			workflow:     Workflow{},
			withDefaults: With{},
			dry:          true,
			expectedOut:  nil,
		},
		{
			name: "uses with template",
			step: Step{
				Uses: "builtin:echo",
				With: With{
					"text": "Hello from template",
				},
			},
			workflow:     Workflow{},
			withDefaults: With{},
			expectedOut:  map[string]any{"stdout": "Hello from template"},
		},
		{
			name: "nonexistent builtin",
			step: Step{
				Uses: "builtin:nonexistent",
			},
			workflow:      Workflow{},
			withDefaults:  With{},
			expectedError: "builtin:nonexistent not found",
			expectedOut:   nil,
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

			result, err := handleUsesStep(ctx, svc, tc.step, tc.workflow, tc.withDefaults, nil, origin, tc.dry)

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
