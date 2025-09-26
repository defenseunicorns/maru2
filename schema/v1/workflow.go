// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
)

// SchemaVersion is the current schema version for workflows
const SchemaVersion = "v1"

// Workflow represents a "tasks.yaml" file
type Workflow struct {
	SchemaVersion string   `json:"schema-version"`
	Aliases       AliasMap `json:"aliases,omitempty"`
	Tasks         TaskMap  `json:"tasks,omitempty"`
}

// JSONSchemaExtend extends the JSON schema for a workflow
func (Workflow) JSONSchemaExtend(schema *jsonschema.Schema) {
	if schemaVersion, ok := schema.Properties.Get("schema-version"); ok && schemaVersion != nil {
		schemaVersion.Description = "Workflow schema version."
		schemaVersion.Enum = []any{SchemaVersion}
		schemaVersion.AdditionalProperties = jsonschema.FalseSchema
	}
	if tasks, ok := schema.Properties.Get("tasks"); ok && tasks != nil {
		tasks.Description = "Map of tasks where the key is the task name, the task named 'default' is called when no task is specified"
	}
	if aliases, ok := schema.Properties.Get("aliases"); ok && aliases != nil {
		aliases.Description = `Aliases for package URLs or local file paths to create shorthand references
See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#package-url-aliases

See https://github.com/defenseunicorns/maru2/blob/main/docs/syntax.md#local-file-aliases
`
	}
}

func (wf Workflow) Explain(taskNames ...string) string {
	var explanation strings.Builder

	explanation.WriteString(fmt.Sprintf("> for schema version %s\n", wf.SchemaVersion))
	explanation.WriteString(">\n")
	explanation.WriteString(fmt.Sprintf("> %s\n\n", "<https://TODO GRAB URL FROM SCHEMA CONST>"))

	// Aliases section if present
	if len(wf.Aliases) > 0 {
		explanation.WriteString("## Aliases\n\n")
		explanation.WriteString("Shortcuts for referencing remote repositories and local files:\n\n")
		explanation.WriteString("| Name | Type | Details |\n")
		explanation.WriteString("|------|------|----------|\n")

		for aliasName, alias := range wf.Aliases.OrderedSeq() {
			if alias.Path != "" {
				// Local file alias
				explanation.WriteString(fmt.Sprintf("| `%s` | Local File | `%s` |\n", aliasName, alias.Path))
			} else {
				// Package URL alias
				details := alias.Type
				if alias.BaseURL != "" {
					details += fmt.Sprintf(" at `%s`", alias.BaseURL)
				}
				if alias.TokenFromEnv != "" {
					details += fmt.Sprintf(" (auth: `$%s`)", alias.TokenFromEnv)
				}
				explanation.WriteString(fmt.Sprintf("| `%s` | Package URL | %s |\n", aliasName, details))
			}
		}
		explanation.WriteString("\n")
	}

	// Tasks section
	explanation.WriteString("## Tasks\n\n")

	// Determine which tasks to explain
	var tasksToExplain []string
	if len(taskNames) > 0 {
		// Filter to only requested tasks
		for _, taskName := range taskNames {
			if _, exists := wf.Tasks[taskName]; exists {
				tasksToExplain = append(tasksToExplain, taskName)
			}
		}
	} else {
		// Explain all tasks in order
		tasksToExplain = wf.Tasks.OrderedTaskNames()
	}

	if len(tasksToExplain) == 0 {
		explanation.WriteString("No tasks found.\n")
		return explanation.String()
	}

	// Explain each task
	for i, taskName := range tasksToExplain {
		if i > 0 {
			explanation.WriteString("\n---\n\n")
		}

		task := wf.Tasks[taskName]

		// Task header
		if taskName == "default" {
			explanation.WriteString("### `default` (Default Task)\n\n")
		} else {
			explanation.WriteString(fmt.Sprintf("### `%s`\n\n", taskName))
		}

		// Task description
		if task.Description != "" {
			explanation.WriteString(fmt.Sprintf("%s\n\n", task.Description))
		}

		// CI integration info
		if task.Collapse {
			explanation.WriteString("*Output will be grouped in CI environments (GitHub Actions, GitLab CI)*\n\n")
		}

		// Input parameters
		if len(task.Inputs) > 0 {
			explanation.WriteString("**Input Parameters:**\n\n")
			explanation.WriteString("| Name | Description | Required | Default | Validation | Notes |\n")
			explanation.WriteString("|------|-------------|----------|---------|------------|-------|\n")

			for inputName, param := range task.Inputs.OrderedSeq() {
				// Name
				name := fmt.Sprintf("`%s`", inputName)

				// Description
				description := param.Description

				// Required/optional
				required := "Yes"
				if param.Required != nil && !*param.Required {
					required = "No"
				}

				// Default value
				defaultValue := "-"
				if param.Default != nil {
					defaultValue = fmt.Sprintf("`%v`", param.Default)
				} else if param.DefaultFromEnv != "" {
					defaultValue = fmt.Sprintf("`$%s`", param.DefaultFromEnv)
				}

				// Validation
				validation := "-"
				if param.Validate != "" {
					validation = fmt.Sprintf("`%s`", param.Validate)
				}

				// Notes (deprecation)
				notes := "-"
				if param.DeprecatedMessage != "" {
					notes = fmt.Sprintf("⚠️ **Deprecated**: %s", param.DeprecatedMessage)
				}

				explanation.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
					name, description, required, defaultValue, validation, notes))
			}
			explanation.WriteString("\n")
		}

		// Steps
		explanation.WriteString("**Steps:**\n\n")
		if len(task.Steps) == 0 {
			explanation.WriteString("No steps defined.\n\n")
			continue
		}

		for stepIdx, step := range task.Steps {
			stepNum := stepIdx + 1

			// Step header with optional name and ID
			stepHeader := fmt.Sprintf("%d.", stepNum)
			if step.Name != "" {
				stepHeader += fmt.Sprintf(" **%s**", step.Name)
			}
			if step.ID != "" {
				stepHeader += fmt.Sprintf(" (`%s`)", step.ID)
			}
			explanation.WriteString(stepHeader + "\n")

			// Step action (run vs uses)
			if step.Run != "" {
				// Shell command
				explanation.WriteString("   ```")
				if step.Shell != "" && step.Shell != "sh" {
					explanation.WriteString(step.Shell)
				} else {
					explanation.WriteString("sh")
				}
				explanation.WriteString("\n   " + step.Run + "\n   ```\n")
			} else if step.Uses != "" {
				// Task reference
				explanation.WriteString(fmt.Sprintf("   Uses: `%s`\n", step.Uses))

				// With parameters
				if len(step.With) > 0 {
					explanation.WriteString("   With:\n")
					for key, value := range step.With {
						explanation.WriteString(fmt.Sprintf("   - `%s`: `%v`\n", key, value))
					}
				}
			}

			// Step configuration details
			var stepDetails []string

			if step.Dir != "" {
				stepDetails = append(stepDetails, fmt.Sprintf("Working directory: `%s`", step.Dir))
			}

			if step.If != "" {
				stepDetails = append(stepDetails, fmt.Sprintf("Condition: `%s`", step.If))
			}

			if step.Timeout != "" {
				stepDetails = append(stepDetails, fmt.Sprintf("Timeout: `%s`", step.Timeout))
			}

			if step.Mute {
				stepDetails = append(stepDetails, "Output muted")
			}

			if step.Show != nil && !*step.Show {
				stepDetails = append(stepDetails, "Script hidden")
			}

			if len(step.Env) > 0 {
				stepDetails = append(stepDetails, fmt.Sprintf("Environment variables: %d set", len(step.Env)))
			}

			if len(stepDetails) > 0 {
				explanation.WriteString("   \n   *Configuration:* " + strings.Join(stepDetails, " • ") + "\n")
			}

			explanation.WriteString("\n")
		}
	}

	// Usage instructions
	if len(taskNames) == 0 {
		explanation.WriteString("---\n\n")
		explanation.WriteString("## Usage\n\n")
		explanation.WriteString("Run tasks using:\n")
		explanation.WriteString("```sh\n")
		if _, hasDefault := wf.Tasks["default"]; hasDefault {
			explanation.WriteString("maru2                    # Run default task\n")
		}
		explanation.WriteString("maru2 <task-name>        # Run specific task\n")
		explanation.WriteString("maru2 <task> --with key=value  # Pass input parameters\n")
		explanation.WriteString("```\n")
	}

	return explanation.String()
}

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/schema/v1/schema.json"

	return schema
}
