// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"cmp"
	"slices"

	"github.com/invopop/jsonschema"
)

// DefaultTaskName is the default task name
const DefaultTaskName = "default"

// DefaultFileName is the default file name
const DefaultFileName = "tasks.yaml"

// Workflow is a wrapper struct around the input map and task map
//
// It represents a "tasks.yaml" file
type Workflow struct {
	Inputs InputMap `json:"inputs,omitempty"`
	Tasks  TaskMap  `json:"tasks,omitempty"`
}

// Task is a list of steps
type Task []Step

// TaskMap is a map of tasks, where the key is the task name
type TaskMap map[string]Task

// Find returns a task by name
func (tm TaskMap) Find(call string) (Task, bool) {
	task, ok := tm[call]
	return task, ok
}

// OrderedTaskNames returns a list of task names in alphabetical order
//
// The default task is always first
func (tm TaskMap) OrderedTaskNames() []string {
	names := make([]string, 0, len(tm))
	for k := range tm {
		names = append(names, k)
	}
	slices.SortStableFunc(names, func(a, b string) int {
		if a == DefaultTaskName {
			return -1
		}
		if b == DefaultTaskName {
			return 1
		}
		return cmp.Compare(a, b)
	})
	return names
}

// WorkFlowSchema returns a JSON schema for a maru2 workflow
func WorkFlowSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{}
	reflector.ExpandedStruct = true
	schema := reflector.Reflect(&TaskMap{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json"

	inputSchema := reflector.Reflect(&InputParameter{})
	inputSchema.ID = jsonschema.EmptyID
	inputSchema.Description = "Input parameter for the workflow"
	schema.Definitions["Input"] = inputSchema

	schema.AdditionalProperties = jsonschema.FalseSchema
	var single uint64 = 1
	schema.PatternProperties = map[string]*jsonschema.Schema{
		"^x-": &jsonschema.Schema{
			Type: "object",
		},
		TaskNamePattern.String(): {
			If: &jsonschema.Schema{
				Type: "array",
			},
			Then: &jsonschema.Schema{
				Description: "Name of the task",
				Ref:         "#/$defs/Task",
			},
			Else: &jsonschema.Schema{
				If: &jsonschema.Schema{
					Type: "object",
				},
				Then: &jsonschema.Schema{
					Ref: "#/$defs/Input",
				},
			},
			MinItems: &single,
		},
	}

	return schema
}
