// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/uses"
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

func TestTaskNamePattern(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"foo", true},
		{"foo-bar", true},
		{"foo_bar", true},
		{"foo-bar-1", true},
		{"foo_bar_1", true},
		{"foo1", true},
		{"foo-bar1", true},
		{"0", false},
		{"-foo", false},
		{"1foo", false},
		{"foo@bar", false},
		{"foo bar", false},
		{"_foo", true},
		{"a", true},
		{"foo-bar_baz", true},
		{"", false},
		{"foo--bar", true},
		{"foo__bar", true},
		{"foo-bar_", true},
		{"foo_bar-", true},
		{"foo--bar__baz", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok := TaskNamePattern.MatchString(tc.name)
			if ok != tc.expected {
				t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", tc.name, ok, tc.expected)
			}
		})
	}
}

func FuzzTaskNamePattern(f *testing.F) {
	// Add a variety of initial test cases, including both valid and invalid ones
	testCases := []string{
		"foo",
		"foo-bar",
		"foo_bar",
		"foo-bar-1",
		"foo_bar_1",
		"foo1",
		"foo-bar1",
		"0",             // invalid: single digit / starts with a digit
		"-foo",          // invalid: starts with a dash
		"1foo",          // invalid: starts with a digit
		"foo@bar",       // invalid: contains an illegal character
		"foo bar",       // invalid: contains a space
		"_foo",          // valid: starts with an underscore
		"a",             // valid: single character
		"foo-bar_baz",   // valid: combination of dash and underscore
		"",              // invalid: empty string
		"foo--bar",      // valid: double dash
		"foo__bar",      // valid: double underscore
		"foo-bar_",      // valid: ends with underscore
		"foo_bar-",      // valid: ends with dash
		"foo--bar__baz", // valid: multiple dashes and underscores
	}

	for _, s := range testCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		ok := TaskNamePattern.MatchString(s)
		// Ensure the match result aligns with the pattern's expected behavior
		if len(s) > 0 {
			startsWithValidChar := s[0] == '_' || (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z')
			containsOnlyValidChars := regexp.MustCompile("^[a-zA-Z0-9_-]*$").MatchString(s[1:])

			if startsWithValidChar && containsOnlyValidChars {
				if !ok {
					t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", s, ok, true)
				}
			} else {
				if ok {
					t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", s, ok, false)
				}
			}
		} else {
			if ok {
				t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", s, ok, false)
			}
		}
	})
}

func TestEnvVariablePattern(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"FOO", true},
		{"_FOO", true},
		{"FOO_BAR", true},
		{"FOO1", true},
		{"_FOO_BAR_1", true},
		{"foo_bar", true},
		{"1FOO", false},
		{"FOO-BAR", false},
		{"FOO@BAR", false},
		{"FOO BAR", false},
		{"FOO$BAR", false},
		{"", false},
		{"FOO__BAR", true},
		{"__FOO", true},
		{"FOO123BAR456", true},
		{"_123FOO", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok := EnvVariablePattern.MatchString(tc.name)
			if ok != tc.expected {
				t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", tc.name, ok, tc.expected)
			}
		})
	}
}

func FuzzEnvVariablePattern(f *testing.F) {
	// Add a variety of initial test cases, including both valid and invalid ones
	testCases := []string{
		"FOO",
		"_FOO",
		"FOO_BAR",
		"FOO1",
		"_FOO_BAR_1",
		"foo_bar",
		"1FOO",         // invalid: starts with a digit
		"FOO-BAR",      // invalid: contains a dash
		"FOO@BAR",      // invalid: contains an illegal character
		"FOO BAR",      // invalid: contains a space
		"FOO$BAR",      // invalid: contains a dollar sign
		"",             // invalid: empty string
		"FOO__BAR",     // valid: double underscore
		"__FOO",        // valid: starts with double underscore
		"FOO123BAR456", // valid: combination of letters and digits
		"_123FOO",      // valid: starts with underscore followed by digits
	}

	for _, s := range testCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		ok := EnvVariablePattern.MatchString(s)
		// Ensure the match result aligns with the pattern's expected behavior
		if len(s) > 0 {
			startsWithValidChar := (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z') || s[0] == '_'
			containsOnlyValidChars := regexp.MustCompile("^[a-zA-Z0-9_]*$").MatchString(s[1:])

			if startsWithValidChar && containsOnlyValidChars {
				if !ok {
					t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", s, ok, true)
				}
			} else {
				if ok {
					t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", s, ok, false)
				}
			}
		} else {
			if ok {
				t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", s, ok, false)
			}
		}
	})
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
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
					}},
				},
			},
		},
		{
			name: "no tasks",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks:  TaskMap{},
			},
			expectedError: "no tasks available",
		},
		{
			name: "invalid task name",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"2-echo": Task{Step{
						Run: "echo",
					}},
				},
			},
			expectedError: fmt.Sprintf("task name \"2-echo\" does not satisfy %q", TaskNamePattern.String()),
		},
		{
			name: "invalid step id",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
						ID:  "&1337",
					}},
				},
			},
			expectedError: fmt.Sprintf(".echo[0].id \"&1337\" does not satisfy %q", TaskNamePattern.String()),
		},
		{
			name: "duplicate step ids",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"echo": Task{
						Step{
							Run: "echo first",
							ID:  "same-id",
						},
						Step{
							Run: "echo second",
							ID:  "same-id",
						},
					},
				},
			},
			expectedError: ".echo[0] and .echo[1] have the same ID \"same-id\"",
		},
		{
			name: "both run and uses set",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Run:  "echo",
						Uses: "other-task",
					}},
				},
			},
			expectedError: ".task[0] has both run and uses fields set",
		},
		{
			name: "neither run nor uses set",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{}},
				},
			},
			expectedError: ".task[0] must have one of [run, uses] fields set",
		},
		{
			name: "uses with invalid URL",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Uses: ":\\invalid",
					}},
				},
			},
			expectedError: ".task[0].uses parse \":\\\\invalid\": missing protocol scheme",
		},
		{
			name: "uses with non-existent task",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Uses: "non-existent-task",
					}},
				},
			},
			expectedError: ".task[0].uses \"non-existent-task\" not found",
		},
		{
			name: "uses with invalid scheme",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Uses: "invalid://scheme",
					}},
				},
			},
			expectedError: fmt.Sprintf(".task[0].uses %q is not one of [%s]", "invalid", strings.Join(append(uses.SupportedSchemes(), "builtin"), ", ")),
		},
		{
			name: "uses with valid task reference",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task1": Task{Step{
						Run: "echo first",
					}},
					"task2": Task{Step{
						Uses: "task1",
					}},
				},
			},
		},
		{
			name: "uses with valid URL scheme",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Uses: "http://example.com/task",
					}},
				},
			},
		},
		{
			name: "valid workflow",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
					}},
				},
			},
		},
		{
			name: "input with valid regex validation",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs: InputMap{
					"name": InputParameter{
						Description: "Name with validation",
						Validate:    "^Hello",
					},
				},
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
					}},
				},
			},
		},
		{
			name: "input with invalid regex validation pattern",
			wf: Workflow{
				Inputs: InputMap{
					"name": InputParameter{
						Description: "Name with invalid validation",
						Validate:    "[", // Invalid regex
					},
				},
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
					}},
				},
			},
			expectedError: "error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "multiple inputs with valid and invalid regex validation",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
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
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
					}},
				},
			},
			expectedError: "error parsing regexp: unexpected ): `)`",
		},
		{
			name: "uses with valid task reference",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task1": Task{Step{
						Run: "echo first",
					}},
					"task2": Task{Step{
						Uses: "task1",
					}},
				},
			},
		},
		{
			name: "task with both run and uses",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Run:  "echo",
						Uses: "builtin:echo",
					}},
				},
			},
			expectedError: ".task[0] has both run and uses fields set",
		},
		{
			name: "task with neither run nor uses",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						// Missing both Run and Uses
					}},
				},
			},
			expectedError: ".task[0] must have one of [run, uses] fields set",
		},
		{
			name: "task with multiple validation errors",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"task": Task{
						Step{
							Run:  "echo",
							Uses: "builtin:echo",
						},
						Step{
							// Missing both Run and Uses
						},
					},
				},
			},
			expectedError: ".task[0] has both run and uses fields set",
		},
		{
			name: "invalid input schema validation",
			wf: Workflow{
				Inputs: InputMap{
					"input": InputParameter{
						Description: "Invalid input",
						Default:     make(chan int), // Invalid type for Default field
					},
				},
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
					}},
				},
			},
			expectedError: "json: unsupported type: chan int",
		},
		{
			name: "invalid task schema",
			wf: Workflow{
				Inputs: InputMap{},
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
						With: map[string]any{
							"invalid": make(chan int), // Invalid type for With field
						},
					}},
				},
			},
			expectedError: "json: unsupported type: chan int",
		},
		{
			name: "valid input schema",
			wf: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs: InputMap{
					"input": InputParameter{
						Description: "A test input",
						Default:     "default value",
					},
				},
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
					}},
				},
			},
		},
		{
			name: "step with absolute dir path",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{Step{
						Run: "echo",
						Dir: "/tmp",
					}},
				},
			},
			expectedError: ".task[0].dir \"/tmp\" must not be absolute",
		},
		{
			name: "step with invalid timeout",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{Step{
						Run:     "echo",
						Timeout: "5",
					}},
				},
			},
			expectedError: ".task[0].timeout \"5\" is not a valid time duration",
		},
		{
			name: "step with valid timeout and dir",
			wf: Workflow{
				Tasks: TaskMap{
					"task": Task{Step{
						Run:     "echo",
						Timeout: "5s",
						Dir:     "tmp",
					}},
				},
			},
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
schema-version: v0
tasks:
  echo:
    - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
					}},
				},
				Aliases: map[string]uses.Alias{},
			},
		},
		{
			name: "workflow with inputs",
			r: strings.NewReader(`
schema-version: v0
tasks:
  echo:
    - run: echo

inputs:
  name:
    description: "string"
    default: "default name"
`),
			expected: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs: InputMap{
					"name": InputParameter{
						Description: "string",
						Default:     "default name",
					},
				},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
					}},
				},
				Aliases: map[string]uses.Alias{},
			},
		},
		{
			name: "workflow with inputs and aliases",
			r: strings.NewReader(`
schema-version: v0
tasks:
  echo:
    - run: echo

inputs:
  name:
    description: "string"
    default: "default name"

aliases:
  gh:
    type: github
`),
			expected: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs: InputMap{
					"name": InputParameter{
						Description: "string",
						Default:     "default name",
					},
				},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
					}},
				},
				Aliases: map[string]uses.Alias{
					"gh": {
						Type: "github",
					},
				},
			},
		},
		{
			name: "workflow with extension keys",
			r: strings.NewReader(`
schema-version: v0
tasks:
  echo:
    - run: echo

x-metadata:
  description: "This is a test workflow"
`),
			expected: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
					}},
				},
				Aliases: map[string]uses.Alias{},
			},
		},
		{
			name: "invalid yaml",
			r:    strings.NewReader(`invalid: yaml::`),
			expected: Workflow{
				Inputs:  InputMap{},
				Tasks:   TaskMap{},
				Aliases: map[string]uses.Alias{},
			},
			expectedError: `[1:10] mapping value is not allowed in this context
>  1 | invalid: yaml::
                ^
`,
		},
		{
			name: "read error from reader",
			r:    badReadSeeker{failOnRead: true},
			expected: Workflow{
				Inputs:  InputMap{},
				Tasks:   TaskMap{},
				Aliases: map[string]uses.Alias{},
			},
			expectedError: "read failed",
		},
		{
			name: "seek error from reader",
			r:    badReadSeeker{failOnSeek: true},
			expected: Workflow{
				Inputs:  InputMap{},
				Tasks:   TaskMap{},
				Aliases: map[string]uses.Alias{},
			},
			expectedError: "seek failed",
		},
		{
			name: "error marshaling task",
			r: strings.NewReader(`
schema-version: v0
tasks:
  echo:
    - run: echo
      with:
      - invalid
`),
			expected: Workflow{
				Inputs:  InputMap{},
				Tasks:   TaskMap{},
				Aliases: map[string]uses.Alias{},
			},
			expectedError: `[7:7] sequence was used where mapping is expected
   4 |   echo:
   5 |     - run: echo
   6 |       with:
>  7 |       - invalid
             ^
`,
		},
		{
			name: "error marshaling input",
			r: strings.NewReader(`
schema-version: v0
tasks:
  echo:
    - run: echo

inputs:
  name:
    description: []
`),
			expected: Workflow{
				Inputs:  InputMap{},
				Tasks:   TaskMap{},
				Aliases: map[string]uses.Alias{},
			},
			expectedError: `[9:18] cannot unmarshal []interface {} into Go struct field Workflow.Inputs of type string
   7 | inputs:
   8 |   name:
>  9 |     description: []
                        ^
`,
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
schema-version: v0
tasks:
  echo:
    - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"echo": Task{Step{
						Run: "echo",
					}},
				},
				Aliases: map[string]uses.Alias{},
			},
			expectedReadErr:     "",
			expectedValidateErr: "",
		},
		{
			name: "read error",
			r:    strings.NewReader(`invalid: yaml::`),
			expected: Workflow{
				Inputs:  InputMap{},
				Tasks:   TaskMap{},
				Aliases: map[string]uses.Alias{},
			},
			expectedReadErr:     "[1:10] mapping value is not allowed in this context\n>  1 | invalid: yaml::\n                ^\n",
			expectedValidateErr: "",
		},
		{
			name: "validation error",
			r: strings.NewReader(`
schema-version: v0
tasks:
  2-echo:
    - run: echo
`),
			expected: Workflow{
				SchemaVersion: SchemaVersionV0,
				Inputs:        InputMap{},
				Tasks: TaskMap{
					"2-echo": Task{Step{
						Run: "echo",
					}},
				},
				Aliases: map[string]uses.Alias{},
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
