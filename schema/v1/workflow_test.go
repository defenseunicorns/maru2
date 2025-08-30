// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helloWorldWorkflow is a simple workflow that prints "Hello World!"
// do not make changes to this variable within tests
var helloWorldWorkflow = Workflow{
	SchemaVersion: SchemaVersion,
	Tasks: TaskMap{
		"default": {Step{Run: "echo 'Hello World!'"}},
		"a-task":  {Step{Run: "echo 'task a'"}},
		"task-b":  {Step{Run: "echo 'task b'"}},
	},
}

func TestWorkflowFind(t *testing.T) {
	task, ok := helloWorldWorkflow.Tasks.Find(DefaultTaskName)
	assert.True(t, ok)

	require.Len(t, task, 1)
	assert.Equal(t, "echo 'Hello World!'", task[0].Run)

	task, ok = helloWorldWorkflow.Tasks.Find("foo")
	assert.Nil(t, task)
	assert.False(t, ok)
}

func TestOrderedTaskNames(t *testing.T) {
	names := helloWorldWorkflow.Tasks.OrderedTaskNames()
	expected := []string{"default", "a-task", "task-b"}
	assert.ElementsMatch(t, expected, names)

	wf := Workflow{Tasks: TaskMap{"foo": nil, "bar": nil, "baz": nil, "default": nil}}
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
