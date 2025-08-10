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
	"github.com/stretchr/testify/require"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	"github.com/defenseunicorns/maru2/uses"
)

func TestFetchAll(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	workflowNoRefs := v0.Workflow{
		SchemaVersion: v0.SchemaVersion,
		Tasks: v0.TaskMap{
			"default": {v0.Step{Run: "echo 'hello'"}},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workflow1.yaml":
			b, _ := yaml.Marshal(workflowNoRefs)
			_, _ = w.Write(b)

		case "/workflow2.yaml":
			wf := v0.Workflow{
				SchemaVersion: v0.SchemaVersion,
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Run: "echo 'nested start'"},
						v0.Step{Uses: "file:workflow3.yaml"},
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
			wf := v0.Workflow{
				SchemaVersion: v0.SchemaVersion,
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: "file:dne.yaml"},
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
		wf          v0.Workflow
		expectedErr string
	}{
		{
			name: "no references",
			wf:   workflowNoRefs,
		},
		{
			name: "with empty uses field",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Run: "echo 'start'"},
						v0.Step{Uses: ""},
						v0.Step{Run: "echo 'end'"},
					},
				},
			},
		},
		{
			name: "with uses referring to internal task",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Run: "echo 'start'"},
						v0.Step{Uses: "another-task"},
						v0.Step{Run: "echo 'end'"},
					},
					"another-task": {
						v0.Step{Run: "echo 'internal task'"},
					},
				},
			},
		},
		{
			name: "with builtin uses",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Run: "echo 'start'"},
						v0.Step{Uses: "builtin:foo"},
						v0.Step{Run: "echo 'end'"},
					},
				},
			},
		},
		{
			name: "with exact duplicate uses strings",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/workflow1.yaml"},
						v0.Step{Uses: server.URL + "/workflow1.yaml"},
					},
				},
			},
		},
		{
			name: "with duplicate references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/workflow1.yaml?task=default"},
						v0.Step{Uses: server.URL + "/workflow1.yaml?task=other"},
					},
				},
			},
		},
		{
			name: "with remote references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/workflow1.yaml"},
					},
				},
			},
		},
		{
			name: "with nested remote references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/workflow2.yaml"},
					},
				},
			},
		},
		{
			name: "with_invalid_remote_references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/invalid.yaml"},
					},
				},
			},
			expectedErr: "[1:1] string was used where mapping is expected\n>  1 | not a valid workflow yaml\n       ^\n",
		},
		{
			name: "with_server_error_references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/error.yaml"},
					},
				},
			},
			expectedErr: "get \"" + server.URL + "/error.yaml\": 500 Internal Server Error",
		},
		{
			name: "with_invalid_url_references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: "invalid:///url"},
					},
				},
			},
			expectedErr: "failed to resolve \"invalid:///url\": unsupported scheme: \"invalid\" in \"invalid:///url\"",
		},
		{
			name: "with_non_existent_references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/non-existent.yaml"},
					},
				},
			},
			expectedErr: "get \"" + server.URL + "/non-existent.yaml\": 404 Not Found",
		},
		{
			name: "with invalid url references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: "invalid:///url"},
					},
				},
			},
			expectedErr: "failed to resolve \"invalid:///url\": unsupported scheme: \"invalid\" in \"invalid:///url\"",
		},
		{
			name: "with nested non_existent_references",
			wf: v0.Workflow{
				Tasks: v0.TaskMap{
					"default": {
						v0.Step{Uses: server.URL + "/nested404.yaml"},
					},
				},
			},
			expectedErr: "get \"" + server.URL + "/dne.yaml\": 404 Not Found",
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

func TestExecuteUses(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	workflowFoo := v0.Workflow{SchemaVersion: v0.SchemaVersion, Tasks: v0.TaskMap{"default": {v0.Step{Run: "echo 'foo'"}, v0.Step{Uses: "file:bar/baz.yaml?task=baz"}}}}
	workflowBaz := v0.Workflow{SchemaVersion: v0.SchemaVersion, Tasks: v0.TaskMap{"baz": {v0.Step{Run: "echo 'baz'"}, v0.Step{Uses: "file:../hello-world.yaml"}}}}

	handleWF := func(w http.ResponseWriter, wf v0.Workflow) {
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
			handleWF(w, v0.Workflow{
				SchemaVersion: v0.SchemaVersion,
				Tasks: v0.TaskMap{
					"default": {v0.Step{Run: "echo 'Hello World!'"}},
					"a-task":  {v0.Step{Run: "echo 'task a'"}},
					"task-b":  {v0.Step{Run: "echo 'task b'"}},
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
		aliases     v0.AliasMap
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
			aliases: v0.AliasMap{
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
				_, err := handleUsesStep(ctx, svc, v0.Step{Uses: tt.uses}, v0.Workflow{Aliases: tt.aliases}, v0.With{}, nil, origin, false)
				require.NoError(t, err)
			} else {
				_, err := handleUsesStep(ctx, svc, v0.Step{Uses: tt.uses}, v0.Workflow{Aliases: tt.aliases}, v0.With{}, nil, origin, false)
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}
