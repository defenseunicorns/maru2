// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"cmp"
	"slices"

	"github.com/defenseunicorns/maru2/config"
	"github.com/invopop/jsonschema"
)

// DefaultTaskName is the default task name
const DefaultTaskName = "default"

// Workflow is a wrapper struct around the input map and task map
//
// It represents a "tasks.yaml" file
type Workflow struct {
	Inputs  InputMap                `json:"inputs,omitempty"`
	Tasks   TaskMap                 `json:"tasks,omitempty"`
	Aliases map[string]config.Alias `json:"aliases,omitempty"`
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
	reflector := jsonschema.Reflector{ExpandedStruct: true}
	schema := reflector.Reflect(&Workflow{})

	schema.ID = "https://raw.githubusercontent.com/defenseunicorns/maru2/main/maru2.schema.json"

	return schema
}
