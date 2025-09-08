// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/rogpeppe/go-internal/testscript"

	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestFetchE2E(t *testing.T) {
	// Set up mock HTTP server for remote workflow fetching
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/simple.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"hello": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'Hello from remote!'"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		case "/with-uses.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"main": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'Starting main task'"},
							{Run: "echo 'Hello from remote!'"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		case "/nested.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"nested": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'Nested task'"},
							{Uses: "file:deeper.yaml"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		case "/deeper.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'Deep nested task'"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		case "/invalid.yaml":
			_, _ = w.Write([]byte("not a valid workflow yaml"))

		case "/error.yaml":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))

		case "/missing-dependency.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"default": v1.Task{
						Steps: []v1.Step{
							{Uses: "file:nonexistent.yaml"},
						},
					},
				},
			}
			b, _ := yaml.Marshal(wf)
			_, _ = w.Write(b)

		case "/workflow-that-uses-missing.yaml":
			wf := v1.Workflow{
				SchemaVersion: v1.SchemaVersion,
				Tasks: v1.TaskMap{
					"task": v1.Task{
						Steps: []v1.Step{
							{Run: "echo 'This workflow uses a missing dependency'"},
							{Uses: "file:definitely-does-not-exist.yaml"},
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

	httpServer := httptest.NewServer(httpHandler)
	t.Cleanup(httpServer.Close)

	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("..", "testdata", "fetch"),
		Setup: func(env *testscript.Env) error {
			env.Setenv("NO_COLOR", "true")
			env.Setenv("HTTP_BASE_URL", httpServer.URL)
			env.Setenv("HOME", filepath.Join(env.WorkDir, "home"))
			return nil
		},
		RequireUniqueNames: true,
		// UpdateScripts:      true,
	})
}
