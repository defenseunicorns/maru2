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

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	"github.com/defenseunicorns/maru2/uses"
)

func handleUsesStep(ctx context.Context, svc *uses.FetcherService, step v0.Step, wf v0.Workflow, withDefaults v0.With,
	outputs CommandOutputs, origin *url.URL, dry bool) (map[string]any, error) {

	ctx = WithCWDContext(ctx, filepath.Join(CWDFromContext(ctx), step.Dir))

	if strings.HasPrefix(step.Uses, "builtin:") {
		return ExecuteBuiltin(ctx, step, withDefaults, outputs, dry)
	}

	logger := log.FromContext(ctx)

	logger.Debug("templating", "input", withDefaults, "local", step.With)

	templatedWith, err := TemplateWithMap(ctx, withDefaults, outputs, step.With, dry)
	if err != nil {
		return nil, err
	}

	logger.Debug("templated", "result", templatedWith)

	if _, ok := wf.Tasks.Find(step.Uses); ok {
		return Run(ctx, svc, wf, step.Uses, templatedWith, origin, dry)
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

	return Run(ctx, svc, nextWf, taskName, templatedWith, next, dry)
}

// Fetch fetches a workflow from a given URL.
func Fetch(ctx context.Context, svc *uses.FetcherService, uri *url.URL) (v0.Workflow, error) {
	logger := log.FromContext(ctx)

	fetcher, err := svc.GetFetcher(uri)
	if err != nil {
		return v0.Workflow{}, err
	}

	fetcherType := fmt.Sprintf("%T", fetcher)
	if sf, ok := fetcher.(*uses.StoreFetcher); ok {
		fetcherType = fmt.Sprintf("%T|%T", sf.Store, sf.Source)
	}

	logger.Debug("fetching", "url", uri, "fetcher", fetcherType)

	rc, err := fetcher.Fetch(ctx, uri)
	if err != nil {
		return v0.Workflow{}, err
	}
	defer rc.Close()

	return v0.ReadAndValidate(rc)
}

// FetchAll fetches all workflows from a given URL.
func FetchAll(ctx context.Context, svc *uses.FetcherService, wf v0.Workflow, src *url.URL) error {
	refs := []string{}

	for _, task := range wf.Tasks {
		for _, step := range task {
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

// ListAllLocal recursively discovers all local references contained in a workflow
func ListAllLocal(ctx context.Context, src *url.URL, fs afero.Fs) ([]string, error) {
	if src.Scheme != "file" {
		return nil, nil
	}

	relativeRefs := []string{}

	rc, err := uses.NewLocalFetcher(fs).Fetch(ctx, src)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	wf, err := v0.ReadAndValidate(rc)
	if err != nil {
		return nil, err
	}

	for _, task := range wf.Tasks {
		for _, step := range task {
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

		rc, err := uses.NewLocalFetcher(fs).Fetch(ctx, resolved)
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		_, err = v0.ReadAndValidate(rc)
		if err != nil {
			return nil, err
		}

		// now we know its a valid workflow, we can save the location
		fullRefs = append(fullRefs, resolved.String())

		sub, err := ListAllLocal(ctx, resolved, fs)
		if err != nil {
			return nil, err
		}
		fullRefs = append(fullRefs, sub...)
	}

	return slices.Compact(fullRefs), nil
}
