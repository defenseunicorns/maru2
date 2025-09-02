// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

func TestMigrate(t *testing.T) {
	t.Parallel()

	boolPtr := func(b bool) *bool {
		return &b
	}

	tests := []struct {
		name        string
		input       any
		expected    Workflow
		expectedErr string
	}{
		{
			name:  "empty workflow",
			input: v0.Workflow{},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks:         TaskMap{},
			},
		},
		{
			name: "valid v0 workflow with inputs and tasks",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Inputs: v0.InputMap{
					"text": v0.InputParameter{
						Description: "Text to echo",
						Default:     "Hello, world!",
						Required:    boolPtr(true),
					},
					"debug": v0.InputParameter{
						Description: "Enable debug mode",
						Default:     false,
					},
				},
				Tasks: v0.TaskMap{
					"echo": v0.Task{
						{
							Run: "echo \"${{ input \"text\" }}\"",
						},
					},
					"complex": v0.Task{
						{
							Uses: "echo",
							With: schema.With{
								"text": "Hello from complex task",
							},
							ID: "step1",
						},
						{
							Run: "echo \"Debug: ${{ input \"debug\" }}\"",
							Env: schema.Env{
								"DEBUG": "${{ input \"debug\" }}",
							},
							If:      "input(\"debug\")",
							Dir:     "subdir",
							Shell:   "bash",
							Timeout: "30s",
							Mute:    true,
						},
					},
				},
				Aliases: v0.AliasMap{
					"gh": v0.Alias{
						Type:         "github",
						Base:         "https://api.github.com",
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{
							"text": InputParameter{
								Description: "Text to echo",
								Default:     "Hello, world!",
								Required:    boolPtr(true),
							},
							"debug": InputParameter{
								Description: "Enable debug mode",
								Default:     false,
							},
						},
						Steps: []Step{
							{
								Run: "echo \"${{ input \"text\" }}\"",
							},
						},
					},
					"complex": Task{
						Inputs: InputMap{
							"text": InputParameter{
								Description: "Text to echo",
								Default:     "Hello, world!",
								Required:    boolPtr(true),
							},
							"debug": InputParameter{
								Description: "Enable debug mode",
								Default:     false,
							},
						},
						Steps: []Step{
							{
								Uses: "echo",
								With: schema.With{
									"text": "Hello from complex task",
								},
								ID: "step1",
							},
							{
								Run: "echo \"Debug: ${{ input \"debug\" }}\"",
								Env: schema.Env{
									"DEBUG": "${{ input \"debug\" }}",
								},
								If:      "input(\"debug\")",
								Dir:     "subdir",
								Shell:   "bash",
								Timeout: "30s",
								Mute:    true,
							},
						},
					},
				},
				Aliases: AliasMap{
					"gh": Alias{
						Type:         "github",
						BaseURL:      "https://api.github.com",
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
			},
		},
		{
			name: "v0 workflow with no inputs",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Tasks: v0.TaskMap{
					"simple": v0.Task{
						{
							Run: "echo hello",
						},
					},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"simple": Task{
						Steps: []Step{
							{
								Run: "echo hello",
							},
						},
					},
				},
			},
		},
		{
			name: "v0 workflow with no tasks",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Inputs: v0.InputMap{
					"name": v0.InputParameter{
						Description: "Name parameter",
						Default:     "test",
					},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks:         TaskMap{},
			},
		},
		{
			name: "v0 workflow with empty tasks",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Tasks: v0.TaskMap{
					"empty": v0.Task{},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"empty": Task{
						Steps: []Step{},
					},
				},
			},
		},
		{
			name: "v0 workflow with complex step properties",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Tasks: v0.TaskMap{
					"complex-step": v0.Task{
						{
							Run:  "echo test",
							Name: "Test step",
							Env: schema.Env{
								"VAR1": "value1",
								"VAR2": 42,
								"VAR3": true,
							},
							With: schema.With{
								"param1": "value1",
								"param2": 123,
								"param3": false,
							},
							ID:      "test-id",
							If:      "always()",
							Dir:     "workdir",
							Shell:   "bash",
							Timeout: "5m",
							Mute:    true,
						},
					},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"complex-step": Task{
						Steps: []Step{
							{
								Run:  "echo test",
								Name: "Test step",
								Env: schema.Env{
									"VAR1": "value1",
									"VAR2": 42,
									"VAR3": true,
								},
								With: schema.With{
									"param1": "value1",
									"param2": 123,
									"param3": false,
								},
								ID:      "test-id",
								If:      "always()",
								Dir:     "workdir",
								Shell:   "bash",
								Timeout: "5m",
								Mute:    true,
							},
						},
					},
				},
			},
		},
		{
			name: "input parameter type migration",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Inputs: v0.InputMap{
					"test": v0.InputParameter{
						Description:       "Test description",
						DeprecatedMessage: "Test deprecation",
						Required:          boolPtr(false),
						Default:           "test default",
						DefaultFromEnv:    "TEST_ENV",
						Validate:          "^test.*",
					},
				},
				Tasks: v0.TaskMap{
					"task": v0.Task{
						{Run: "echo test"},
					},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"test": InputParameter{
								Description:       "Test description",
								DeprecatedMessage: "Test deprecation",
								Required:          boolPtr(false),
								Default:           "test default",
								DefaultFromEnv:    "TEST_ENV",
								Validate:          "^test.*",
							},
						},
						Steps: []Step{
							{Run: "echo test"},
						},
					},
				},
			},
		},
		{
			name: "alias type migration",
			input: v0.Workflow{
				SchemaVersion: "v0",
				Aliases: v0.AliasMap{
					"gh": v0.Alias{
						Type:         "github",
						Base:         "https://github.com",
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
				Tasks: v0.TaskMap{
					"task": v0.Task{
						{Run: "echo test"},
					},
				},
			},
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Steps: []Step{
							{Run: "echo test"},
						},
					},
				},
				Aliases: AliasMap{
					"gh": Alias{
						Type:         "github",
						BaseURL:      "https://github.com",
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
			},
		},
		{
			name:        "invalid input type string",
			input:       "not a workflow",
			expectedErr: "unsupported type: string",
		},
		{
			name:        "invalid input type nil",
			input:       nil,
			expectedErr: "unsupported type: <nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := Migrate(tt.input)

			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
