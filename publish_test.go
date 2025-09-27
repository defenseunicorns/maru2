// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
)

func TestPublish(t *testing.T) {
	remoteWorkflowContent := `
schema-version: v0
tasks:
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
	usesRemoteContent := fmt.Sprintf(`
schema-version: v0
tasks:
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
				"tasks.yaml": `
schema-version: v0
tasks:
  noop:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:bab034b4352bf26f8543ff6499a56210a0cd9acdac02c8cb545f678a58d18a34",
					Size:        53,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
			},
		},
		{
			name:        "with local dependency",
			entrypoints: []string{"tasks.yaml"},
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
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:ebd11b8920091e2a6e2f2050ee18d456bc8041a8601cf131a84507f6d1ad3b5a",
					Size:        72,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:cf8bcd8f445d8611ba14b04f283ba9c4e1fa18a04635b30cf19d048abb60614d",
					Size:        52,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep.yaml"},
				},
			},
		},
		{
			name:        "with nested local dependency",
			entrypoints: []string{"tasks.yaml"},
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
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:cfaa905058cee7a842b6a829db1098b5649b27fdc94192234ee8a88b00d84e3a",
					Size:        74,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:b4de33822540858d402dab6e7e46bc3988cf0bea060d8781b24d0cef3ac5b371",
					Size:        74,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep1.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:066e03e70397ce63a111d086f09a584a6b8ac707c8cbe9ce68680d4aba185820",
					Size:        53,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:dep2.yaml"},
				},
			},
		},
		{
			name:        "with directory dependency",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "file:./nested/tasks.yaml?task=dep"
`,
				"nested/tasks.yaml": `
schema-version: v0
tasks:
  dep:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:578d9a9ce72c8b11141df11deb355505ca0fac55b8b499c918783be309ae480d",
					Size:        83,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:cf8bcd8f445d8611ba14b04f283ba9c4e1fa18a04635b30cf19d048abb60614d",
					Size:        52,
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
				"tasks.yaml": `
schema-version: v0
tasks:
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
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:97e2c0262ec9cc6c5afb8b5c1298f475f1d2422e09db3ce5b511df2b23c49f0e",
					Size:        53,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
			},
		},
		{
			name:        "multiple entrypoints",
			entrypoints: []string{"tasks.yaml", "dep.yaml"},
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - run: "true"
`,
				"dep.yaml": `
schema-version: v0
tasks:
  dep:
    - run: "true"
`,
			},
			expectedLayers: []ocispec.Descriptor{
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:97e2c0262ec9cc6c5afb8b5c1298f475f1d2422e09db3ce5b511df2b23c49f0e",
					Size:        53,
					Annotations: map[string]string{ocispec.AnnotationTitle: "file:tasks.yaml"},
				},
				{
					MediaType:   MediaTypeWorkflow,
					Digest:      "sha256:cf8bcd8f445d8611ba14b04f283ba9c4e1fa18a04635b30cf19d048abb60614d",
					Size:        52,
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
		{
			name:        "entrypoint with invalid workflow syntax",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": "invalid: yaml: syntax",
			},
			expectErr: "mapping value is not allowed in this context",
		},
		{
			name:        "entrypoint with local dependency that has invalid syntax",
			entrypoints: []string{"tasks.yaml"},
			files: map[string]string{
				"tasks.yaml": `
schema-version: v0
tasks:
  main:
    - uses: "file:invalid.yaml?task=task"
`,
				"invalid.yaml": "not: valid: workflow: syntax",
			},
			expectErr: "mapping value is not allowed in this context",
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
				err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
			}
			t.Chdir(tmpDir)

			serverURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			ref := fmt.Sprintf("%s/test-repo:latest", serverURL.Host)

			dst, err := remote.NewRepository(ref)
			require.NoError(t, err)
			dst.PlainHTTP = true

			// not testing context cancellation at this time
			ctx := log.WithContext(t.Context(), log.New(io.Discard))
			err = Publish(ctx, dst, tc.entrypoints)

			if tc.expectErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErr)
				return
			}
			require.NoError(t, err)

			manifestDesc, manifest, err := fetchManifest(t, dst)
			require.NoError(t, err)

			assert.Equal(t, MediaTypeWorkflowCollection, manifest.ArtifactType)
			assert.Equal(t, ocispec.MediaTypeImageManifest, manifestDesc.MediaType)
			assert.Equal(t, ocispec.MediaTypeImageManifest, manifest.MediaType)
			assert.Equal(t, ocispec.DescriptorEmptyJSON, manifest.Config)

			assert.ElementsMatch(t, tc.expectedLayers, manifest.Layers)
		})
	}

	t.Run("mkdirtemp fails", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("TMPDIR", filepath.Join(tmp, "dir", "dne"))
		ctx := log.WithContext(t.Context(), log.New(io.Discard))
		err := Publish(ctx, nil, []string{"tasks.yaml"})
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("cwd fails", func(t *testing.T) {
		tmp := t.TempDir()
		sub := filepath.Join(tmp, "dir")
		require.NoError(t, os.Mkdir(sub, 0o755))
		t.Chdir(sub)
		require.NoError(t, os.Remove(sub))
		ctx := log.WithContext(t.Context(), log.New(io.Discard))
		err := Publish(ctx, nil, []string{"tasks.yaml"})
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("context is pre-cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := Publish(ctx, nil, []string{"tasks.yaml"})
		require.ErrorIs(t, err, context.Canceled)
	})
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
