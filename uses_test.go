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
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestFetchAll(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: time.Second}))
	require.NoError(t, err)

	workflowNoRefs := Workflow{
		Tasks: TaskMap{
			"default": {Step{Run: "echo 'hello'"}},
		},
	}

	workflowWithRefs := Workflow{
		Tasks: TaskMap{
			"default": {
				Step{Run: "echo 'start'"},
				Step{Uses: "file:testdata/hello-world.yaml"},
				Step{Uses: "file:testdata/hello-world.yaml?task=another-task"},
			},
		},
	}

	workflowWithDuplicates := Workflow{
		Tasks: TaskMap{
			"default": {
				Step{Uses: "file:testdata/hello-world.yaml"},
				Step{Uses: "file:testdata/hello-world.yaml"},
			},
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workflow1.yaml":
			b, _ := yaml.Marshal(workflowNoRefs)
			_, _ = w.Write(b)

		case "/workflow2.yaml":
			// Create a workflow that references another workflow on the same server
			wf := Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Run: "echo 'nested start'"},
						Step{Uses: "file:workflow3.yaml"},
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

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	tests := []struct {
		name        string
		wf          Workflow
		expectedErr string
	}{
		{
			name: "no references",
			wf:   workflowNoRefs,
		},
		{
			name: "with references",
			wf:   workflowWithRefs,
		},
		{
			name: "with duplicate references",
			wf:   workflowWithDuplicates,
		},
		{
			name: "with remote references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: server.URL + "/workflow1.yaml"},
					},
				},
			},
		},
		{
			name: "with nested remote references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: server.URL + "/workflow2.yaml"},
					},
				},
			},
		},
		{
			name: "with_invalid_remote_references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: server.URL + "/invalid.yaml"},
					},
				},
			},
			expectedErr: "[1:1] string was used where mapping is expected\n>  1 | not a valid workflow yaml\n       ^\n",
		},
		{
			name: "with_server_error_references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: server.URL + "/error.yaml"},
					},
				},
			},
			expectedErr: "get \"" + server.URL + "/error.yaml\": 500 Internal Server Error",
		},
		{
			name: "with_invalid_url_references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: "invalid:///url"},
					},
				},
			},
			expectedErr: "failed to resolve \"invalid:///url\": unsupported scheme: \"invalid\" in \"invalid:///url\"",
		},
		{
			name: "with_non_existent_references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: server.URL + "/non-existent.yaml"},
					},
				},
			},
			expectedErr: "get \"" + server.URL + "/non-existent.yaml\": 404 Not Found",
		},
		{
			name: "with invalid url references",
			wf: Workflow{
				Tasks: TaskMap{
					"default": {
						Step{Uses: "invalid:///url"},
					},
				},
			},
			expectedErr: "failed to resolve \"invalid:///url\": unsupported scheme: \"invalid\" in \"invalid:///url\"",
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

	workflowFoo := Workflow{Tasks: TaskMap{"default": {Step{Run: "echo 'foo'"}, Step{Uses: "file:bar/baz.yaml?task=baz"}}}}
	workflowBaz := Workflow{Tasks: TaskMap{"baz": {Step{Run: "echo 'baz'"}, Step{Uses: "file:../hello-world.yaml"}}}}

	handleWF := func(w http.ResponseWriter, wf Workflow) {
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
			handleWF(w, helloWorldWorkflow)
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
		aliases     map[string]config.Alias
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
			aliases: map[string]config.Alias{
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
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			if tt.skipShort && testing.Short() {
				t.Skip("skipping test in short mode")
			}

			origin, err := url.Parse(tt.origin)
			require.NoError(t, err)

			if tt.expectedErr == "" {
				_, err := handleUsesStep(ctx, svc, Step{Uses: tt.uses}, Workflow{Aliases: tt.aliases}, With{}, nil, origin, false)
				require.NoError(t, err)
			} else {
				_, err := handleUsesStep(ctx, svc, Step{Uses: tt.uses}, Workflow{Aliases: tt.aliases}, With{}, nil, origin, false)
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}
