// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/uses"
)

func TestRunExtendedWindows(t *testing.T) {
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
							Run:   "echo hello",
							Shell: "powershell",
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
							Run:   "echo \"result=success\" | Out-File -Append -Encoding ascii $env:MARU2_OUTPUT",
							ID:    "step1",
							Shell: "powershell",
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
							Run:   "echo step1",
							ID:    "step1",
							Shell: "powershell",
						},
						{
							Run:   "echo step2",
							ID:    "step2",
							If:    "",
							Shell: "powershell",
						},
						{
							Run:   "echo failure step",
							ID:    "failure-step",
							If:    "failure()",
							Shell: "powershell",
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
							Run:   "exit 1",
							ID:    "step1",
							Shell: "powershell",
						},
						{
							Run:   "echo normal step",
							ID:    "normal-step",
							If:    "",
							Shell: "powershell",
						},
						{
							Run:   "echo \"result=handled\" | Out-File -Append -Encoding ascii $env:MARU2_OUTPUT",
							ID:    "failure-step",
							If:    "failure()",
							Shell: "powershell",
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
