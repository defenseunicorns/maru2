// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"

	"github.com/defenseunicorns/maru2/builtins"
	"github.com/invopop/jsonschema"
)

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
		Type:        "string",
		Description: "Environment variable to use as default value for the parameter",
		Pattern:     EnvVariablePattern.String(),
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
		Type:        "string",
		Description: "Regular expression to validate the value of the parameter",
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

// Step is a single step in a task
//
// While a step can have any combination of `run`, and `uses` fields, only one of them should be set
// at a time.
//
// This is enforced by JSON schema validation.
type Step struct {
	// Run is the command/script to run
	Run string `json:"run,omitempty"`
	// Uses is a reference to another task
	Uses string `json:"uses,omitempty"`
	// With is a map of additional parameters for the step/task call
	With `json:"with,omitempty"`
	// ID is a unique identifier for the step
	ID string `json:"id,omitempty"`
	// Name is a human-readable name for the step, pure sugar
	Name string `json:"name,omitempty"`
	// If controls whether the step is executed
	If string `json:"if,omitempty"`
	// Dir is the directory to run the step in
	Dir string `json:"dir,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for a step
func (Step) JSONSchemaExtend(schema *jsonschema.Schema) {
	not := &jsonschema.Schema{
		Not: &jsonschema.Schema{},
	}

	props := jsonschema.NewProperties()
	props.Set("run", &jsonschema.Schema{
		Type:        "string",
		Description: "Command/script to run",
	})
	props.Set("uses", &jsonschema.Schema{
		Type:        "string",
		Description: "Location of a remote task to call conforming to the package URL spec",
	})
	props.Set("id", &jsonschema.Schema{
		Type:        "string",
		Description: "Unique identifier for the step, required to access step outputs",
	})
	props.Set("name", &jsonschema.Schema{
		Type:        "string",
		Description: "Human-readable name for the step, pure sugar",
	})
	props.Set("if", &jsonschema.Schema{
		Type:        "string",
		Description: "Expression that controls whether the step is executed",
		Enum: []any{
			"always",
			"success",
			"failure",
		},
	})
	props.Set("dir", &jsonschema.Schema{
		Type:        "string",
		Description: "Relative directory to run the step in",
	})

	runProps := jsonschema.NewProperties()
	runProps.Set("run", &jsonschema.Schema{
		Type: "string",
	})
	runProps.Set("uses", not)
	oneOfRun := &jsonschema.Schema{
		Required:   []string{"run"},
		Properties: runProps,
	}

	usesProps := jsonschema.NewProperties()
	usesProps.Set("run", not)
	usesProps.Set("uses", &jsonschema.Schema{
		Type: "string",
	})
	oneOfUses := &jsonschema.Schema{
		Required:   []string{"uses"},
		Properties: usesProps,
	}

	var allBuiltinSchemas []*jsonschema.Schema
	reflector := jsonschema.Reflector{ExpandedStruct: true}

	builtinNames := builtins.Names()

	for _, name := range builtinNames {
		builtinEmpty := builtins.Get(name)

		builtinSchema := &jsonschema.Schema{
			If: &jsonschema.Schema{
				Properties: jsonschema.NewProperties(),
			},
			Then: &jsonschema.Schema{
				Properties: jsonschema.NewProperties(),
			},
		}

		builtinSchema.If.Properties.Set("uses", &jsonschema.Schema{
			Type:    "string",
			Pattern: "^builtin:" + name + "(@.*)?$",
		})

		withSchema := reflector.Reflect(builtinEmpty)

		if withSchema != nil {
			withSchema.Description = fmt.Sprintf("Configuration for builtin:%s", name)

			// processSchema allows schema types to be either string or their original type for templating
			var processSchema func(schema *jsonschema.Schema)
			processSchema = func(schema *jsonschema.Schema) {
				if schema.Type == "string" {
					return
				}

				if schema.Type != "array" && schema.Type != "object" {
					schema.OneOf = []*jsonschema.Schema{
						{Type: "string"},
						{Type: schema.Type},
					}
					schema.Type = ""
					return
				}

				if schema.Type == "array" && schema.Items != nil {
					processSchema(schema.Items)
					return
				}
				if schema.Type == "object" && schema.Properties != nil {
					for nestedPair := schema.Properties.Oldest(); nestedPair != nil; nestedPair = nestedPair.Next() {
						processSchema(nestedPair.Value)
					}
				}
			}

			for pair := withSchema.Properties.Oldest(); pair != nil; pair = pair.Next() {
				if pair.Value.Type == "string" {
					continue
				}

				switch pair.Value.Type {
				case "array":
					if pair.Value.Items != nil {
						processSchema(pair.Value.Items)
					}

				case "object":
					if pair.Value.AdditionalProperties != nil && pair.Value.AdditionalProperties != jsonschema.FalseSchema {
						if pair.Value.AdditionalProperties.Type != "string" {
							processSchema(pair.Value.AdditionalProperties)
						}
					} else {
						objectSchema := *pair.Value

						if objectSchema.Properties != nil {
							for nestedPair := objectSchema.Properties.Oldest(); nestedPair != nil; nestedPair = nestedPair.Next() {
								processSchema(nestedPair.Value)
							}
						}

						pair.Value.OneOf = []*jsonschema.Schema{
							{Type: "string"},
							&objectSchema,
						}
						pair.Value.Type = ""
						pair.Value.Properties = nil
						pair.Value.PatternProperties = nil
						pair.Value.AdditionalProperties = nil
					}

				default:
					pair.Value.OneOf = []*jsonschema.Schema{
						{Type: "string"},
						{Type: pair.Value.Type},
					}
					pair.Value.Type = ""
				}
			}
		}

		if withSchema != nil {
			withSchema.ID = jsonschema.EmptyID
			withSchema.Type = "object"
			withSchema.AdditionalProperties = jsonschema.FalseSchema

			builtinSchema.Then.Properties.Set("with", withSchema)

			if len(withSchema.Required) > 0 {
				builtinSchema.Then.Required = []string{"with"}
			}

			allBuiltinSchemas = append(allBuiltinSchemas, builtinSchema)
		}
	}

	var single uint64 = 1

	oneOfGenericWith := &jsonschema.Schema{
		If: &jsonschema.Schema{
			Properties: jsonschema.NewProperties(),
		},
		Then: &jsonschema.Schema{
			Properties: jsonschema.NewProperties(),
		},
	}

	oneOfGenericWith.If.Properties.Set("uses", &jsonschema.Schema{
		Type: "string",
		Not:  &jsonschema.Schema{Pattern: "^builtin:.*$"},
	})

	withSchema := &jsonschema.Schema{
		Type:        "object",
		Description: "Additional parameters for the step/task call\n\nSee https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#passing-inputs",
		MinItems:    &single,
		PatternProperties: map[string]*jsonschema.Schema{
			EnvVariablePattern.String(): {
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
			},
		},
		AdditionalProperties: jsonschema.FalseSchema,
	}

	oneOfGenericWith.Then.Properties.Set("with", withSchema)

	allBuiltinSchemas = append(allBuiltinSchemas, oneOfGenericWith)

	oneOfUses.AllOf = allBuiltinSchemas

	props.Set("with", &jsonschema.Schema{Type: "object"})

	schema.Properties = props
	schema.OneOf = []*jsonschema.Schema{
		oneOfRun,
		oneOfUses,
	}
}
