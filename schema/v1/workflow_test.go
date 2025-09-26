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

func TestWorkflowExplain(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	// Complex workflow with all features
	complexWorkflow := Workflow{
		SchemaVersion: SchemaVersion,
		Aliases: AliasMap{
			"gh": Alias{
				Type:         "github",
				BaseURL:      "https://api.github.com",
				TokenFromEnv: "GITHUB_TOKEN",
			},
			"local": Alias{
				Path: "common/tasks.yaml",
			},
		},
		Tasks: TaskMap{
			"default": Task{
				Description: "Default build task",
				Collapse:    true,
				Inputs: InputMap{
					"version": InputParameter{
						Description:    "Version to build",
						Required:       boolPtr(true),
						Default:        "latest",
						DefaultFromEnv: "BUILD_VERSION",
						Validate:       `^v?\d+\.\d+\.\d+$`,
					},
					"debug": InputParameter{
						Description:       "Enable debug mode",
						Required:          boolPtr(false),
						Default:           false,
						DeprecatedMessage: "Use --verbose instead",
					},
				},
				Steps: []Step{
					{
						Name: "Setup environment",
						ID:   "setup",
						Run:  "export PATH=$PATH:/usr/local/bin",
						Env: map[string]any{
							"NODE_ENV": "production",
							"DEBUG":    "${{ input \"debug\" }}",
						},
						Dir:     "src",
						Shell:   "bash",
						Timeout: "30s",
						Show:    boolPtr(false),
					},
					{
						Uses: "gh:defenseunicorns/maru2@main?task=build",
						With: map[string]any{
							"version": "${{ input \"version\" }}",
							"target":  "linux",
						},
						If:   "input(\"debug\") == false",
						Mute: true,
					},
					{
						Run: "echo 'Build completed'",
					},
				},
			},
			"test": Task{
				Steps: []Step{
					{Run: "go test ./..."},
				},
			},
		},
	}

	testCases := []struct {
		name        string
		workflow    Workflow
		taskNames   []string
		contains    []string
		notContains []string
	}{
		{
			name:     "simple workflow - all tasks",
			workflow: helloWorldWorkflow,
			contains: []string{
				"# Workflow (v1)",
				"## Tasks",
				"### `default` (Default Task)",
				"### `a-task`",
				"### `task-b`",
				"echo 'Hello World!'",
				"echo 'task a'",
				"echo 'task b'",
				"## Usage",
				"maru2                    # Run default task",
			},
		},
		{
			name:      "simple workflow - specific task",
			workflow:  helloWorldWorkflow,
			taskNames: []string{"default"},
			contains: []string{
				"# Workflow (v1)",
				"## Tasks",
				"### `default` (Default Task)",
				"echo 'Hello World!'",
			},
			notContains: []string{
				"### `a-task`",
				"### `task-b`",
				"## Usage",
			},
		},
		{
			name:     "complex workflow with all features",
			workflow: complexWorkflow,
			contains: []string{
				"# Workflow (v1)",
				"## Aliases",
				"| Name | Type | Details |",
				"|------|------|----------|",
				"| `gh` | Package URL | github at `https://api.github.com` (auth: `$GITHUB_TOKEN`) |",
				"| `local` | Local File | `common/tasks.yaml` |",
				"## Tasks",
				"### `default` (Default Task)",
				"Default build task",
				"*Output will be grouped in CI environments (GitHub Actions, GitLab CI)*",
				"**Input Parameters:**",
				"| Name | Description | Required | Default | Validation | Notes |",
				"|------|-------------|----------|---------|------------|-------|",
				"| `debug` | Enable debug mode | No | `false` | - | ⚠️ **Deprecated**: Use --verbose instead |",
				"| `version` | Version to build | Yes | `latest` | `^v?\\d+\\.\\d+\\.\\d+$` | - |",
				"**Steps:**",
				"1. **Setup environment** (`setup`)",
				"```bash",
				"export PATH=$PATH:/usr/local/bin",
				"Uses: `gh:defenseunicorns/maru2@main?task=build`",
				"- `version`: `${{ input \"version\" }}`",
				"- `target`: `linux`",
				"*Configuration:* Working directory: `src` • Timeout: `30s` • Script hidden • Environment variables: 2 set",
				"*Configuration:* Condition: `input(\"debug\") == false` • Output muted",
				"### `test`",
				"go test ./...",
			},
		},
		{
			name:      "non-existent task",
			workflow:  helloWorldWorkflow,
			taskNames: []string{"non-existent"},
			contains: []string{
				"# Workflow (v1)",
				"## Tasks",
				"No tasks found.",
			},
			notContains: []string{
				"### `default`",
				"## Usage",
			},
		},
		{
			name: "empty workflow",
			workflow: Workflow{
				SchemaVersion: SchemaVersion,
				Tasks:         TaskMap{},
			},
			contains: []string{
				"# Workflow (v1)",
				"## Tasks",
				"No tasks found.",
			},
			notContains: []string{
				"## Aliases",
				"## Usage",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.workflow.Explain(tc.taskNames...)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected, "Expected to find: %s", expected)
			}

			for _, unexpected := range tc.notContains {
				assert.NotContains(t, result, unexpected, "Expected NOT to find: %s", unexpected)
			}
		})
	}
}
