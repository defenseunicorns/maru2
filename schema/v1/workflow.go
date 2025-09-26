// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"fmt"
	"slices"
	"strings"

	"github.com/invopop/jsonschema"
)

// SchemaVersion is the current schema version for workflows
const SchemaVersion = "v1"

// SchemaURL is the URL to the generated schema on GitHub
const SchemaURL = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/schema/v1/schema.json"

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

// Explain generates a markdown explanation of the workflow and its tasks
func (wf Workflow) Explain(taskNames ...string) string {
	var explanation strings.Builder

	if len(taskNames) == 0 {
		explanation.WriteString(fmt.Sprintf("> for schema version %s\n", wf.SchemaVersion))
		explanation.WriteString(">\n")
		explanation.WriteString(fmt.Sprintf("> <%s>\n\n", SchemaURL))

		if len(wf.Aliases) > 0 {
			explanation.WriteString("## Aliases\n\n")
			explanation.WriteString("Shortcuts for referencing remote repositories and local files:\n\n")
			explanation.WriteString("| Name | Type | Details |\n")
			explanation.WriteString("|------|------|----------|\n")

			for aliasName, alias := range wf.Aliases.OrderedSeq() {
				if alias.Path != "" {
					explanation.WriteString(fmt.Sprintf("| `%s` | Local File | `%s` |\n", aliasName, alias.Path))
				} else {
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

		explanation.WriteString("## Tasks\n\n")
	}

	for name, task := range wf.Tasks.OrderedSeq() {
		// note: this will not error if you ask to explain a task the does not exist in the workflow
		if len(taskNames) > 0 && !slices.Contains(taskNames, name) {
			continue
		}

		if name == "default" {
			explanation.WriteString("### `default` (Default Task)\n\n")
		} else {
			explanation.WriteString(fmt.Sprintf("### `%s`\n\n", name))
		}

		if task.Description != "" {
			explanation.WriteString(fmt.Sprintf("%s\n\n", task.Description))
		}

		if task.Collapse {
			explanation.WriteString("*Output will be grouped in CI environments (GitHub Actions, GitLab CI)*\n\n")
		}

		if len(task.Inputs) > 0 {
			explanation.WriteString("**Input Parameters:**\n\n")
			explanation.WriteString("| Name | Description | Required | Default | Validation | Notes |\n")
			explanation.WriteString("|------|-------------|----------|---------|------------|-------|\n")

			for inputName, param := range task.Inputs.OrderedSeq() {
				name := fmt.Sprintf("`%s`", inputName)

				description := param.Description

				required := "Yes"
				if param.Required != nil && !*param.Required {
					required = "No"
				}

				defaultValue := "-"
				if param.Default != nil {
					defaultValue = fmt.Sprintf("`%v`", param.Default)
				} else if param.DefaultFromEnv != "" {
					defaultValue = fmt.Sprintf("`$%s`", param.DefaultFromEnv)
				}

				validation := "-"
				if param.Validate != "" {
					validation = fmt.Sprintf("`%s`", param.Validate)
				}

				notes := "-"
				if param.DeprecatedMessage != "" {
					notes = fmt.Sprintf("⚠️ **Deprecated**: %s", param.DeprecatedMessage)
				}

				explanation.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
					name, description, required, defaultValue, validation, notes))
			}
			explanation.WriteString("\n")
		}

		uses := []string{}
		for _, step := range task.Steps {
			if step.Uses != "" {
				uses = append(uses, step.Uses)
			}
		}
		uses = slices.Compact(uses)

		if len(uses) > 0 {
			explanation.WriteString("**Uses:**\n\n")
			for _, u := range uses {
				explanation.WriteString(fmt.Sprintf("- `%s`\n", u))
			}
			explanation.WriteString("\n\n")
		}
	}

	return explanation.String()
}

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{DoNotReference: true, ExpandedStruct: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = SchemaURL

	return schema
}
