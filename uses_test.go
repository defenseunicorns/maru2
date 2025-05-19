// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestExecuteUses(t *testing.T) {
	svc, err := uses.NewFetcherService()
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
		resolver    uses.AliasResolver
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
			resolver: uses.MapBasedResolver(map[string]config.Alias{
				"custom": {
					Type: "gitlab",
				},
			}),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			if tt.skipShort { // && testing.Short() {
				t.Skip("skipping test in short mode")
			}

			if tt.expectedErr == "" {
				_, err := ExecuteUses(ctx, svc, tt.resolver, tt.uses, with, tt.origin, false)
				require.NoError(t, err)
			} else {
				_, err := ExecuteUses(ctx, svc, tt.resolver, tt.uses, with, tt.origin, false)
				require.EqualError(t, err, tt.expectedErr)
			}
		})
	}
}
