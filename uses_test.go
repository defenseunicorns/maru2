// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestExecuteUses(t *testing.T) {
	ctx := log.WithContext(t.Context(), log.New(io.Discard))
	
	// Create test fetcher service
	svc, err := uses.NewFetcherService(nil, nil)
	require.NoError(t, err)

	workflowFoo := Workflow{Tasks: TaskMap{"default": {Step{Run: "echo 'foo'"}, Step{Uses: "file:bar/baz.yaml?task=baz"}}}}
	workflowBaz := Workflow{Tasks: TaskMap{"baz": {Step{Run: "echo 'baz'"}, Step{Uses: "file:../hello-world.yaml"}}}}

	handleWF := func(w http.ResponseWriter, wf Workflow) {
		b, err := yaml.Marshal(wf.Tasks)
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
		// handle /hello-world.yaml
		if r.URL.Path == "/hello-world.yaml" {
			handleWF(w, helloWorldWorkflow)
			return
		}

		// handle /foo.yaml
		if r.URL.Path == "/foo.yaml" {
			handleWF(w, workflowFoo)
			return
		}

		// handle /bar/baz.yaml
		if r.URL.Path == "/bar/baz.yaml" {
			handleWF(w, workflowBaz)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// run default task because no ?task=
	helloWorld := server.URL + "/hello-world.yaml"
	with := With{}

	_, err = ExecuteUses(ctx, "file:testdata/hello-world.yaml", with, "file:test", false, svc)
	require.NoError(t, err)

	_, err = ExecuteUses(ctx, "file:testdata/hello-world.yaml?task=a-task", with, "file:test", false, svc)
	require.NoError(t, err)

	_, err = ExecuteUses(ctx, helloWorld, with, "file:test", false, svc)
	require.NoError(t, err)

	_, err = ExecuteUses(ctx, "./path-with-no-scheme", with, "file:test", false, svc)
	require.EqualError(t, err, `must contain a scheme: "./path-with-no-scheme"`)

	_, err = ExecuteUses(ctx, "file:test", with, "./missing-scheme", false, svc)
	require.EqualError(t, err, `must contain a scheme: "./missing-scheme"`)

	_, err = ExecuteUses(ctx, "http://www.example.com/\x7f", with, "file:test", false, svc)
	require.EqualError(t, err, `parse "http://www.example.com/\x7f": net/url: invalid control character in URL`)

	_, err = ExecuteUses(ctx, "file:test", with, "http://www.example.com/\x7f", false, svc)
	require.EqualError(t, err, `parse "http://www.example.com/\x7f": net/url: invalid control character in URL`)

	_, err = ExecuteUses(ctx, "ssh:not-supported", with, "file:test", false, svc)
	require.EqualError(t, err, `unsupported scheme: "ssh"`)

	_, err = ExecuteUses(ctx, "pkg:bitbucket/owner/repo", with, "file:test", false, svc)
	require.EqualError(t, err, `unsupported type: "bitbucket"`)

	_, err = ExecuteUses(ctx, "file:..?task=hello-world", with, "pkg:", false, svc)
	require.EqualError(t, err, `purl is missing type or name`)

	// TODO: restore this test
	// if !testing.Short() {
	// 	err = ExecuteUses(ctx, "file:..?task=hello-world", with, "pkg:github/defenseunicorns/maru2#testdata/hello-world.yaml", false)
	// 	require.NoError(t, err)
	// }

	// lets get crazy w/ it
	// foo.yaml uses baz.yaml which uses hello-world.yaml
	_, err = ExecuteUses(ctx, server.URL+"/foo.yaml", with, "file:test", false, svc)
	require.NoError(t, err)
}
