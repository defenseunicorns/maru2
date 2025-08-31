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

	testCases := []struct {
		name     string
		input    any
		expected Workflow
		wantErr  bool
	}{
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
						Base:         "https://api.github.com",
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
			},
			wantErr: false,
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
						Inputs: InputMap{},
						Steps: []Step{
							{
								Run: "echo hello",
							},
						},
					},
				},
			},
			wantErr: false,
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
			wantErr: false,
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
						Inputs: InputMap{},
						Steps:  []Step{},
					},
				},
			},
			wantErr: false,
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
						Inputs: InputMap{},
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
			wantErr: false,
		},
		{
			name:    "invalid input type",
			input:   "not a workflow",
			wantErr: true,
		},
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := Migrate(tc.input)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMigrateInputParameterTypes(t *testing.T) {
	t.Parallel()

	v0Input := v0.InputParameter{
		Description:       "Test description",
		DeprecatedMessage: "Test deprecation",
		Required:          boolPtr(false),
		Default:           "test default",
		DefaultFromEnv:    "TEST_ENV",
		Validate:          "^test.*",
	}

	v0Workflow := v0.Workflow{
		SchemaVersion: "v0",
		Inputs: v0.InputMap{
			"test": v0Input,
		},
		Tasks: v0.TaskMap{
			"task": v0.Task{
				{Run: "echo test"},
			},
		},
	}

	result, err := Migrate(v0Workflow)
	require.NoError(t, err)

	expectedInput := InputParameter{
		Description:       "Test description",
		DeprecatedMessage: "Test deprecation",
		Required:          boolPtr(false),
		Default:           "test default",
		DefaultFromEnv:    "TEST_ENV",
		Validate:          "^test.*",
	}

	assert.Equal(t, expectedInput, result.Tasks["task"].Inputs["test"])
}

func TestMigrateAliasTypes(t *testing.T) {
	t.Parallel()

	v0Alias := v0.Alias{
		Type:         "github",
		Base:         "https://github.com",
		TokenFromEnv: "GITHUB_TOKEN",
	}

	v0Workflow := v0.Workflow{
		SchemaVersion: "v0",
		Aliases: v0.AliasMap{
			"gh": v0Alias,
		},
		Tasks: v0.TaskMap{
			"task": v0.Task{
				{Run: "echo test"},
			},
		},
	}

	result, err := Migrate(v0Workflow)
	require.NoError(t, err)

	expectedAlias := Alias{
		Type:         "github",
		Base:         "https://github.com",
		TokenFromEnv: "GITHUB_TOKEN",
	}

	assert.Equal(t, expectedAlias, result.Aliases["gh"])
}

// boolPtr returns a pointer to a boolean value
func boolPtr(b bool) *bool {
	return &b
}
