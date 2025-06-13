// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"runtime"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateString(t *testing.T) {
	tests := []struct {
		name           string
		input          With
		previousOutput CommandOutputs
		str            string
		expected       string
		expectedError  string
		dryRun         bool
	}{
		{
			name:     "no template",
			str:      "hello world",
			expected: "hello world",
		},
		{
			name:     "with input",
			input:    With{"name": "test"},
			str:      "hello ${{ input \"name\" }}",
			expected: "hello test",
		},
		{
			name:          "with missing input",
			input:         With{},
			str:           "hello ${{ input \"name\" }}",
			expectedError: "\"name\" does not exist in []",
		},
		{
			name: "with previous output",
			previousOutput: CommandOutputs{
				"step1": map[string]any{
					"result": "success",
				},
			},
			str:      "status: ${{ from \"step1\" \"result\" }}",
			expected: "status: success",
		},
		{
			name:           "with missing previous output",
			previousOutput: CommandOutputs{},
			str:            "status: ${{ from \"step1\" \"result\" }}",
			expectedError:  "no outputs from step \"step1\"",
		},
		{
			name:     "with OS variable",
			str:      "OS: ${{ .OS }}",
			expected: "OS: " + runtime.GOOS,
		},
		{
			name:     "with ARCH variable",
			str:      "ARCH: ${{ .ARCH }}",
			expected: "ARCH: " + runtime.GOARCH,
		},
		{
			name:     "with PLATFORM variable",
			str:      "PLATFORM: ${{ .PLATFORM }}",
			expected: "PLATFORM: " + runtime.GOOS + "/" + runtime.GOARCH,
		},
		{
			name:  "with multiple variables",
			input: With{"name": "test"},
			previousOutput: CommandOutputs{
				"step1": map[string]any{
					"result": "success",
				},
			},
			str:      "Hello ${{ input \"name\" }}, status: ${{ from \"step1\" \"result\" }}, OS: ${{ .OS }}",
			expected: "Hello test, status: success, OS: " + runtime.GOOS,
		},
		{
			name:          "invalid template syntax",
			str:           "Hello ${{ input",
			expectedError: "unclosed action",
		},
		// Dry run tests
		{
			name:     "dry run - no template",
			str:      "hello world",
			expected: "hello world",
			dryRun:   true,
		},
		{
			name:     "dry run - with input",
			input:    With{"name": "test"},
			str:      `hello ${{ input "name" }}`,
			expected: "hello test",
			dryRun:   true,
		},
		{
			name:     "dry run - with missing input",
			input:    With{},
			str:      `hello ${{ input "name" }}`,
			expected: "hello ❯ input name ❮",
			dryRun:   true,
		},
		{
			name: "dry run - with previous output",
			previousOutput: CommandOutputs{
				"step1": map[string]any{
					"result": "success",
				},
			},
			str:      `status: ${{ from "step1" "result" }}`,
			expected: "status: success",
			dryRun:   true,
		},
		{
			name:     "dry run - with missing previous output",
			str:      `status: ${{ from "step1" "result" }}`,
			expected: "status: ❯ from step1 result ❮",
			dryRun:   true,
		},
		{
			name: "dry run - with missing previous output arg",
			str:  `status: ${{ from "step1" "result" }}`,
			previousOutput: CommandOutputs{
				"step1": map[string]any{
					"foo": "bar",
				},
			},
			expected: "status: ❯ from step1 result ❮",
			dryRun:   true,
		},
		{
			name:     "dry run - with OS variable",
			str:      `OS: ${{ .OS }}`,
			expected: "OS: " + runtime.GOOS,
			dryRun:   true,
		},
		{
			name:  "dry run - with multiple variables",
			input: With{"name": "test"},
			previousOutput: CommandOutputs{
				"step1": map[string]any{
					"result": "success",
				},
			},
			str:      `Hello ${{ input "name" }}, status: ${{ from "step1" "result" }}, OS: ${{ .OS }}`,
			expected: "Hello test, status: success, OS: " + runtime.GOOS,
			dryRun:   true,
		},
		{
			name:          "dry run - invalid template syntax",
			str:           "Hello ${{ input",
			expectedError: "unclosed action",
			dryRun:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			result, err := TemplateString(ctx, tc.input, tc.previousOutput, tc.str, tc.dryRun)

			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestMergeWithAndParams(t *testing.T) {
	requiredFalse := false
	requiredTrue := true

	t.Setenv("TEST_ENV_VAR", "env-value")
	t.Setenv("TEST_ENV_BOOL", "true")
	t.Setenv("TEST_ENV_INT", "42")

	tests := []struct {
		name          string
		with          With
		params        InputMap
		expected      With
		expectedError string
	}{
		{
			name:     "empty inputs",
			with:     With{},
			params:   InputMap{},
			expected: With{},
		},
		{
			name: "nil default input parameter",
			with: With{},
			params: InputMap{
				"name": InputParameter{Default: nil, Required: &requiredFalse},
			},
			expected: With{},
		},
		{
			name: "with default values",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Default: "default-name",
				},
				"version": InputParameter{
					Default: "1.0.0",
				},
			},
			expected: With{
				"name":    "default-name",
				"version": "1.0.0",
			},
		},
		{
			name: "with overridden values",
			with: With{
				"name": "custom-name",
			},
			params: InputMap{
				"name": InputParameter{
					Default: "default-name",
				},
				"version": InputParameter{
					Default: "1.0.0",
				},
			},
			expected: With{
				"name":    "custom-name",
				"version": "1.0.0",
			},
		},
		{
			name: "with required parameter missing",
			with: With{},
			params: InputMap{
				"name": InputParameter{},
			},
			expectedError: "missing required input: \"name\"",
		},
		{
			name: "with required parameter explicitly set to true",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Required: &requiredTrue,
				},
			},
			expectedError: "missing required input: \"name\"",
		},
		{
			name: "with required parameter explicitly set to false",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Required: &requiredFalse,
				},
			},
			expected: With{},
		},
		{
			name: "with required parameter provided",
			with: With{
				"name": "custom-name",
			},
			params: InputMap{
				"name": InputParameter{},
			},
			expected: With{
				"name": "custom-name",
			},
		},
		{
			name: "with deprecated parameter",
			with: With{
				"old-param": "value",
			},
			params: InputMap{
				"old-param": InputParameter{
					DeprecatedMessage: "Use new-param instead",
				},
			},
			expected: With{
				"old-param": "value",
			},
		},
		{
			name: "with extra parameters",
			with: With{
				"name":    "custom-name",
				"extra":   "extra-value",
				"another": 123,
			},
			params: InputMap{
				"name": InputParameter{
					Default: "default-name",
				},
			},
			expected: With{
				"name":    "custom-name",
				"extra":   "extra-value",
				"another": 123,
			},
		},
		{
			name: "string input with string default - type match",
			with: With{
				"name": "custom-name",
			},
			params: InputMap{
				"name": InputParameter{
					Default: "default-name",
				},
			},
			expected: With{
				"name": "custom-name",
			},
		},
		{
			name: "string input with non-string default - type cast",
			with: With{
				"count": "10",
			},
			params: InputMap{
				"count": InputParameter{
					Default: 5,
				},
			},
			expected: With{
				"count": 10,
			},
		},
		{
			name: "bool input with bool default - type match",
			with: With{
				"enabled": true,
			},
			params: InputMap{
				"enabled": InputParameter{
					Default: false,
				},
			},
			expected: With{
				"enabled": true,
			},
		},
		{
			name: "bool input with non-bool default - type cast",
			with: With{
				"enabled": true,
			},
			params: InputMap{
				"enabled": InputParameter{
					Default: "false",
				},
			},
			expected: With{
				"enabled": "true",
			},
		},
		{
			name: "int input with int default - type match",
			with: With{
				"count": 10,
			},
			params: InputMap{
				"count": InputParameter{
					Default: 5,
				},
			},
			expected: With{
				"count": 10,
			},
		},
		{
			name: "int input with non-int default - type cast",
			with: With{
				"count": 10,
			},
			params: InputMap{
				"count": InputParameter{
					Default: "5",
				},
			},
			expected: With{
				"count": "10",
			},
		},
		{
			name: "int input with non-int default - failed type cast",
			with: With{
				"count": "hello",
			},
			params: InputMap{
				"count": InputParameter{
					Default: true,
				},
			},
			expectedError: "strconv.ParseBool: parsing \"hello\": invalid syntax",
		},
		{
			name: "unknown type input",
			with: With{
				"data": []string{"a", "b"},
			},
			params: InputMap{
				"data": InputParameter{
					Default: true,
				},
			},
			expectedError: "unable to cast []string{\"a\", \"b\"} of type []string to bool",
		},
		{
			name: "type mismatch with default",
			with: With{
				"count": "not-a-number",
			},
			params: InputMap{
				"count": InputParameter{
					Default: 42,
				},
			},
			expectedError: "unable to cast \"not-a-number\" of type string to int: strconv.ParseInt: parsing \"not-a-number\": invalid syntax",
		},
		{
			name: "valid regex validation passes",
			with: With{
				"name": "Hello World",
			},
			params: InputMap{
				"name": InputParameter{
					Description: "Name with validation",
					Validate:    "^Hello",
				},
			},
			expected: With{
				"name": "Hello World",
			},
		},
		{
			name: "invalid regex validation fails",
			with: With{
				"name": "Goodbye World",
			},
			params: InputMap{
				"name": InputParameter{
					Description: "Name with validation",
					Validate:    "^Hello",
				},
			},
			expectedError: "failed to validate: input=name, value=Goodbye World, regexp=^Hello",
		},
		{
			name: "invalid regex pattern",
			with: With{
				"name": "Hello World",
			},
			params: InputMap{
				"name": InputParameter{
					Description: "Name with validation",
					Validate:    "[", // Invalid regex
				},
			},
			expectedError: "error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "validation with default value passes",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Description: "Name with validation and default",
					Default:     "Hello Default",
					Validate:    "^Hello",
				},
			},
			expected: With{
				"name": "Hello Default",
			},
		},
		{
			name: "validation with good default value bad provided value fails",
			with: With{
				"name": "Goodbye World", // Provide a value that fails validation
			},
			params: InputMap{
				"name": InputParameter{
					Description: "Name with validation and default",
					Default:     "Hello Default", // Default would pass validation
					Validate:    "^Hello",
				},
			},
			expectedError: "failed to validate: input=name, value=Goodbye World, regexp=^Hello",
		},
		{
			name: "validation with bad default value fails",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Description: "Name with validation and default",
					Default:     "Goodbye World",
					Validate:    "^Hello",
				},
			},
			expectedError: "failed to validate: input=name, value=Goodbye World, regexp=^Hello",
		},
		{
			name: "non-string value with validation",
			with: With{
				"count": 42,
			},
			params: InputMap{
				"count": InputParameter{
					Description: "Count with validation",
					Validate:    "^4",
				},
			},
			expected: With{
				"count": 42,
			},
		},
		{
			name: "with default-from-env value",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Description:    "Name from environment",
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: With{
				"name": "env-value",
			},
		},
		{
			name: "with default-from-env for bool value",
			with: With{},
			params: InputMap{
				"enabled": InputParameter{
					Description:    "Boolean from environment",
					DefaultFromEnv: "TEST_ENV_BOOL",
				},
			},
			expected: With{
				"enabled": "true",
			},
		},
		{
			name: "with default-from-env for int value",
			with: With{},
			params: InputMap{
				"count": InputParameter{
					Description:    "Integer from environment",
					DefaultFromEnv: "TEST_ENV_INT",
				},
			},
			expected: With{
				"count": "42",
			},
		},
		{
			name: "with missing environment variable",
			with: With{},
			params: InputMap{
				"missing": InputParameter{
					Description:    "Missing environment variable",
					DefaultFromEnv: "NON_EXISTENT_ENV_VAR",
				},
			},
			expectedError: "environment variable \"NON_EXISTENT_ENV_VAR\" not set and no input provided for \"missing\"",
		},
		{
			name: "with provided value overriding default-from-env",
			with: With{
				"name": "provided-value",
			},
			params: InputMap{
				"name": InputParameter{
					Description:    "Name with provided value",
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: With{
				"name": "provided-value",
			},
		},
		{
			name: "with validation on default-from-env value - passing",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Description:    "Name from environment with validation",
					DefaultFromEnv: "TEST_ENV_VAR",
					Validate:       "^env",
				},
			},
			expected: With{
				"name": "env-value",
			},
		},
		{
			name: "with validation on default-from-env value - failing",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Description:    "Name from environment with validation",
					DefaultFromEnv: "TEST_ENV_VAR",
					Validate:       "^invalid",
				},
			},
			expectedError: "failed to validate: input=name, value=env-value, regexp=^invalid",
		},
		{
			name: "test mutual exclusivity between default and default-from-env",
			with: With{},
			params: InputMap{
				"name": InputParameter{
					Description:    "Name with both default and default-from-env",
					Default:        "default-value",
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: With{
				"name": "default-value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			result, err := MergeWithAndParams(ctx, tc.with, tc.params)

			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestTemplateWithMap(t *testing.T) {
	tests := []struct {
		name           string
		input          With
		previousOutput CommandOutputs
		withMap        map[string]any
		expected       With
		expectedError  string
	}{
		{
			name:     "nil map",
			withMap:  nil,
			expected: nil,
		},
		{
			name:     "empty map",
			withMap:  map[string]any{},
			expected: With{},
		},
		{
			name: "simple string value",
			input: With{
				"name": "test",
			},
			withMap: map[string]any{
				"greeting": "Hello ${{ input \"name\" }}",
			},
			expected: With{
				"greeting": "Hello test",
			},
		},
		{
			name: "nested map",
			input: With{
				"name": "test",
			},
			withMap: map[string]any{
				"config": map[string]any{
					"greeting": "Hello ${{ input \"name\" }}",
					"version":  "1.0",
				},
			},
			expected: With{
				"config": With{
					"greeting": "Hello test",
					"version":  "1.0",
				},
			},
		},
		{
			name: "array with strings",
			input: With{
				"name": "test",
			},
			withMap: map[string]any{
				"greetings": []any{
					"Hello ${{ input \"name\" }}",
					"Hi ${{ input \"name\" }}",
				},
			},
			expected: With{
				"greetings": []any{
					"Hello test",
					"Hi test",
				},
			},
		},
		{
			name: "array with maps",
			input: With{
				"name": "test",
			},
			withMap: map[string]any{
				"users": []any{
					map[string]any{
						"name": "${{ input \"name\" }}",
						"role": "admin",
					},
					map[string]any{
						"name": "other",
						"role": "user",
					},
				},
			},
			expected: With{
				"users": []any{
					With{
						"name": "test",
						"role": "admin",
					},
					With{
						"name": "other",
						"role": "user",
					},
				},
			},
		},
		{
			name: "nested arrays",
			input: With{
				"name": "test",
			},
			withMap: map[string]any{
				"data": []any{
					[]any{
						"${{ input \"name\" }}",
						"value",
					},
				},
			},
			expected: With{
				"data": []any{
					[]any{
						"test",
						"value",
					},
				},
			},
		},
		{
			name: "complex nested structure",
			input: With{
				"name":    "test",
				"version": "2.0",
			},
			previousOutput: CommandOutputs{
				"step1": map[string]any{
					"result": "success",
				},
			},
			withMap: map[string]any{
				"config": map[string]any{
					"app": map[string]any{
						"name":    "${{ input \"name\" }}",
						"version": "${{ input \"version\" }}",
					},
					"status": "${{ from \"step1\" \"result\" }}",
				},
				"data": []any{
					map[string]any{
						"key":   "app_name",
						"value": "${{ input \"name\" }}",
					},
					map[string]any{
						"key":   "app_version",
						"value": "${{ input \"version\" }}",
					},
				},
			},
			expected: With{
				"config": With{
					"app": With{
						"name":    "test",
						"version": "2.0",
					},
					"status": "success",
				},
				"data": []any{
					With{
						"key":   "app_name",
						"value": "test",
					},
					With{
						"key":   "app_version",
						"value": "2.0",
					},
				},
			},
		},
		{
			name:  "with template error",
			input: With{},
			withMap: map[string]any{
				"greeting": "Hello ${{ input \"missing\" }}",
			},
			expectedError: "input \"missing\" does not exist in []",
		},
		{
			name: "non-string primitive values",
			withMap: map[string]any{
				"number":  42,
				"boolean": true,
				"null":    nil,
			},
			expected: With{
				"number":  42,
				"boolean": true,
				"null":    nil,
			},
		},
		{
			name: "With type instead of map[string]any",
			input: With{
				"name": "test",
			},
			withMap: With{
				"greeting": "Hello ${{ input \"name\" }}",
			},
			expected: With{
				"greeting": "Hello test",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			result, err := TemplateWithMap(ctx, tc.input, tc.previousOutput, tc.withMap, false)

			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestTemplateSlice(t *testing.T) {
	tests := []struct {
		name           string
		input          With
		previousOutput CommandOutputs
		slice          []any
		expected       []any
		expectedError  string
	}{
		{
			name:     "empty slice",
			slice:    []any{},
			expected: []any{},
		},
		{
			name: "slice with strings",
			input: With{
				"name": "test",
			},
			slice: []any{
				"Hello ${{ input \"name\" }}",
				"Hi ${{ input \"name\" }}",
			},
			expected: []any{
				"Hello test",
				"Hi test",
			},
		},
		{
			name: "slice with maps",
			input: With{
				"name": "test",
			},
			slice: []any{
				map[string]any{
					"greeting": "Hello ${{ input \"name\" }}",
				},
				map[string]any{
					"greeting": "Hi ${{ input \"name\" }}",
				},
			},
			expected: []any{
				With{
					"greeting": "Hello test",
				},
				With{
					"greeting": "Hi test",
				},
			},
		},
		{
			name: "nested slices",
			input: With{
				"name": "test",
			},
			slice: []any{
				[]any{
					"${{ input \"name\" }}",
					"value",
				},
			},
			expected: []any{
				[]any{
					"test",
					"value",
				},
			},
		},
		{
			name: "non-string primitive values",
			slice: []any{
				42,
				true,
				nil,
			},
			expected: []any{
				42,
				true,
				nil,
			},
		},
		{
			name: "mixed types",
			input: With{
				"name": "test",
			},
			slice: []any{
				"Hello ${{ input \"name\" }}",
				42,
				map[string]any{
					"greeting": "Hi ${{ input \"name\" }}",
				},
				[]any{
					"${{ input \"name\" }}",
					"value",
				},
			},
			expected: []any{
				"Hello test",
				42,
				With{
					"greeting": "Hi test",
				},
				[]any{
					"test",
					"value",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			result, err := templateSlice(ctx, tc.input, tc.previousOutput, tc.slice, false)

			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestPerformLookups(t *testing.T) {
	testCases := []struct {
		name          string
		input         With
		local         With
		previous      CommandOutputs
		expected      With
		expectedError string
	}{
		{
			name: "no lookups",
		},
		{
			name: "invalid template",
			local: With{
				"foo": `${{ input`,
			},
			expectedError: "template: expression evaluator:1: unclosed action",
		},
		{
			name: "simple lookup + builtins",
			input: With{
				"key": "value",
			},
			local: With{
				"key":      "${{ input \"key\" }}",
				"os":       "${{ .OS }}",
				"arch":     "${{ .ARCH }}",
				"platform": "${{ .PLATFORM }}",
				"int":      1,
				"bool":     false,
			},
			expected: With{
				"key":      "value",
				"os":       runtime.GOOS,
				"arch":     runtime.GOARCH,
				"platform": runtime.GOOS + "/" + runtime.GOARCH,
				"int":      1,
				"bool":     false,
			},
		},
		{
			name: "missing input",
			input: With{
				"a": "b",
				"c": "d",
			},
			local: With{
				"key": `${{ input "foo" }}`,
			},
			expectedError: "template: expression evaluator:1:4: executing \"expression evaluator\" at <input \"foo\">: error calling input: input \"foo\" does not exist in [a c]",
		},
		{
			name: "lookup from previous outputs",
			previous: CommandOutputs{
				"step-1": map[string]any{
					"bar": "baz",
				},
			},
			local: With{
				"foo": `${{ from "step-1" "bar" }}`,
			},
			expected: With{
				"foo": "baz",
			},
		},
		{
			name: "lookup from previous outputs - no outputs from step",
			local: With{
				"foo": `${{ from "step-1" "bar" }}`,
			},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <from "step-1" "bar">: error calling from: no outputs from step "step-1"`,
		},
		{
			name: "lookup from previous outputs - missing arg",
			local: With{
				"foo": `${{ from "step-1" }}`,
			},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <from>: wrong number of args for from: want 2 got 1`,
		},
		{
			name: "lookup from previous outputs - output from step not found",
			previous: CommandOutputs{
				"step-1": map[string]any{
					"bar": "baz",
				},
			},
			local: With{
				"foo": `${{ from "step-1" "dne" }}`,
			},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <from "step-1" "dne">: error calling from: no output "dne" from step "step-1"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			templated, err := TemplateWith(ctx, tc.input, tc.local, tc.previous, false)
			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.expected, templated)
		})
	}
}
