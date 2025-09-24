// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

// handleUsesStep executes remote workflow imports
//
// Fetches, validates, and executes tasks from remote sources (GitHub, GitLab, OCI, HTTP) or local file paths
// using package URL resolution and alias expansion
func handleUsesStep(
	ctx context.Context,
	svc *uses.FetcherService,
	step v1.Step,
	wf v1.Workflow,
	withDefaults schema.With,
	outputs CommandOutputs,
	origin *url.URL,
	ro RuntimeOptions,
) (map[string]any, error) {
	ro.WorkingDir = filepath.Join(ro.WorkingDir, step.Dir)

	if strings.HasPrefix(step.Uses, "builtin:") {
		return ExecuteBuiltin(ctx, step, withDefaults, outputs, ro.Dry)
	}

	logger := log.FromContext(ctx)

	logger.Debug("templating", "input", withDefaults, "local", step.With)

	templatedWith, err := TemplateWithMap(ctx, step.With, withDefaults, outputs, ro.Dry)
	if err != nil {
		return nil, err
	}

	logger.Debug("templated", "result", templatedWith)

	templatedEnv, err := TemplateWithMap(ctx, step.Env, withDefaults, outputs, ro.Dry)
	if err != nil {
		return nil, err
	}

	env, err := prepareEnvironment(ro.Env, nil, "", templatedEnv)
	if err != nil {
		return nil, err
	}
	ro.Env = env

	if _, ok := wf.Tasks.Find(step.Uses); ok {
		return Run(ctx, svc, wf, step.Uses, templatedWith, origin, ro)
	}

	next, err := uses.ResolveRelative(origin, step.Uses, wf.Aliases)
	if err != nil {
		return nil, err
	}

	nextWf, err := Fetch(ctx, svc, next)
	if err != nil {
		return nil, err
	}

	taskName := next.Query().Get(uses.QualifierTask)

	return Run(ctx, svc, nextWf, taskName, templatedWith, next, ro)
}

// Fetch downloads and validates a workflow from a remote or local source
//
// Supports multiple fetcher types (GitHub, GitLab, OCI, HTTP, local files) with
// automatic fetcher selection based on URL scheme and configuration
func Fetch(ctx context.Context, svc *uses.FetcherService, uri *url.URL) (v1.Workflow, error) {
	logger := log.FromContext(ctx)

	fetcher, err := svc.GetFetcher(uri)
	if err != nil {
		return v1.Workflow{}, err
	}

	fetcherType := fmt.Sprintf("%T", fetcher)
	if sf, ok := fetcher.(*uses.StoreFetcher); ok {
		fetcherType = fmt.Sprintf("%T|%T", sf.Store, sf.Source)
	}

	logger.Debug("fetching", "url", uri, "fetcher", fetcherType)

	rc, err := fetcher.Fetch(ctx, uri)
	if err != nil {
		return v1.Workflow{}, err
	}
	defer rc.Close()

	return v1.ReadAndValidate(rc)
}

// FetchAll recursively downloads all remote workflow dependencies
//
// Scans the workflow for uses: references, resolves URLs relative to the source,
// and pre-fetches all dependencies into the cache for offline execution
func FetchAll(ctx context.Context, svc *uses.FetcherService, wf v1.Workflow, src *url.URL) error {
	refs := []string{}

	for _, task := range wf.Tasks {
		for _, step := range task.Steps {
			if step.Uses == "" {
				continue
			}
			_, found := wf.Tasks.Find(step.Uses)
			if found {
				continue
			}

			if strings.HasPrefix(step.Uses, "builtin:") {
				continue
			}

			if slices.Contains(refs, step.Uses) { // could use a map[string] here, would also need to dedup same import but different tasks
				continue
			}

			refs = append(refs, step.Uses)
		}
	}

	for _, ref := range refs {
		resolved, err := uses.ResolveRelative(src, ref, wf.Aliases)
		if err != nil {
			return fmt.Errorf("failed to resolve %q: %w", ref, err)
		}
		wf, err = Fetch(ctx, svc, resolved)
		if err != nil {
			return err
		}
		err = FetchAll(ctx, svc, wf, resolved)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListAllLocal recursively discovers all local file dependencies in a workflow tree
//
// Scans file:// workflows for local uses: references, validates them, and returns
// the complete list of local files needed for execution
func ListAllLocal(ctx context.Context, src *url.URL, fsys afero.Fs) ([]string, error) {
	if src.Scheme != "file" {
		return nil, nil
	}

	relativeRefs := []string{}

	rc, err := uses.NewLocalFetcher(fsys).Fetch(ctx, src)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	wf, err := v1.ReadAndValidate(rc)
	if err != nil {
		return nil, err
	}

	for _, task := range wf.Tasks {
		for _, step := range task.Steps {
			if step.Uses == "" {
				continue
			}
			uri, err := url.Parse(step.Uses)
			if err != nil {
				return nil, err
			}
			if uri.Scheme != "file" {
				continue
			}

			relativeRefs = append(relativeRefs, step.Uses)
		}
	}

	for _, alias := range wf.Aliases {
		if alias.Path != "" {
			relativeRefs = append(relativeRefs, fmt.Sprintf("file:%s", alias.Path))
		}
	}

	clone := *src
	clone.RawQuery = ""
	fullRefs := []string{clone.String()}

	for _, ref := range relativeRefs {
		resolved, err := uses.ResolveRelative(src, ref, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %q: %w", ref, err)
		}

		// strip query params, like ?task=
		resolved.RawQuery = ""

		rc, err := uses.NewLocalFetcher(fsys).Fetch(ctx, resolved)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		_, err = v1.ReadAndValidate(rc)
		if err != nil {
			return nil, err
		}

		// now we know its a valid workflow, we can save the location
		fullRefs = append(fullRefs, resolved.String())

		sub, err := ListAllLocal(ctx, resolved, fsys)
		if err != nil {
			return nil, err
		}
		fullRefs = append(fullRefs, sub...)
	}

	return slices.Compact(fullRefs), nil
}
