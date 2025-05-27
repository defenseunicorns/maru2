// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestExecuteUses(t *testing.T) {
	svc, err := uses.NewFetcherService(uses.WithClient(&http.Client{Timeout: 5 * time.Second}))
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
			time.Sleep(10 * time.Second)
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

	with := With{}
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
			expectedErr: `must contain a scheme: "./path-with-no-scheme"`,
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
			expectedErr: `unsupported scheme: "ssh"`,
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
			expectedErr: "failed to fetch " + server.URL + "/non-existent.yaml: 404 Not Found",
		},
		{
			name:        "timeout",
			uses:        server.URL + "/timeout.yaml",
			origin:      dummyOrigin,
			expectedErr: fmt.Sprintf("Get \"%s/timeout.yaml\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)", server.URL),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			if tt.skipShort && testing.Short() {
				t.Skip("skipping test in short mode")
			}

			if tt.expectedErr == "" {
				_, err := ExecuteUses(ctx, svc, tt.aliases, tt.uses, with, tt.origin, false)
				require.NoError(t, err)
			} else {
				_, err := ExecuteUses(ctx, svc, tt.aliases, tt.uses, with, tt.origin, false)
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}
