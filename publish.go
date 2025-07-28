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

	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
)

// MediaTypeWorkflow is the mediatype for all maru2 workflows
const MediaTypeWorkflow = "application/vnd.maru2.workflow.v1+yaml"

// Publish fetches all remote imports in <cwd>/tasks.yaml, stores them in a temp dir, then pushes them to a OCI registry
func Publish(ctx context.Context, cfg *config.Config, dst *remote.Repository, entrypoints []string) error {
	logger := log.FromContext(ctx)

	if len(entrypoints) == 0 {
		return fmt.Errorf("need at least one entrypoint")
	}

	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	store, err := uses.NewLocalStore(afero.NewBasePathFs(afero.NewOsFs(), tmp))
	if err != nil {
		return err
	}

	svc, err := uses.NewFetcherService(
		uses.WithStorage(store),
		uses.WithFetchPolicy(config.FetchPolicyAlways),
	)
	if err != nil {
		return err
	}

	localPaths := []string{}

	fs := afero.NewOsFs()
	for _, point := range entrypoints {
		src, err := uses.ResolveRelative(nil, point, cfg.Aliases)
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

	cwd, err := os.Getwd()
	if err != nil {
		return err
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

	root, err := oras.PackManifest(ctx, ociStore, oras.PackManifestVersion1_1, MediaTypeWorkflow, oras.PackManifestOptions{
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
