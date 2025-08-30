// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"cmp"
	"slices"

	"github.com/invopop/jsonschema"
)

// DefaultTaskName is the default task name
const DefaultTaskName = "default"

// Task is a list of steps
type Task []Step

// JSONSchemaExtend extends the JSON schema for a task
func (Task) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Description = "A task definition, aka a collection of steps"
}

// TaskMap is a map of tasks, where the key is the task name
type TaskMap map[string]Task

// JSONSchemaExtend extends the JSON schema for a task map
func (TaskMap) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.PropertyNames = &jsonschema.Schema{
		Pattern: TaskNamePattern.String(),
	}
}

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
