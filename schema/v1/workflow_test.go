// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/schema"
)

// helloWorldWorkflow is a simple workflow that prints "Hello World!"
// do not make changes to this variable within tests
var helloWorldWorkflow = Workflow{
	SchemaVersion: SchemaVersion,
	Tasks: TaskMap{
		"default": Task{
			Inputs: InputMap{},
			Steps:  []Step{{Run: "echo 'Hello World!'"}},
		},
		"a-task": Task{
			Inputs: InputMap{},
			Steps:  []Step{{Run: "echo 'task a'"}},
		},
		"task-b": Task{
			Inputs: InputMap{},
			Steps:  []Step{{Run: "echo 'task b'"}},
		},
	},
}

func TestWorkflowFind(t *testing.T) {
	task, ok := helloWorldWorkflow.Tasks.Find(schema.DefaultTaskName)
	assert.True(t, ok)

	require.Len(t, task.Steps, 1)
	assert.Equal(t, "echo 'Hello World!'", task.Steps[0].Run)

	task, ok = helloWorldWorkflow.Tasks.Find("foo")
	assert.False(t, ok)
	assert.Equal(t, Task{}, task)
}

func TestOrderedTaskNames(t *testing.T) {
	names := helloWorldWorkflow.Tasks.OrderedTaskNames()
	expected := []string{"default", "a-task", "task-b"}
	assert.ElementsMatch(t, expected, names)

	wf := Workflow{Tasks: TaskMap{
		"foo":     Task{Inputs: InputMap{}, Steps: []Step{}},
		"bar":     Task{Inputs: InputMap{}, Steps: []Step{}},
		"baz":     Task{Inputs: InputMap{}, Steps: []Step{}},
		"default": Task{Inputs: InputMap{}, Steps: []Step{}},
	}}
	names = wf.Tasks.OrderedTaskNames()
	expected = []string{"default", "bar", "baz", "foo"}
	assert.ElementsMatch(t, expected, names)

	delete(wf.Tasks, "default")

	names = wf.Tasks.OrderedTaskNames()
	expected = []string{"bar", "baz", "foo"}
	assert.ElementsMatch(t, expected, names)
}

func TestWorkflowSchemaGen(t *testing.T) {
	schema := WorkFlowSchema()

	assert.NotNil(t, schema)

	b, err := json.Marshal(schema)
	require.NoError(t, err)

	current, err := os.ReadFile("schema.json")
	require.NoError(t, err)

	assert.JSONEq(t, string(current), string(b))
}

func TestOrderedTasks(t *testing.T) {
	testCases := []struct {
		name     string
		tasks    TaskMap
		expected []string
	}{
		{
			name:     "nil",
			tasks:    nil,
			expected: []string{},
		},
		{
			name:     "empty",
			tasks:    TaskMap{},
			expected: []string{},
		},
		{
			name: "single task",
			tasks: TaskMap{
				"build": Task{},
			},
			expected: []string{"build"},
		},
		{
			name: "single default task",
			tasks: TaskMap{
				"default": Task{},
			},
			expected: []string{"default"},
		},
		{
			name: "multiple tasks - sorted order",
			tasks: TaskMap{
				"zebra": Task{},
				"alpha": Task{},
				"beta":  Task{},
			},
			expected: []string{"alpha", "beta", "zebra"},
		},
		{
			name: "multiple tasks with default - default first",
			tasks: TaskMap{
				"zebra":   Task{},
				"default": Task{},
				"alpha":   Task{},
			},
			expected: []string{"default", "alpha", "zebra"},
		},
		{
			name: "tasks with similar names",
			tasks: TaskMap{
				"task-2":  Task{},
				"task-10": Task{},
				"task-1":  Task{},
			},
			expected: []string{"task-1", "task-10", "task-2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make([]string, 0)
			for name := range tc.tasks.OrderedSeq() {
				got = append(got, name)
			}
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("partial iteration", func(t *testing.T) {
		tasks := TaskMap{
			"zebra":   Task{},
			"default": Task{},
			"alpha":   Task{},
			"gamma":   Task{},
		}

		got := make([]string, 0)
		for name := range tasks.OrderedSeq() {
			got = append(got, name)
			if len(got) == 2 {
				break
			}
		}

		expected := []string{"default", "alpha"}
		assert.Equal(t, expected, got)
	})
}
