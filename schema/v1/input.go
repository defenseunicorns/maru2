// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"cmp"
	"iter"
	"slices"

	"github.com/invopop/jsonschema"
)

// InputMap defines input parameters for task execution
//
// Maps parameter names to their definitions including validation, defaults, and documentation
type InputMap map[string]InputParameter

// JSONSchemaExtend restricts input parameter names to valid patterns
//
// Enforces naming conventions for input parameters (kebab-case, alphanumeric + hyphens)
func (InputMap) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.PropertyNames = &jsonschema.Schema{
		Pattern: InputNamePattern.String(),
	}
}

// OrderedSeq returns an iterator over input parameter names and values in alphabetical order by name
func (im InputMap) OrderedSeq() iter.Seq2[string, InputParameter] {
	names := make([]string, 0, len(im))
	for name := range im {
		names = append(names, name)
	}
	slices.SortStableFunc(names, cmp.Compare)
	return func(yield func(string, InputParameter) bool) {
		for _, name := range names {
			input := im[name]
			if !yield(name, input) {
				return
			}
		}
	}
}

// InputParameter defines a single input parameter for tasks and steps
//
// Supports validation, default values from environment variables,
// deprecation warnings, and required/optional configuration
type InputParameter struct {
	// Description of the input parameter
	Description string `json:"description"`
	// Message to display when the parameter is deprecated
	DeprecatedMessage string `json:"deprecated-message,omitempty"`
	// Whether the parameter is required, defaults to true
	Required *bool `json:"required,omitempty"`
	// Default value for the parameter, can be a string or a primitive type
	Default any `json:"default,omitempty"`
	// Environment variable to use as default value for the parameter
	DefaultFromEnv string `json:"default-from-env,omitempty"`
	// Regular expression to validate the value of the parameter
	Validate string `json:"validate,omitempty"`
}

// JSONSchemaExtend generates detailed schema documentation for input parameters
//
// Creates comprehensive validation rules for parameter configuration including
// type constraints, validation patterns, and environment variable integration
func (InputParameter) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Description = "Input parameter for the step"

	schema.Properties.Set("description", &jsonschema.Schema{
		Type:        "string",
		Description: "Description of the parameter",
	})

	schema.Properties.Set("deprecated-message", &jsonschema.Schema{
		Type:        "string",
		Description: "Message to display when the parameter is deprecated",
	})

	schema.Properties.Set("required", &jsonschema.Schema{
		Type:        "boolean",
		Description: "Whether the parameter is required",
		Default:     true,
	})

	schema.Properties.Set("validate", &jsonschema.Schema{
		Type: "string",
		Description: `Regular expression to validate the value of the parameter

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#input-validation`,
	})

	schema.Properties.Set("default", &jsonschema.Schema{
		Description: "Default value for the parameter, can be a string or a primitive type",
		OneOf: []*jsonschema.Schema{
			{
				Type: "string",
			},
			{
				Type: "boolean",
			},
			{
				Type: "integer",
			},
		},
	})
	schema.Properties.Set("default-from-env", &jsonschema.Schema{
		Type: "string",
		Description: `Environment variable to use as default value for the parameter

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#default-values-from-environment-variables`,
		Pattern: EnvVariablePattern.String(),
	})
}
