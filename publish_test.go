// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/olareg/olareg"
	olaregcfg "github.com/olareg/olareg/config"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"

	"net/http"
)

func TestPublish(t *testing.T) {
	// not testing context cancellation at this time
	ctx := log.WithContext(t.Context(), log.New(io.Discard))

	remoteWorkflowContent := `tasks:
  remote:
    - run: "echo 'remote'"
`
	remoteHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/remote-dep.yaml":
			_, _ = w.Write([]byte(remoteWorkflowContent))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		}
	})
	remoteServer := httptest.NewServer(remoteHandler)
	t.Cleanup(remoteServer.Close)
	remoteDesc := content.NewDescriptorFromBytes(MediaTypeWorkflow, []byte(remoteWorkflowContent))
	remoteDesc.Annotations = map[string]string{ocispec.AnnotationTitle: remoteServer.URL + "/remote-dep.yaml"}
	usesRemoteContent := fmt.Sprintf(`tasks:
  main:
    - uses: "%s/remote-dep.yaml?task=remote"
`, remoteServer.URL)
	usesRemoteDesc := content.NewDescriptorFromBytes(MediaTypeWorkflow, []byte(usesRemoteContent))
	usesRemoteDesc.Annotations = map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"}

	tt := []struct {
		name           string
		workflow       string
		files          map[string]string
		entrypoints    []string
		expectedLayers []ocispec.Descriptor
		expectErr      string
	}{
		{
			name:        "simple workflow",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  noop:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:76b5a65b41aab5e570aae6af57e61748954334c587e578cb7eaa5a808265c82f",
					Size:        33,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
			},
		},
		{
			name:        "with local dependency",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  main:
    - uses: "file:dep.yaml?task=dep"
`,
				"dep.yaml": `tasks:
  dep:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:083dd91056ea12399edb42f99905d563fb55e7b1f4b3672b72efcda67582b660",
					Size:        52,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:ce250b935a88555f72f9e4499353ff8173ab4dc0f476b46d51c566f1906c4a61",
					Size:        32,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep.yaml"},
				},
			},
		},
		{
			name:        "with nested local dependency",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  main:
    - uses: "file:dep1.yaml?task=dep1"
`,
				"dep1.yaml": `tasks:
  dep1:
    - uses: "file:dep2.yaml?task=dep2"
`,
				"dep2.yaml": `tasks:
  dep2:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:8c98231524bd8db5fd647c6d282e9f42956f72b71a571a5eefe6bf27852dc980",
					Size:        54,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:26945f5cee5e3f2ebfdbc4b820bd6ce7abca6a25dd534b516d914f6545ca34a2",
					Size:        54,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep1.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:8ff065cf16ba56474165bc2033a2fce530309b6a8a816d1a6f5f14d9c232c278",
					Size:        33,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep2.yaml"},
				},
			},
		},
		{
			name:        "with directory dependency",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  main:
    - uses: "file:./nested/tasks.yaml?task=dep"
`,
				"nested/tasks.yaml": `tasks:
  dep:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:741938f3090969c83104f288b332cd41d5424e6c5ce8d77200e97eb74299b857",
					Size:        63,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:ce250b935a88555f72f9e4499353ff8173ab4dc0f476b46d51c566f1906c4a61",
					Size:        32,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:./nested/tasks.yaml"},
				},
			},
		},
		{
			name:        "non-existent entrypoint",
			entrypoints: []string{"non-existent.yaml"},
			files:       map[string]string{},
			expectErr:   "no such file or directory",
		},
		{
			name:        "non-existent local dependency",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  main:
    - uses: "file:non-existent.yaml?task=dep"
`,
			},
			expectErr: "no such file or directory",
		},
		{
			name:        "no entrypoints",
			entrypoints: []string{},
			files:       map[string]string{},
			expectErr:   "need at least one entrypoint",
		},
		{
			name:        "invalid entrypoint path",
			entrypoints: []string{"::invalid.yaml"},
			files:       map[string]string{},
			expectErr:   "missing protocol scheme",
		},
		{
			name:        "entrypoint with query params",
			entrypoints: []string{"tasks.yaml?task=main"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  main:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:a6c1eda52d254444e70b6be557e0e5e97726cad9c368b4b48622f8ca6006e2c4",
					Size:        33,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
			},
		},
		{
			name:        "multiple entrypoints",
			entrypoints: []string{"tasks.yaml", "dep.yaml"},
			files: map[string]string{
				"tasks.yaml": `tasks:
  main:
    - run: "true"
`,
				"dep.yaml": `tasks:
  dep:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:a6c1eda52d254444e70b6be557e0e5e97726cad9c368b4b48622f8ca6006e2c4",
					Size:        33,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:ce250b935a88555f72f9e4499353ff8173ab4dc0f476b46d51c566f1906c4a61",
					Size:        32,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep.yaml"},
				},
			},
		},
		{
			name:        "with remote dependency",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": usesRemoteContent,
			},
			expectedLayers: []ocispec.Descriptor{
				usesRemoteDesc,
				remoteDesc,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := olareg.New(olaregcfg.Config{
				Storage: olaregcfg.ConfigStorage{
					StoreType: olaregcfg.StoreMem,
				},
			})
			s := httptest.NewServer(r)
			t.Cleanup(func() {
				s.Close()
				_ = r.Close()
			})

			tmpDir := t.TempDir()
			for path, content := range tc.files {
				fullPath := filepath.Join(tmpDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}
			t.Chdir(tmpDir)

			serverURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			ref := fmt.Sprintf("%s/test-repo:latest", serverURL.Host)

			dst, err := remote.NewRepository(ref)
			require.NoError(t, err)
			dst.PlainHTTP = true

			// TODO: test w/ aliases?
			err = Publish(ctx, dst, tc.entrypoints, nil)

			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			require.NoError(t, err)

			manifestDesc, manifest, err := fetchManifest(t, dst)
			require.NoError(t, err)

			assert.Equal(t, MediaTypeWorkflow, manifest.ArtifactType)
			assert.Equal(t, ocispec.MediaTypeImageManifest, manifestDesc.MediaType)
			assert.Equal(t, ocispec.MediaTypeImageManifest, manifest.MediaType)
			assert.Equal(t, ocispec.DescriptorEmptyJSON, manifest.Config)

			assert.ElementsMatch(t, tc.expectedLayers, manifest.Layers)
		})
	}
}

func fetchManifest(t *testing.T, repo *remote.Repository) (desc ocispec.Descriptor, manifest ocispec.Manifest, err error) {
	t.Helper()

	desc, rc, err := repo.FetchReference(t.Context(), repo.Reference.String())
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Manifest{}, err
	}
	defer rc.Close()

	var manifestObj ocispec.Manifest
	b, err := io.ReadAll(rc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Manifest{}, err
	}
	if err := json.Unmarshal(b, &manifestObj); err != nil {
		return ocispec.Descriptor{}, ocispec.Manifest{}, err
	}
	return desc, manifestObj, nil
}
