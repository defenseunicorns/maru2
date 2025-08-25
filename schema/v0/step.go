// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
	"fmt"

	"github.com/invopop/jsonschema"

	"github.com/defenseunicorns/maru2/builtins"
)

// With is a map of string keys and WithEntry values used to pass parameters to called tasks and within steps
type With = map[string]any

// Step is a single step in a task
//
// While a step can have any combination of `run`, and `uses` fields, only one of them should be set
// at a time.
//
// This is enforced by JSON schema validation.
type Step struct {
	// Run is the command/script to run
	Run string `json:"run,omitempty"`
	// Env is a map of environment variables
	Env map[string]any `json:"env,omitempty"`
	// Uses is a reference to another task
	Uses string `json:"uses,omitempty"`
	// With is a map of additional parameters for the step/task call
	With With `json:"with,omitempty"`
	// ID is a unique identifier for the step
	ID string `json:"id,omitempty"`
	// Name is a human-readable name for the step, pure sugar
	Name string `json:"name,omitempty"`
	// If controls whether the step is executed
	If string `json:"if,omitempty"`
	// Dir is the directory to run the step in
	Dir string `json:"dir,omitempty"`
	// Set the shell to execute run with (default: sh)
	Shell string `json:"shell,omitempty"`
	// Set how long to run the command before timing out
	Timeout string `json:"timeout,omitempty"`
	// Mute controls whether the rendered script, STDOUT and STDERR are printed
	//
	// it is similar to set +x and 2>&1 >/dev/null
	Mute bool `json:"mute,omitempty"`
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
	if env, ok := props.Get("env"); ok && env != nil {
		env.Description = "Extra environment variables for this step"
		env.Type = "object"
		env.PropertyNames = &jsonschema.Schema{
			Pattern: EnvVariablePattern.String(),
		}
		env.AdditionalProperties = &jsonschema.Schema{
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
	}
	props.Set("uses", &jsonschema.Schema{
		Type: "string",
		Description: `Location of a task to call

Calling tasks from within the same file: https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#run-another-task-as-a-step
Calling tasks from local files: https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#run-a-task-from-a-local-file
Calling tasks from remote files: https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#run-a-task-from-a-remote-file`,
		Examples: []any{
			"local-task",
			"file:testdata/simple.yaml?task=echo",
			"builtin:echo",
			"pkg:github/defenseunicorns/maru2@main?task=echo",
			"https://raw.githubusercontent.com/defenseunicorns/maru2/main/testdata/simple.yaml?task=echo",
		},
	})
	props.Set("id", &jsonschema.Schema{
		Type: "string",
		Description: `Unique identifier for the step, required to access step outputs

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#passing-outputs`,
	})
	props.Set("name", &jsonschema.Schema{
		Type:        "string",
		Description: "Human-readable name for the step, pure sugar",
	})
	props.Set("if", &jsonschema.Schema{
		Type: "string",
		Description: `Expression that controls whether the step is executed

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#conditional-execution-with-if`,
	})
	props.Set("dir", &jsonschema.Schema{
		Type:        "string",
		Description: "Relative directory to run the step in",
	})
	props.Set("shell", &jsonschema.Schema{
		Type: "string",
		Description: `Set the shell to execute (default: sh)

sh -e -u -c {}
bash -e -u -o pipefail -c {}
pwsh -Command $ErrorActionPreference = 'Stop'; {}; if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }
powershell -Command $ErrorActionPreference = 'Stop'; {}; if ((Test-Path -LiteralPath variable:\LASTEXITCODE)) { exit $LASTEXITCODE }`,
		Enum: []any{"sh", "bash", "pwsh", "powershell"},
	})
	props.Set("timeout", &jsonschema.Schema{
		Type: "string",
		Description: `Set how long to run the command before timing out (e.g., "30s", "1m30s", "1h")

See https://pkg.go.dev/time#ParseDuration for more information.`,
	})
	props.Set("mute", &jsonschema.Schema{
		Type:        "boolean",
		Description: "Mute STDOUT and STDERR for the current script. Has no effect on uses.",
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
	reflector := jsonschema.Reflector{DoNotReference: true}

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
		withSchema.Version = ""

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
