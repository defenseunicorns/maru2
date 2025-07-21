// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
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
	"oras.land/oras-go/v2/registry/remote"

	"github.com/defenseunicorns/maru2/config"
)

func TestPublish(t *testing.T) {
	// not testing context cancellation at this time
	ctx := log.WithContext(context.Background(), log.New(io.Discard))

	tt := []struct {
		name          string
		workflow      string
		files         map[string]string // map of filename to content
		entrypoints   []string
		expectedFiles []string
		expectErr     string
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
			expectedFiles: []string{"file:tasks.yaml"},
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
			expectedFiles: []string{"file:tasks.yaml", "file:dep.yaml"},
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
			expectedFiles: []string{"file:tasks.yaml", "file:dep1.yaml", "file:dep2.yaml"},
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
			expectedFiles: []string{"file:tasks.yaml", "file:./nested/tasks.yaml"},
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

			// setup test directory
			tmpDir := t.TempDir()
			for path, content := range tc.files {
				fullPath := filepath.Join(tmpDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
			}
			// change to test directory
			t.Chdir(tmpDir)

			// setup remote repository
			serverURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			ref := fmt.Sprintf("%s/test-repo:latest", serverURL.Host)

			dst, err := remote.NewRepository(ref)
			require.NoError(t, err)
			dst.PlainHTTP = true

			// publish the workflow
			err = Publish(ctx, &config.Config{}, dst, tc.entrypoints)

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

			var foundFiles []string
			for _, layer := range manifest.Layers {
				foundFiles = append(foundFiles, layer.Annotations[ocispec.AnnotationTitle])
			}

			assert.ElementsMatch(t, tc.expectedFiles, foundFiles)
		})
	}
}

// fetchManifest fetches the manifest descriptor and manifest object from a remote repository.
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
