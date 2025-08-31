// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/goccy/go-yaml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

func TestFetchAll(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	workflowNoRefs := v1.Workflow{
		SchemaVersion: v1.SchemaVersion,
		Tasks: v1.TaskMap{
			"default": v1.Task{
				Steps: []v1.Step{
					{Run: "echo 'hello'"},
				},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workflow1.yaml":
			b, _ := yaml.Marshal(workflowNoRefs)
			_, _ = w.Write(b)

		case "/workflow2.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'nested start'"},
							{Uses: "file:workflow3.yaml"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		case "/workflow3.yaml":
			b, _ := yaml.Marshal(workflowNoRefs)
			_, _ = w.Write(b)

		case "/error.yaml":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))

		case "/invalid.yaml":
			_, _ = w.Write([]byte("not a valid workflow yaml"))

		case "/nested404.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: "file:dne.yaml"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tests := []struct {
		name        string
		wf          v1.Workflow
		expectedErr string
	}{
		{
			name: "no references",
			wf:   workflowNoRefs,
		},
		{
			name: "with empty uses field",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'start'"},
							{Uses: ""},
							{Run: "echo 'end'"},
						},
					},
				},
			},
		},
		{
			name: "with uses referring to internal task",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'start'"},
							{Uses: "another-task"},
							{Run: "echo 'end'"},
						},
					},
					"another-task": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'internal task'"},
						},
					},
				},
			},
		},
		{
			name: "with builtin uses",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'start'"},
							{Uses: "builtin:foo"},
							{Run: "echo 'end'"},
						},
					},
				},
			},
		},
		{
			name: "with exact duplicate uses strings",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/workflow1.yaml"},
							{Uses: server.URL + "/workflow1.yaml"},
						},
					},
				},
			},
		},
		{
			name: "with duplicate references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/workflow1.yaml?task=default"},
							{Uses: server.URL + "/workflow1.yaml?task=other"},
						},
					},
				},
			},
		},
		{
			name: "with remote references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/workflow1.yaml"},
						},
					},
				},
			},
		},
		{
			name: "with nested remote references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/workflow2.yaml"},
						},
					},
				},
			},
		},
		{
			name: "with_invalid_remote_references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/invalid.yaml"},
						},
					},
				},
			},
			expectedErr: "[1:1] string was used where mapping is expected\n>  1 | not a valid workflow yaml\n       ^\n",
		},
		{
			name: "with_server_error_references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/error.yaml"},
						},
					},
				},
			},
			expectedErr: fmt.Sprintf("get \"%s/error.yaml\": 500 Internal Server Error", server.URL),
		},
		{
			name: "with_invalid_url_references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: "invalid:///url"},
						},
					},
				},
			},
			expectedErr: `failed to resolve "invalid:///url": unsupported scheme: "invalid" in "invalid:///url"`,
		},
		{
			name: "with_non_existent_references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/non-existent.yaml"},
						},
					},
				},
			},
			expectedErr: fmt.Sprintf("get \"%s/non-existent.yaml\": 404 Not Found", server.URL),
		},
		{
			name: "with invalid url references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: "invalid:///url"},
						},
					},
				},
			},
			expectedErr: `failed to resolve "invalid:///url": unsupported scheme: "invalid" in "invalid:///url"`,
		},
		{
			name: "with nested non_existent_references",
			wf: v1.Workflow{
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: server.URL + "/nested404.yaml"},
						},
					},
				},
			},
			expectedErr: fmt.Sprintf("get \"%s/dne.yaml\": 404 Not Found", server.URL),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			err := FetchAll(ctx, svc, tt.wf, nil)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestListAllLocal(t *testing.T) {
	ctx := log.WithContext(t.Context(), log.New(io.Discard))

	tests := []struct {
		name         string
		files        map[string]string
		srcURL       string
		expectedRefs []string
		expectErr    string
	}{
		{
			name:         "non-file scheme returns empty",
			srcURL:       "https://example.com/workflow.yaml",
			expectedRefs: nil,
		},
		{
			name: "workflow with no local references",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - run: "echo hello"
`,
			},
			srcURL:       "file:tasks.yaml",
			expectedRefs: []string{"file:tasks.yaml"},
		},
		{
			name: "workflow with single local reference",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "file:dep.yaml?task=dep"
`,
				"dep.yaml": `
schema-version: v0
tasks:
  dep:
    - run: "echo dep"
`,
			},
			srcURL:       "file:tasks.yaml",
			expectedRefs: []string{"file:tasks.yaml", "file:dep.yaml"},
		},
		{
			name: "workflow with nested local references",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "file:dep1.yaml?task=dep1"
`,
				"dep1.yaml": `
schema-version: v0
tasks:
  dep1:
    - uses: "file:dep2.yaml?task=dep2"
`,
				"dep2.yaml": `
schema-version: v0
tasks:
  dep2:
    - run: "echo dep2"
`,
			},
			srcURL:       "file:tasks.yaml",
			expectedRefs: []string{"file:tasks.yaml", "file:dep1.yaml", "file:dep2.yaml"},
		},
		{
			name: "workflow with mixed remote and local references",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "https://example.com/remote.yaml"
    - uses: "file:local.yaml?task=local"
`,
				"local.yaml": `
schema-version: v0
tasks:
  local:
    - run: "echo local"
`,
			},
			srcURL:       "file:tasks.yaml",
			expectedRefs: []string{"file:tasks.yaml", "file:local.yaml"},
		},
		{
			name: "workflow with duplicate local references",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "file:dep.yaml?task=dep1"
    - uses: "file:dep.yaml?task=dep2"
`,
				"dep.yaml": `
schema-version: v0
tasks:
  dep1:
    - run: "echo dep1"
  dep2:
    - run: "echo dep2"
`,
			},
			srcURL:       "file:tasks.yaml",
			expectedRefs: []string{"file:tasks.yaml", "file:dep.yaml"},
		},
		{
			name: "workflow with invalid URL in uses",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "::invalid-url"
`,
			},
			srcURL:    "file:tasks.yaml",
			expectErr: "missing protocol scheme",
		},
		{
			name:      "non-existent file",
			files:     map[string]string{},
			srcURL:    "file:nonexistent.yaml",
			expectErr: "file does not exist",
		},
		{
			name: "invalid workflow syntax",
			files: map[string]string{
				"invalid.yaml": "not: a: valid: workflow",
			},
			srcURL:    "file:invalid.yaml",
			expectErr: "mapping value is not allowed in this context",
		},
		{
			name: "non-existent local dependency",
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "file:nonexistent.yaml?task=task"
`,
			},
			srcURL:    "file:tasks.yaml",
			expectErr: "file does not exist",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			for path, content := range tc.files {
				err := afero.WriteFile(fs, path, []byte(content), 0644)
				require.NoError(t, err)
			}

			srcURL, err := url.Parse(tc.srcURL)
			require.NoError(t, err)

			refs, err := ListAllLocal(ctx, srcURL, fs)

			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}

			require.NoError(t, err)
			assert.ElementsMatch(t, tc.expectedRefs, refs)
		})
	}
}

func TestExecuteUses(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	workflowFoo := v1.Workflow{SchemaVersion: v1.SchemaVersion, Tasks: v1.TaskMap{"default": v1.Task{Steps: []v1.Step{{Run: "echo 'foo'"}, {Uses: "file:bar/baz.yaml?task=baz"}}}}}
	workflowBaz := v1.Workflow{SchemaVersion: v1.SchemaVersion, Tasks: v1.TaskMap{"baz": v1.Task{Steps: []v1.Step{{Run: "echo 'baz'"}, {Uses: "file:../hello-world.yaml"}}}}}

	handleWF := func(w http.ResponseWriter, wf v1.Workflow) {
		b, err := yaml.Marshal(wf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_, err = w.Write(b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hello-world.yaml":
			handleWF(w, v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"default": v1.Task{Steps: []v1.Step{{Run: "echo 'Hello World!'"}}},
					"a-task":  v1.Task{Steps: []v1.Step{{Run: "echo 'task a'"}}},
					"task-b":  v1.Task{Steps: []v1.Step{{Run: "echo 'task b'"}}},
				},
			})
		case "/foo.yaml":
			handleWF(w, workflowFoo)
		case "/bar/baz.yaml":
			handleWF(w, workflowBaz)
		case "/bad.yaml":
			_, _ = w.Write([]byte("not a workflow"))
		case "/timeout.yaml":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(server.Close)

	helloWorldURL := server.URL + "/hello-world.yaml"

	dummyOrigin := "file:tasks.yaml"

	tests := []struct {
		name        string
		uses        string
		origin      string
		aliases     v1.AliasMap
		skipShort   bool
		expectedErr string
	}{
		{
			name:   "local file",
			uses:   "file:testdata/hello-world.yaml",
			origin: dummyOrigin,
		},
		{
			name:   "local file with task",
			uses:   "file:testdata/hello-world.yaml?task=a-task",
			origin: dummyOrigin,
		},
		{
			name:   "http url",
			uses:   helloWorldURL,
			origin: dummyOrigin,
		},
		{
			name:        "missing scheme",
			uses:        "./path-with-no-scheme",
			origin:      dummyOrigin,
			expectedErr: `unsupported scheme: "" in "./path-with-no-scheme"`,
		},
		{
			name:        "invalid control character in URL",
			uses:        "http://www.example.com/\x7f",
			origin:      dummyOrigin,
			expectedErr: `parse "http://www.example.com/\x7f": net/url: invalid control character in URL`,
		},
		{
			name:        "unsupported scheme",
			uses:        "ssh:not-supported",
			origin:      dummyOrigin,
			expectedErr: `unsupported scheme: "ssh" in "ssh:not-supported"`,
		},
		{
			name:        "unsupported package type",
			uses:        "pkg:bitbucket/owner/repo",
			origin:      dummyOrigin,
			expectedErr: `unsupported package type: "bitbucket"`,
		},
		{
			name:   "with map based resolver",
			uses:   "pkg:custom/noxsios/mar2-test?task=hello-world",
			origin: dummyOrigin,
			aliases: v1.AliasMap{
				"custom": {
					Type: "gitlab",
				},
			},
			skipShort: true,
		},
		{
			name:      "pkg scheme with github",
			uses:      "file:..?task=hello-world",
			origin:    "pkg:github/defenseunicorns/maru2#testdata/hello-world.yaml",
			skipShort: true,
		},
		{
			name:   "nested uses foo.yaml -> baz.yaml -> hello-world.yaml",
			uses:   server.URL + "/foo.yaml",
			origin: dummyOrigin,
		},
		{
			name:        "bad workflow",
			uses:        server.URL + "/bad.yaml",
			origin:      dummyOrigin,
			expectedErr: "[1:1] string was used where mapping is expected\n>  1 | not a workflow\n       ^\n",
		},
		{
			name:        "failed to fetch",
			uses:        server.URL + "/non-existent.yaml",
			origin:      dummyOrigin,
			expectedErr: fmt.Sprintf("get %q: 404 Not Found", server.URL+"/non-existent.yaml"),
		},
		{
			name:        "timeout",
			uses:        server.URL + "/timeout.yaml",
			origin:      dummyOrigin,
			expectedErr: fmt.Sprintf("Get %q: context deadline exceeded (Client.Timeout exceeded while awaiting headers)", server.URL+"/timeout.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			if tt.skipShort && testing.Short() {
				t.Skip("skipping test in short mode")
			}

			origin, err := url.Parse(tt.origin)
			require.NoError(t, err)

			if tt.expectedErr == "" {
				_, err := handleUsesStep(ctx, svc, v1.Step{Uses: tt.uses}, v1.Workflow{Aliases: tt.aliases}, v1.With{}, nil, origin, "", nil, false)
				require.NoError(t, err)
			} else {
				_, err := handleUsesStep(ctx, svc, v1.Step{Uses: tt.uses}, v1.Workflow{Aliases: tt.aliases}, v1.With{}, nil, origin, "", nil, false)
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}

func TestUsesEnvironmentVariables(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	// Create a test workflow that accepts inputs and accesses environment variables
	testWorkflow := v0.Workflow{
		SchemaVersion: v0.SchemaVersion,
		Inputs: v0.InputMap{
			"message": {
				Description: "Test message input",
				Default:     "default-message",
			},
		},
		Tasks: v0.TaskMap{
			"env-test": {
				v0.Step{
					Run: `
						echo "Parent env: $PARENT_ENV"
						echo "Input message: $INPUT_MESSAGE"
						echo "result=success" >> $MARU2_OUTPUT
					`,
				},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/env-test.yaml" {
			b, _ := yaml.Marshal(testWorkflow)
			_, _ = w.Write(b)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tests := []struct {
		name        string
		step        v1.Step
		environ     []string
		withInputs  v1.With
		expectedErr string
	}{
		{
			name: "environment variables passed through uses",
			step: v1.Step{
				Uses: server.URL + "/env-test.yaml?task=env-test",
				With: v1.With{
					"message": "test-from-parent",
				},
			},
			environ: []string{
				"PARENT_ENV=parent-value",
				"PATH=/usr/bin",
			},
			withInputs: v1.With{},
		},
		{
			name: "step-level env not passed to uses (current behavior)",
			step: v1.Step{
				Uses: server.URL + "/env-test.yaml?task=env-test",
				Env: v1.Env{
					"STEP_LEVEL_VAR": "step-value",
				},
				With: v1.With{
					"message": "test-with-step-env",
				},
			},
			environ: []string{
				"PARENT_ENV=parent-value",
				"PATH=/usr/bin",
			},
			withInputs: v1.With{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			origin, err := url.Parse("file:test.yaml")
			require.NoError(t, err)

			result, err := handleUsesStep(
				ctx,
				svc,
				tt.step,
				v1.Workflow{},
				tt.withInputs,
				nil,
				origin,
				"",
				tt.environ,
				true, // dry run to avoid actual execution
			)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				// In dry run mode, result should be nil since no actual execution happens
				assert.Nil(t, result)
			}
		})
	}
}

func TestUsesEnvironmentVariablesExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that executes real workflows in short mode")
	}

	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	ctx := log.WithContext(t.Context(), log.New(io.Discard))

	// Test using the existing testdata files
	origin, err := url.Parse("file:testdata/hello-world.yaml")
	require.NoError(t, err)

	wf, err := Fetch(ctx, svc, origin)
	require.NoError(t, err)

	// Set up environment variables that should be passed through
	environ := []string{
		"TEST_PARENT_ENV=test-parent-value",
		"PATH=/usr/bin",
	}

	// Execute a simple task to verify environment variables are accessible
	// This doesn't test the specific environment variable passing but confirms
	// the basic uses functionality works with environment context
	result, err := Run(ctx, svc, wf, "default", v1.With{}, origin, "", environ, false)
	require.NoError(t, err)

	// For this simple test, we just verify no error occurred
	// The comprehensive environment variable testing is covered by the E2E test
	// Result is nil for tasks that don't write to $MARU2_OUTPUT, which is expected
	assert.Nil(t, result)
}
