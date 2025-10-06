// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	"github.com/charmbracelet/log"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/afero"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/defenseunicorns/maru2/uses"
)

// MediaTypeWorkflow is the mediatype for all maru2 workflows
const MediaTypeWorkflow = "application/vnd.maru2.workflow.v1+yaml"

// MediaTypeWorkflowCollection is the mediatype for the maru2 OCI collection artifact
const MediaTypeWorkflowCollection = "application/vnd.maru2.collection.v1"

// Publish packages workflows as OCI artifacts in a container registry
//
// Fetches all remote imports, stores them in a temp directory, then pushes
// the complete workflow bundle to the OCI registry for distribution
func Publish(ctx context.Context, dst *remote.Repository, entrypoints []string) error {
	logger := log.FromContext(ctx)

	if len(entrypoints) == 0 {
		return fmt.Errorf("need at least one entrypoint")
	}

	// using os.CreateTemp w/ an empty string as the first argument
	// leverages the TMPDIR environment variable, otherwise OS specific defaults
	// see `go doc os.TempDir`
	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	// leverages the PWD environment variable, otherwise OS specific defaults
	// see `go doc os.Getwd`
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	fs := afero.NewOsFs()

	store, err := uses.NewLocalStore(afero.NewBasePathFs(fs, tmp))
	if err != nil {
		return err
	}

	svc, err := uses.NewFetcherService(
		uses.WithStorage(store),
		uses.WithFetchPolicy(uses.FetchPolicyAlways),
	)
	if err != nil {
		return err
	}

	localPaths := []string{}

	for _, point := range entrypoints {
		src, err := uses.ResolveRelative(nil, point, nil)
		if err != nil {
			return err
		}

		wf, err := Fetch(ctx, svc, src)
		if err != nil {
			return err
		}

		if err := FetchAll(ctx, svc, wf, src); err != nil {
			return err
		}

		paths, err := ListAllLocal(ctx, src, fs)
		if err != nil {
			return err
		}
		localPaths = append(localPaths, paths...)
	}

	localPaths = slices.Compact(localPaths)

	if err := store.GC(); err != nil {
		return err
	}

	ociStore, err := file.New(tmp)
	if err != nil {
		return err
	}

	layers := []ocispec.Descriptor{}
	for name, storeDesc := range store.List() {
		logger.Debug("staging", "entry", name)

		desc, err := ociStore.Add(ctx, name, MediaTypeWorkflow, storeDesc.Hex)
		if err != nil {
			return err
		}
		layers = append(layers, desc)
	}

	for _, localPath := range localPaths {
		uri, err := url.Parse(localPath)
		if err != nil {
			return err
		}
		// replicates id() method on store and local fetcher
		// should dedupe logic
		uri.Scheme = ""
		uri.RawQuery = ""
		rel := uri.String()

		abs := filepath.Join(cwd, rel)

		logger.Debug("staging", "entry", rel)
		desc, err := ociStore.Add(ctx, localPath, MediaTypeWorkflow, abs)
		if err != nil {
			return err
		}
		layers = append(layers, desc)
	}

	root, err := oras.PackManifest(ctx, ociStore, oras.PackManifestVersion1_1, MediaTypeWorkflowCollection, oras.PackManifestOptions{
		Layers: layers,
	})
	if err != nil {
		return err
	}

	if err := ociStore.Tag(ctx, root, root.Digest.String()); err != nil {
		return err
	}

	desc, err := oras.Copy(ctx, ociStore, root.Digest.String(), dst, dst.Reference.Reference, oras.DefaultCopyOptions)
	if err != nil {
		return err
	}
	logger.Info("published", "digest", desc.Digest, "to", dst.Reference.Reference)

	return nil
}
