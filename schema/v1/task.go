// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"cmp"
	"iter"
	"slices"

	"github.com/invopop/jsonschema"

	"github.com/defenseunicorns/maru2/schema"
)

// Task is a list of steps and input parameters
type Task struct {
	Description string   `json:"description,omitempty"`
	Collapse    bool     `json:"collapse,omitempty"`
	Inputs      InputMap `json:"inputs,omitempty"`
	Steps       []Step   `json:"steps"`
}

// JSONSchemaExtend extends the JSON schema for a task
func (Task) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Description = "A task definition, aka a collection of steps"

	if desc, ok := schema.Properties.Get("description"); ok && desc != nil {
		desc.Description = "Human-readable description of the task"
	}

	if collapse, ok := schema.Properties.Get("collapse"); ok && collapse != nil {
		collapse.Description = "Group task output in CI environments (GitHub Actions, GitLab CI)"
	}

	if inputs, ok := schema.Properties.Get("inputs"); ok && inputs != nil {
		inputs.Description = "Input parameters for the task"
	}
	if steps, ok := schema.Properties.Get("steps"); ok && steps != nil {
		steps.Description = "Task steps"
	}
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
//
// Yes, this function is essentially syntactic sugar for Go map functionality, but I like it, so I'm keeping it
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
		if a == schema.DefaultTaskName {
			return -1
		}
		if b == schema.DefaultTaskName {
			return 1
		}
		return cmp.Compare(a, b)
	})
	return names
}

func (tm TaskMap) OrderedSeq() iter.Seq2[string, Task] {
	names := tm.OrderedTaskNames()
	return func(yield func(string, Task) bool) {
		for _, name := range names {
			task := tm[name]
			if !yield(name, task) {
				return
			}
		}
	}
}
