// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
)

type badReadSeeker struct {
	failOnRead bool
	failOnSeek bool
}

func (b badReadSeeker) Read(_ []byte) (n int, err error) {
	if b.failOnRead {
		return 0, fmt.Errorf("read failed")
	}
	return 0, nil
}

func (b badReadSeeker) Seek(_ int64, _ int) (int64, error) {
	if b.failOnSeek {
		return 0, fmt.Errorf("seek failed")
	}
	return 0, nil
}

func (badReadSeeker) Close() error {
	return nil
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name          string
		wf            Workflow
		expectedError string
	}{
		{
			name: "valid workflow",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
		},
		{
			name:          "no tasks",
			wf:            Workflow{},
			expectedError: "no tasks available",
		},
		{
			name: "invalid task name",
			wf: Workflow{
				Tasks: TaskMap{
					"2-echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedError: fmt.Sprintf("task name \"2-echo\" does not satisfy %q", TaskNamePattern.String()),
		},
		{
			name: "invalid step id",
			wf: Workflow{
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
							ID:  "&1337",
						}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.echo[0].id \"&1337\" does not satisfy %q", TaskNamePattern.String()),
		},
		{
			name: "duplicate step ids",
			wf: Workflow{
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{},
						Steps: []Step{
							{
								Run: "echo first",
								ID:  "same-id",
							},
							{
								Run: "echo second",
								ID:  "same-id",
							},
						},
					},
				},
			},
			expectedError: ".tasks.echo[0] and .tasks.echo[1] have the same ID \"same-id\"",
		},
		{
			name: "both run and uses set",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run:  "echo",
							Uses: "other-task",
						}},
					},
				},
			},
			expectedError: ".tasks.task[0] has both run and uses fields set",
		},
		{
			name: "neither run nor uses set",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps:  []Step{{}},
					},
				},
			},
			expectedError: ".tasks.task[0] must have one of [run, uses] fields set",
		},
		{
			name: "uses with invalid URL",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: ":\\invalid",
						}},
					},
				},
			},
			expectedError: ".tasks.task[0].uses parse \":\\\\invalid\": missing protocol scheme",
		},
		{
			name: "uses with non-existent task",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: "non-existent-task",
						}},
					},
				},
			},
			expectedError: ".tasks.task[0].uses \"non-existent-task\" not found",
		},
		{
			name: "uses cannot reference itself",
			wf: Workflow{
				Tasks: TaskMap{
					"self-task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: "self-task",
						}},
					},
				},
			},
			expectedError: ".tasks.self-task[0].uses cannot reference itself",
		},
		{
			name: "uses with invalid scheme",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: "invalid://scheme",
						}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.task[0].uses %q is not one of [%s]", "invalid", strings.Join(append(SupportedSchemes(), "builtin"), ", ")),
		},
		{
			name: "uses with valid task reference",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task1": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo first",
						}},
					},
					"task2": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: "task1",
						}},
					},
				},
			},
		},
		{
			name: "uses with valid URL scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: "http://example.com/task",
						}},
					},
				},
			},
		},
		{
			name: "task input with valid regex validation",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"name": InputParameter{
								Description: "Name with validation",
								Validate:    "^Hello",
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
		},
		{
			name: "task input with invalid regex validation pattern",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"name": InputParameter{
								Description: "Name with invalid validation",
								Validate:    "[", // Invalid regex
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedError: ".tasks.task.inputs.name: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "multiple task inputs with valid and invalid regex validation",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"name": InputParameter{
								Description: "Name with validation",
								Validate:    "^Hello",
							},
							"email": InputParameter{
								Description: "Email with invalid validation",
								Validate:    ")", // Invalid regex
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedError: ".tasks.task.inputs.email: error parsing regexp: unexpected ): `)`",
		},
		{
			name: "uses with valid task reference",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task1": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo first",
						}},
					},
					"task2": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Uses: "task1",
						}},
					},
				},
			},
		},
		{
			name: "task with both run and uses",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run:  "echo",
							Uses: "builtin:echo",
						}},
					},
				},
			},
			expectedError: ".tasks.task[0] has both run and uses fields set",
		},
		{
			name: "task with neither run nor uses",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps:  []Step{{
							// Missing both Run and Uses
						}},
					},
				},
			},
			expectedError: ".tasks.task[0] must have one of [run, uses] fields set",
		},
		{
			name: "task with multiple validation errors",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{
							{
								Run:  "echo",
								Uses: "builtin:echo",
							},
							{
								// Missing both Run and Uses
							},
						},
					},
				},
			},
			expectedError: ".tasks.task[0] has both run and uses fields set",
		},
		{
			name: "invalid task input schema validation",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"input": InputParameter{
								Description: "Invalid input",
								Default:     make(chan int), // Invalid type for Default field
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedError: "json: unsupported type: chan int",
		},
		{
			name: "invalid task schema",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
							With: map[string]any{
								"invalid": make(chan int), // Invalid type for With field
							},
						}},
					},
				},
			},
			expectedError: "json: unsupported type: chan int",
		},
		{
			name: "valid task input schema",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"input": InputParameter{
								Description: "A test input",
								Default:     "default value",
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
		},
		{
			name: "step with absolute dir path",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
							Dir: "/tmp",
						}},
					},
				},
			},
			expectedError: ".tasks.task[0].dir \"/tmp\" must not be absolute",
		},
		{
			name: "step with invalid timeout",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run:     "echo",
							Timeout: "5",
						}},
					},
				},
			},
			expectedError: ".tasks.task[0].timeout \"5\" is not a valid time duration",
		},
		{
			name: "step with valid timeout and dir",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run:     "echo",
							Timeout: "5s",
							Dir:     "tmp",
						}},
					},
				},
			},
		},
		{
			name: "valid env with string values",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo test",
							Env: schema.Env{
								"VAR1":  "value1",
								"VAR_2": "value2",
								"_VAR3": "value3",
							},
						}},
					},
				},
			},
		},
		{
			name: "valid env with different types",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo test",
							Env: schema.Env{
								"STRING_VAR": "hello",
								"INT_VAR":    42,
								"BOOL_VAR":   true,
							},
						}},
					},
				},
			},
		},
		{
			name: "valid env with underscore variations",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo test",
							Env: schema.Env{
								"_VAR":    "value1",
								"VAR_":    "value2",
								"VAR_1_2": "value3",
								"__VAR__": "value4",
							},
						}},
					},
				},
			},
		},
		{
			name: "empty env object should be valid",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo test",
							Env: schema.Env{},
						}},
					},
				},
			},
		},
		{
			name: "invalid env variable name violates schema",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo test",
							Env: schema.Env{
								"1INVALID": "value",
							},
						}},
					},
				},
			},
			expectedError: ".tasks.task[0].env \"1INVALID\" does not satisfy \"^[a-zA-Z_]+[a-zA-Z0-9_]*$\"",
		},
		{
			name: "invalid task input name",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{
						Inputs: InputMap{
							"2-invalid": InputParameter{
								Description: "Invalid input name",
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.task.inputs.2-invalid \"2-invalid\" does not satisfy %q", InputNamePattern.String()),
		},
		{
			name: "valid alias with path",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"local": {
						Path: "relative/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
		},
		{
			name: "invalid alias with absolute path",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"local": {
						Path: "/absolute/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: ".aliases.local cannot be an absolute path: /absolute/path.yaml",
		},
		{
			name: "valid alias with remote type",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"gh": {
						Type:         "github",
						BaseURL:      "https://api.github.com",
						TokenFromEnv: "GITHUB_TOKEN",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
		},
		{
			name: "valid input with regex validation",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"test": Task{
						Inputs: InputMap{
							"version": {
								Description: "Version number",
								Validate:    `^v\d+\.\d+\.\d+$`,
							},
						},
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
		},
		{
			name: "invalid input name pattern",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"test": Task{
						Inputs: InputMap{
							"2invalid": {
								Description: "Invalid input name",
							},
						},
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.test.inputs.2invalid \"2invalid\" does not satisfy %q", InputNamePattern.String()),
		},
		{
			name: "invalid regex pattern in input validation",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"test": Task{
						Inputs: InputMap{
							"version": {
								Description: "Version number",
								Validate:    "[invalid regex",
							},
						},
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: ".tasks.test.inputs.version: error parsing regexp: missing closing ]: `[invalid regex`",
		},
		{
			name: "valid step env variable names",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{
							Run: "echo test",
							Env: map[string]any{
								"VALID_ENV":   "value1",
								"ANOTHER_VAR": "value2",
								"VAR123":      "value3",
							},
						}},
					},
				},
			},
		},
		{
			name: "invalid step env variable name",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{
							Run: "echo test",
							Env: map[string]any{
								"2invalid": "value",
							},
						}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.test[0].env \"2invalid\" does not satisfy %q", EnvVariablePattern.String()),
		},
		{
			name: "valid uses with alias namespace",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"custom": {
						Path: "custom/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{
							Uses: "custom:task-name",
						}},
					},
				},
			},
		},
		{
			name: "invalid uses with unknown namespace",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{
							Uses: "unknown:task-name",
						}},
					},
				},
			},
			expectedError: ".tasks.test[0].uses \"unknown\" is not one of [file, http, https, pkg, oci, builtin]",
		},
		{
			name: "invalid uses with alias namespace and invalid task name",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"custom": {
						Path: "custom/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{
							Uses: "custom:2-invalid-task",
						}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.test[0].uses does not satisfy alias:task syntax: task \"2-invalid-task\" does not satisfy %q", TaskNamePattern.String()),
		},
		{
			name: "invalid uses with alias namespace and empty task name",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"custom": {
						Path: "custom/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{
							Uses: "custom:",
						}},
					},
				},
			},
			expectedError: fmt.Sprintf(".tasks.test[0].uses does not satisfy alias:task syntax: task \"\" does not satisfy %q", TaskNamePattern.String()),
		},
		{
			name: "invalid alias name using supported scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"file": {
						Path: "some/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: fmt.Sprintf(".aliases.file cannot be one of [%s]", strings.Join(SupportedSchemes(), ", ")),
		},
		{
			name: "invalid alias name using http scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"http": {
						Type: "github",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: fmt.Sprintf(".aliases.http cannot be one of [%s]", strings.Join(SupportedSchemes(), ", ")),
		},
		{
			name: "invalid alias name using pkg scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"pkg": {
						Path: "local/path.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: fmt.Sprintf(".aliases.pkg cannot be one of [%s]", strings.Join(SupportedSchemes(), ", ")),
		},
		{
			name: "invalid alias name using https scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"https": {
						Type: "gitlab",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: fmt.Sprintf(".aliases.https cannot be one of [%s]", strings.Join(SupportedSchemes(), ", ")),
		},
		{
			name: "invalid alias name using oci scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersion,
				Aliases: AliasMap{
					"oci": {
						Path: "workflows/common.yaml",
					},
				},
				Tasks: TaskMap{
					"test": Task{
						Steps: []Step{{Run: "echo test"}},
					},
				},
			},
			expectedError: fmt.Sprintf(".aliases.oci cannot be one of [%s]", strings.Join(SupportedSchemes(), ", ")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Validate(tc.wf)
			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestRead(t *testing.T) {
	testCases := []struct {
		name          string
		r             io.Reader
		expected      Workflow
		expectedError string
	}{
		{
			name: "simple workflow",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs: {}
    steps:
      - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
		},
		{
			name: "workflow with task inputs",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs:
      name:
        description: "string"
        default: "default name"
    steps:
      - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{
							"name": InputParameter{
								Description: "string",
								Default:     "default name",
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
		},
		{
			name: "workflow with task inputs and aliases",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs:
      name:
        description: "string"
        default: "default name"
    steps:
      - run: echo

aliases:
  gh:
    type: github
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{
							"name": InputParameter{
								Description: "string",
								Default:     "default name",
							},
						},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
				Aliases: AliasMap{
					"gh": {
						Type: "github",
					},
				},
			},
		},
		{
			name: "workflow with extension keys",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs: {}
    steps:
      - run: echo

x-metadata:
  description: "This is a test workflow"
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
		},
		{
			name: "v0 schema migration",
			r: strings.NewReader(`
schema-version: v0
inputs:
  name:
    description: "Name to echo"
    default: "world"
tasks:
  echo:
    - run: echo "Hello ${{ input \"name\" }}"
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{
							"name": InputParameter{
								Description: "Name to echo",
								Default:     "world",
							},
						},
						Steps: []Step{{
							Run: `echo "Hello ${{ input \"name\" }}"`,
						}},
					},
				},
			},
		},
		{
			name: "v0 schema migration with unmarshal error",
			r: strings.NewReader(`
schema-version: v0
inputs:
  name: invalid_structure
tasks:
  echo:
    - run: echo
`),
			expected: Workflow{},
			expectedError: `[4:9] string was used where mapping is expected
   2 | schema-version: v0
   3 | inputs:
>  4 |   name: invalid_structure
               ^
   5 | tasks:
   6 |   echo:
   7 |     - run: echo`,
		},
		{
			name: "v0 schema migration with Migrate error",
			r: strings.NewReader(`
schema-version: v0
tasks:
  echo:
    - run: echo
      with:
        - invalid: structure
`),
			expected: Workflow{},
			expectedError: `[7:9] sequence was used where mapping is expected
   4 |   echo:
   5 |     - run: echo
   6 |       with:
>  7 |         - invalid: structure
               ^
`,
		},
		{
			name:     "invalid yaml",
			r:        strings.NewReader(`invalid: yaml::`),
			expected: Workflow{},
			expectedError: `[1:10] mapping value is not allowed in this context
>  1 | invalid: yaml::
                ^
`,
		},
		{
			name: "missing schema version",
			r: strings.NewReader(`
tasks:
  echo:
    inputs: {}
    steps:
      - run: echo
`),
			expected:      Workflow{},
			expectedError: `unsupported schema version: expected oneof ["v1", "v0"], got ""`,
		},
		{
			name:          "read error from reader",
			r:             badReadSeeker{failOnRead: true},
			expected:      Workflow{},
			expectedError: "read failed",
		},
		{
			name:          "seek error from reader",
			r:             badReadSeeker{failOnSeek: true},
			expected:      Workflow{},
			expectedError: "seek failed",
		},
		{
			name: "error marshaling task",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs: {}
    steps:
      - run: echo
        with:
        - invalid
`),
			expected: Workflow{},
			expectedError: `[9:9] sequence was used where mapping is expected
   6 |     steps:
   7 |       - run: echo
   8 |         with:
>  9 |         - invalid
               ^
`,
		},
		{
			name: "error marshaling task input",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs:
      name:
        description: []
    steps:
      - run: echo
`),
			expected: Workflow{},
			expectedError: `[7:22] cannot unmarshal []interface {} into Go struct field Workflow.Tasks of type string
   4 |   echo:
   5 |     inputs:
   6 |       name:
>  7 |         description: []
                            ^
   8 |     steps:
   9 |       - run: echo`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wf, err := Read(tc.r)
			if tc.expectedError == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expected, wf)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestReadAndValidate(t *testing.T) {
	testCases := []struct {
		name                string
		r                   io.Reader
		expected            Workflow
		expectedReadErr     string
		expectedValidateErr string
	}{
		{
			name: "simple good read",
			r: strings.NewReader(`
schema-version: v1
tasks:
  echo:
    inputs: {}
    steps:
      - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedReadErr:     "",
			expectedValidateErr: "",
		},
		{
			name:                "read error",
			r:                   strings.NewReader(`invalid: yaml::`),
			expected:            Workflow{},
			expectedReadErr:     "[1:10] mapping value is not allowed in this context\n>  1 | invalid: yaml::\n                ^\n",
			expectedValidateErr: "",
		},
		{
			name: "validation error",
			r: strings.NewReader(`
schema-version: v1
tasks:
  2-echo:
    inputs: {}
    steps:
      - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks: TaskMap{
					"2-echo": Task{
						Inputs: InputMap{},
						Steps: []Step{{
							Run: "echo",
						}},
					},
				},
			},
			expectedReadErr:     "",
			expectedValidateErr: fmt.Sprintf("task name \"2-echo\" does not satisfy %q", TaskNamePattern.String()),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wf, err := ReadAndValidate(tc.r)
			if tc.expectedReadErr != "" {
				require.EqualError(t, err, tc.expectedReadErr)
			} else if tc.expectedValidateErr != "" {
				require.EqualError(t, err, tc.expectedValidateErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, wf)
			}
		})
	}
}

func TestValidateSchemaOnce(t *testing.T) {
	tests := []struct {
		name           string
		setupSchema    func() (string, error)
		expectedErrMsg string
	}{
		{
			name: "schema generation error",
			setupSchema: func() (string, error) {
				return "", assert.AnError
			},
			expectedErrMsg: assert.AnError.Error(),
		},
		{
			name: "invalid schema loader",
			setupSchema: func() (string, error) {
				return `{"type": "invalid-json-schema", "properties": {`, nil
			},
			expectedErrMsg: "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSchemaOnce := schemaOnce
			t.Cleanup(func() {
				schemaOnce = originalSchemaOnce
			})

			schemaOnce = sync.OnceValues(tt.setupSchema)

			err := Validate(Workflow{
				Tasks: TaskMap{"default": Task{}},
			})
			require.ErrorContains(t, err, tt.expectedErrMsg)
		})
	}
}
