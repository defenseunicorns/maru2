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

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestTemplateString(t *testing.T) {
	tests := []struct {
		name           string
		input          schema.With
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
			input:    schema.With{"name": "test"},
			str:      "hello ${{ input \"name\" }}",
			expected: "hello test",
		},
		{
			name:          "with missing input",
			input:         schema.With{},
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
			input: schema.With{"name": "test"},
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
		{
			name:     "with which shortcut",
			str:      "shortcut: ${{ which \"foo\" }}",
			expected: "shortcut: bar",
			dryRun:   false,
		},
		{
			name:          "with missing which shortcut",
			str:           "shortcut: ${{ which \"missing\" }}",
			expectedError: "exec: \"missing\": executable file not found in $PATH",
			dryRun:        false,
		},
		{
			name:     "dry run - with which shortcut",
			str:      "shortcut: ${{ which \"foo\" }}",
			expected: "shortcut: bar",
			dryRun:   true,
		},
		{
			name:          "dry run - with missing which shortcut",
			str:           "shortcut: ${{ which \"missing\" }}",
			expectedError: "exec: \"missing\": executable file not found in $PATH",
			dryRun:        true,
		},
		{
			name:     "dry run - no template",
			str:      "hello world",
			expected: "hello world",
			dryRun:   true,
		},
		{
			name:     "dry run - with input",
			input:    schema.With{"name": "test"},
			str:      `hello ${{ input "name" }}`,
			expected: "hello test",
			dryRun:   true,
		},
		{
			name:     "dry run - with missing input",
			input:    schema.With{},
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
			input: schema.With{"name": "test"},
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

	// Register a shortcut for "which" tests
	RegisterWhichShortcut("foo", "bar")
	t.Cleanup(func() {
		shortcuts.Clear()
	})

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
		with          schema.With
		params        v1.InputMap
		expected      schema.With
		expectedError string
	}{
		{
			name:     "empty inputs",
			with:     schema.With{},
			params:   v1.InputMap{},
			expected: schema.With{},
		},
		{
			name: "nil default input parameter",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{Default: nil, Required: &requiredFalse},
			},
			expected: schema.With{},
		},
		{
			name: "with default values",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Default: "default-name",
				},
				"version": v1.InputParameter{
					Default: "1.0.0",
				},
			},
			expected: schema.With{
				"name":    "default-name",
				"version": "1.0.0",
			},
		},
		{
			name: "with overridden values",
			with: schema.With{
				"name": "custom-name",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Default: "default-name",
				},
				"version": v1.InputParameter{
					Default: "1.0.0",
				},
			},
			expected: schema.With{
				"name":    "custom-name",
				"version": "1.0.0",
			},
		},
		{
			name: "with required parameter missing",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{},
			},
			expectedError: "missing required input: \"name\"",
		},
		{
			name: "with required parameter explicitly set to true",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Required: &requiredTrue,
				},
			},
			expectedError: "missing required input: \"name\"",
		},
		{
			name: "with required parameter explicitly set to false",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Required: &requiredFalse,
				},
			},
			expected: schema.With{},
		},
		{
			name: "with required parameter provided",
			with: schema.With{
				"name": "custom-name",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{},
			},
			expected: schema.With{
				"name": "custom-name",
			},
		},
		{
			name: "with deprecated parameter",
			with: schema.With{
				"old-param": "value",
			},
			params: v1.InputMap{
				"old-param": v1.InputParameter{
					DeprecatedMessage: "Use new-param instead",
				},
			},
			expected: schema.With{
				"old-param": "value",
			},
		},
		{
			name: "with extra parameters",
			with: schema.With{
				"name":    "custom-name",
				"extra":   "extra-value",
				"another": 123,
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Default: "default-name",
				},
			},
			expected: schema.With{
				"name":    "custom-name",
				"extra":   "extra-value",
				"another": 123,
			},
		},
		{
			name: "string input with string default - type match",
			with: schema.With{
				"name": "custom-name",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Default: "default-name",
				},
			},
			expected: schema.With{
				"name": "custom-name",
			},
		},
		{
			name: "string input with non-string default - type cast",
			with: schema.With{
				"count": "10",
			},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Default: 5,
				},
			},
			expected: schema.With{
				"count": 10,
			},
		},
		{
			name: "bool input with bool default - type match",
			with: schema.With{
				"enabled": true,
			},
			params: v1.InputMap{
				"enabled": v1.InputParameter{
					Default: false,
				},
			},
			expected: schema.With{
				"enabled": true,
			},
		},
		{
			name: "bool input with non-bool default - type cast",
			with: schema.With{
				"enabled": true,
			},
			params: v1.InputMap{
				"enabled": v1.InputParameter{
					Default: "false",
				},
			},
			expected: schema.With{
				"enabled": "true",
			},
		},
		{
			name: "int input with int default - type match",
			with: schema.With{
				"count": 10,
			},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Default: 5,
				},
			},
			expected: schema.With{
				"count": 10,
			},
		},
		{
			name: "int input with non-int default - type cast",
			with: schema.With{
				"count": 10,
			},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Default: "5",
				},
			},
			expected: schema.With{
				"count": "10",
			},
		},
		{
			name: "int input with non-int default - failed type cast",
			with: schema.With{
				"count": "hello",
			},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Default: true,
				},
			},
			expectedError: "strconv.ParseBool: parsing \"hello\": invalid syntax",
		},
		{
			name: "uint64 input with uint64 default - type match",
			with: schema.With{
				"size": uint64(1024),
			},
			params: v1.InputMap{
				"size": v1.InputParameter{
					Default: uint64(512),
				},
			},
			expected: schema.With{
				"size": uint64(1024),
			},
		},
		{
			name: "uint64 input with non-uint64 default - type cast",
			with: schema.With{
				"size": "2048",
			},
			params: v1.InputMap{
				"size": v1.InputParameter{
					Default: uint64(512),
				},
			},
			expected: schema.With{
				"size": uint64(2048),
			},
		},
		{
			name: "uint64 input with non-uint64 default - failed type cast",
			with: schema.With{
				"size": "not-a-number",
			},
			params: v1.InputMap{
				"size": v1.InputParameter{
					Default: uint64(512),
				},
			},
			expectedError: "unable to cast \"not-a-number\" of type string to uint64: strconv.ParseUint: parsing \"not-a-number\": invalid syntax",
		},
		{
			name: "unsupported type default - slice type",
			with: schema.With{
				"data": "some-value",
			},
			params: v1.InputMap{
				"data": v1.InputParameter{
					Default: []string{"default", "values"},
				},
			},
			expectedError: "unable to cast input \"data\" from string to []string",
		},
		{
			name: "unsupported type default - map type",
			with: schema.With{
				"config": "some-value",
			},
			params: v1.InputMap{
				"config": v1.InputParameter{
					Default: map[string]string{"key": "value"},
				},
			},
			expectedError: "unable to cast input \"config\" from string to map[string]string",
		},
		{
			name: "unknown type input",
			with: schema.With{
				"data": []string{"a", "b"},
			},
			params: v1.InputMap{
				"data": v1.InputParameter{
					Default: true,
				},
			},
			expectedError: "unable to cast []string{\"a\", \"b\"} of type []string to bool",
		},
		{
			name: "type mismatch with default",
			with: schema.With{
				"count": "not-a-number",
			},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Default: 42,
				},
			},
			expectedError: "unable to cast \"not-a-number\" of type string to int: strconv.ParseInt: parsing \"not-a-number\": invalid syntax",
		},
		{
			name: "valid regex validation passes",
			with: schema.With{
				"name": "Hello World",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description: "Name with validation",
					Validate:    "^Hello",
				},
			},
			expected: schema.With{
				"name": "Hello World",
			},
		},
		{
			name: "invalid regex validation fails",
			with: schema.With{
				"name": "Goodbye World",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description: "Name with validation",
					Validate:    "^Hello",
				},
			},
			expectedError: "failed to validate: input=name, value=Goodbye World, regexp=^Hello",
		},
		{
			name: "invalid regex pattern",
			with: schema.With{
				"name": "Hello World",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description: "Name with validation",
					Validate:    "[", // Invalid regex
				},
			},
			expectedError: "error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "validation with default value passes",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description: "Name with validation and default",
					Default:     "Hello Default",
					Validate:    "^Hello",
				},
			},
			expected: schema.With{
				"name": "Hello Default",
			},
		},
		{
			name: "validation with good default value bad provided value fails",
			with: schema.With{
				"name": "Goodbye World", // Provide a value that fails validation
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description: "Name with validation and default",
					Default:     "Hello Default", // Default would pass validation
					Validate:    "^Hello",
				},
			},
			expectedError: "failed to validate: input=name, value=Goodbye World, regexp=^Hello",
		},
		{
			name: "validation with bad default value fails",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description: "Name with validation and default",
					Default:     "Goodbye World",
					Validate:    "^Hello",
				},
			},
			expectedError: "failed to validate: input=name, value=Goodbye World, regexp=^Hello",
		},
		{
			name: "non-string value with validation",
			with: schema.With{
				"count": 42,
			},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Description: "Count with validation",
					Validate:    "^4",
				},
			},
			expected: schema.With{
				"count": 42,
			},
		},
		{
			name: "with default-from-env value",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description:    "Name from environment",
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: schema.With{
				"name": "env-value",
			},
		},
		{
			name: "with default-from-env for bool value",
			with: schema.With{},
			params: v1.InputMap{
				"enabled": v1.InputParameter{
					Description:    "Boolean from environment",
					DefaultFromEnv: "TEST_ENV_BOOL",
				},
			},
			expected: schema.With{
				"enabled": "true",
			},
		},
		{
			name: "with default-from-env for int value",
			with: schema.With{},
			params: v1.InputMap{
				"count": v1.InputParameter{
					Description:    "Integer from environment",
					DefaultFromEnv: "TEST_ENV_INT",
				},
			},
			expected: schema.With{
				"count": "42",
			},
		},
		{
			name: "with missing environment variable",
			with: schema.With{},
			params: v1.InputMap{
				"missing": v1.InputParameter{
					Description:    "Missing environment variable",
					DefaultFromEnv: "NON_EXISTENT_ENV_VAR",
				},
			},
			expected: schema.With{},
		},
		{
			name: "with provided value overriding default-from-env",
			with: schema.With{
				"name": "provided-value",
			},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description:    "Name with provided value",
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: schema.With{
				"name": "provided-value",
			},
		},
		{
			name: "with validation on default-from-env value - passing",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description:    "Name from environment with validation",
					DefaultFromEnv: "TEST_ENV_VAR",
					Validate:       "^env",
				},
			},
			expected: schema.With{
				"name": "env-value",
			},
		},
		{
			name: "with validation on default-from-env value - failing",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description:    "Name from environment with validation",
					DefaultFromEnv: "TEST_ENV_VAR",
					Validate:       "^invalid",
				},
			},
			expectedError: "failed to validate: input=name, value=env-value, regexp=^invalid",
		},
		{
			name: "test priority order: default-from-env over default",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description:    "Name with both default and default-from-env",
					Default:        "default-value",
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: schema.With{
				"name": "env-value",
			},
		},
		{
			name: "test fallback from missing env var to default",
			with: schema.With{},
			params: v1.InputMap{
				"name": v1.InputParameter{
					Description:    "Name with both default and missing default-from-env",
					Default:        "fallback-value",
					DefaultFromEnv: "NON_EXISTENT_ENV_VAR",
				},
			},
			expected: schema.With{
				"name": "fallback-value",
			},
		},
		{
			name: "nil with parameter creates new map",
			with: nil,
			params: v1.InputMap{
				"name": v1.InputParameter{
					Default: "default-value",
				},
			},
			expected: schema.With{
				"name": "default-value",
			},
		},
		{
			name: "nil with parameter with required input missing",
			with: nil,
			params: v1.InputMap{
				"name": v1.InputParameter{
					Required: &requiredTrue,
				},
			},
			expectedError: "missing required input: \"name\"",
		},
		{
			name: "nil with parameter with env var",
			with: nil,
			params: v1.InputMap{
				"name": v1.InputParameter{
					DefaultFromEnv: "TEST_ENV_VAR",
				},
			},
			expected: schema.With{
				"name": "env-value",
			},
		},
		{
			name: "validation with non-string value that cannot be cast to string",
			with: schema.With{
				"data": complex(1, 2), // complex numbers cannot be cast to string
			},
			params: v1.InputMap{
				"data": v1.InputParameter{
					Validate: "^test",
				},
			},
			expectedError: "unable to cast (1+2i) of type complex128 to string",
		},
		{
			name: "string casting error in type matching section",
			with: schema.With{
				"data": complex(1, 2), // complex numbers cannot be cast to string
			},
			params: v1.InputMap{
				"data": v1.InputParameter{
					Default: "string-default", // This will trigger string casting
				},
			},
			expectedError: "unable to cast (1+2i) of type complex128 to string",
		},
		{
			name: "nil with parameter requiring default assignment triggers map creation",
			with: nil, // This ensures merged starts as nil
			params: v1.InputMap{
				"name": v1.InputParameter{
					Default:  "test-value",
					Required: &requiredFalse, // Ensure it's not required
				},
			},
			expected: schema.With{
				"name": "test-value",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
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
		input          schema.With
		previousOutput CommandOutputs
		withMap        map[string]any
		expected       schema.With
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
			expected: nil,
		},
		{
			name: "simple string value",
			input: schema.With{
				"name": "test",
			},
			withMap: map[string]any{
				"greeting": "Hello ${{ input \"name\" }}",
			},
			expected: schema.With{
				"greeting": "Hello test",
			},
		},
		{
			name: "nested map",
			input: schema.With{
				"name": "test",
			},
			withMap: map[string]any{
				"config": map[string]any{
					"greeting": "Hello ${{ input \"name\" }}",
					"version":  "1.0",
				},
			},
			expected: schema.With{
				"config": schema.With{
					"greeting": "Hello test",
					"version":  "1.0",
				},
			},
		},
		{
			name: "array with strings",
			input: schema.With{
				"name": "test",
			},
			withMap: map[string]any{
				"greetings": []any{
					"Hello ${{ input \"name\" }}",
					"Hi ${{ input \"name\" }}",
				},
			},
			expected: schema.With{
				"greetings": []any{
					"Hello test",
					"Hi test",
				},
			},
		},
		{
			name: "array with maps",
			input: schema.With{
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
			expected: schema.With{
				"users": []any{
					schema.With{
						"name": "test",
						"role": "admin",
					},
					schema.With{
						"name": "other",
						"role": "user",
					},
				},
			},
		},
		{
			name: "nested arrays",
			input: schema.With{
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
			expected: schema.With{
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
			input: schema.With{
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
			expected: schema.With{
				"config": schema.With{
					"app": schema.With{
						"name":    "test",
						"version": "2.0",
					},
					"status": "success",
				},
				"data": []any{
					schema.With{
						"key":   "app_name",
						"value": "test",
					},
					schema.With{
						"key":   "app_version",
						"value": "2.0",
					},
				},
			},
		},
		{
			name:  "with template error",
			input: schema.With{},
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
			expected: schema.With{
				"number":  42,
				"boolean": true,
				"null":    nil,
			},
		},
		{
			name: "With type instead of map[string]any",
			input: schema.With{
				"name": "test",
			},
			withMap: schema.With{
				"greeting": "Hello ${{ input \"name\" }}",
			},
			expected: schema.With{
				"greeting": "Hello test",
			},
		},
		{
			name:  "nested map with template error",
			input: schema.With{},
			withMap: map[string]any{
				"config": map[string]any{
					"greeting": "Hello ${{ input \"missing\" }}",
				},
			},
			expectedError: "input \"missing\" does not exist in []",
		},
		{
			name:  "slice with template error",
			input: schema.With{},
			withMap: map[string]any{
				"items": []any{
					"Hello ${{ input \"missing\" }}",
				},
			},
			expectedError: "input \"missing\" does not exist in []",
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
		input          schema.With
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
			input: schema.With{
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
			input: schema.With{
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
				schema.With{
					"greeting": "Hello test",
				},
				schema.With{
					"greeting": "Hi test",
				},
			},
		},
		{
			name: "nested slices",
			input: schema.With{
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
			input: schema.With{
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
				schema.With{
					"greeting": "Hi test",
				},
				[]any{
					"test",
					"value",
				},
			},
		},
		{
			name: "slice with nil values",
			slice: []any{
				nil,
				"test",
				nil,
			},
			expected: []any{
				nil,
				"test",
				nil,
			},
		},
		{
			name: "slice with unsupported type (function) - should pass through",
			slice: []any{
				42,
				true,
			},
			expected: []any{
				42,
				true,
			},
		},
		{
			name: "slice with nested slices containing maps",
			input: schema.With{
				"name": "test",
			},
			slice: []any{
				[]any{
					map[string]any{
						"key": "${{ input \"name\" }}",
					},
				},
			},
			expected: []any{
				[]any{
					schema.With{
						"key": "test",
					},
				},
			},
		},
		{
			name: "slice with deeply nested structure",
			input: schema.With{
				"value": "nested",
			},
			slice: []any{
				map[string]any{
					"level1": map[string]any{
						"level2": []any{
							"${{ input \"value\" }}",
						},
					},
				},
			},
			expected: []any{
				schema.With{
					"level1": schema.With{
						"level2": []any{
							"nested",
						},
					},
				},
			},
		},
		{
			name: "slice with template error in nested map",
			slice: []any{
				map[string]any{
					"key": "${{ invalid template syntax",
				},
			},
			expectedError: "function \"invalid\" not defined",
		},
		{
			name: "slice with template error in nested slice",
			slice: []any{
				[]any{
					"${{ invalid template syntax",
				},
			},
			expectedError: "function \"invalid\" not defined",
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
		input         schema.With
		local         schema.With
		previous      CommandOutputs
		expected      schema.With
		expectedError string
	}{
		{
			name: "no lookups",
		},
		{
			name: "invalid template",
			local: schema.With{
				"foo": `${{ input`,
			},
			expectedError: "template: expression evaluator:1: unclosed action",
		},
		{
			name: "simple lookup + builtins",
			input: schema.With{
				"key": "value",
			},
			local: schema.With{
				"key":      "${{ input \"key\" }}",
				"os":       "${{ .OS }}",
				"arch":     "${{ .ARCH }}",
				"platform": "${{ .PLATFORM }}",
				"int":      1,
				"bool":     false,
			},
			expected: schema.With{
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
			input: schema.With{
				"a": "b",
				"c": "d",
			},
			local: schema.With{
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
			local: schema.With{
				"foo": `${{ from "step-1" "bar" }}`,
			},
			expected: schema.With{
				"foo": "baz",
			},
		},
		{
			name: "lookup from previous outputs - no outputs from step",
			local: schema.With{
				"foo": `${{ from "step-1" "bar" }}`,
			},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <from "step-1" "bar">: error calling from: no outputs from step "step-1"`,
		},
		{
			name: "lookup from previous outputs - missing arg",
			local: schema.With{
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
			local: schema.With{
				"foo": `${{ from "step-1" "dne" }}`,
			},
			expectedError: `template: expression evaluator:1:4: executing "expression evaluator" at <from "step-1" "dne">: error calling from: no output "dne" from step "step-1"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			templated, err := TemplateWithMap(ctx, tc.input, tc.previous, tc.local, false)
			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.expected, templated)
		})
	}
}
