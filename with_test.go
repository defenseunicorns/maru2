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
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			result, err := TemplateString(ctx, tc.input, tc.previousOutput, tc.str, false)

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
				"name": InputParameter{
					Required: true,
				},
			},
			expectedError: "missing required input: \"name\"",
		},
		{
			name: "with required parameter provided",
			with: With{
				"name": "custom-name",
			},
			params: InputMap{
				"name": InputParameter{
					Required: true,
				},
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
	}

	for _, tc := range tests {
		tc := tc
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
				"greetings": []interface{}{
					"Hello ${{ input \"name\" }}",
					"Hi ${{ input \"name\" }}",
				},
			},
			expected: With{
				"greetings": []interface{}{
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
				"users": []interface{}{
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
				"users": []interface{}{
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
				"data": []interface{}{
					[]interface{}{
						"${{ input \"name\" }}",
						"value",
					},
				},
			},
			expected: With{
				"data": []interface{}{
					[]interface{}{
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
				"data": []interface{}{
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
				"data": []interface{}{
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
		tc := tc
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
		tc := tc
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
		tc := tc
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
