// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import "github.com/invopop/jsonschema"

// InputMap is a map of input parameters for a workflow
type InputMap map[string]InputParameter

// InputParameter represents a single input parameter for a workflow, to be used w/ `with`
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

// JSONSchemaExtend extends the JSON schema for a step
func (InputParameter) JSONSchemaExtend(schema *jsonschema.Schema) {
	defaultSchema := &jsonschema.Schema{
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
	}

	defaultFromEnvSchema := &jsonschema.Schema{
		Type: "string",
		Description: `Environment variable to use as default value for the parameter

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#default-values-from-environment-variables`,
		Pattern: EnvVariablePattern.String(),
	}

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

	schema.Properties.Set("default", defaultSchema)
	schema.Properties.Set("default-from-env", defaultFromEnvSchema)

	// Add a constraint to ensure they are mutually exclusive
	schema.DependentRequired = map[string][]string{
		"default":          {},
		"default-from-env": {},
	}

	schema.OneOf = []*jsonschema.Schema{
		{
			Required: []string{"default"},
			Not: &jsonschema.Schema{
				Required: []string{"default-from-env"},
			},
		},
		{
			Required: []string{"default-from-env"},
			Not: &jsonschema.Schema{
				Required: []string{"default"},
			},
		},
		{
			Not: &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{
					{
						Required: []string{"default"},
					},
					{
						Required: []string{"default-from-env"},
					},
				},
			},
		},
	}
}
