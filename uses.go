// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/uses"
)

func handleUsesStep(ctx context.Context, svc *uses.FetcherService, step Step, wf Workflow, withDefaults With,
	outputs CommandOutputs, origin *url.URL, dry bool) (map[string]any, error) {

	ctx = WithCWDContext(ctx, filepath.Join(CWDFromContext(ctx), step.Dir))

	if strings.HasPrefix(step.Uses, "builtin:") {
		return ExecuteBuiltin(ctx, step, withDefaults, outputs, dry)
	}

	templatedWith, err := TemplateWith(ctx, withDefaults, step.With, outputs, dry)
	if err != nil {
		return nil, err
	}

	if _, ok := wf.Tasks.Find(step.Uses); ok {
		return Run(ctx, svc, wf, step.Uses, templatedWith, origin, dry)
	}

	aliases := svc.PkgAliases()
	maps.Copy(aliases, wf.Aliases)

	next, err := uses.ResolveRelative(origin, step.Uses, aliases)
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
func Fetch(ctx context.Context, svc *uses.FetcherService, uri *url.URL) (Workflow, error) {
	logger := log.FromContext(ctx)

	fetcher, err := svc.GetFetcher(uri)
	if err != nil {
		return Workflow{}, err
	}

	fetcherType := fmt.Sprintf("%T", fetcher)
	if sf, ok := fetcher.(*uses.StoreFetcher); ok {
		fetcherType = fmt.Sprintf("%T|%T", sf.Store, sf.Source)
	}

	logger.Debug("fetching", "url", uri, "fetcher", fetcherType)

	rc, err := fetcher.Fetch(ctx, uri)
	if err != nil {
		return Workflow{}, err
	}
	defer rc.Close()

	return ReadAndValidate(rc)
}

// FetchAll fetches all workflows from a given URL.
func FetchAll(ctx context.Context, svc *uses.FetcherService, wf Workflow, src *url.URL) error {
	refs := []string{}

	for _, task := range wf.Tasks {
		for _, step := range task {
			if step.Uses != "" {
				refs = append(refs, step.Uses)
			}
		}
	}

	aliases := svc.PkgAliases()
	maps.Copy(aliases, wf.Aliases)

	for _, ref := range refs {
		resolved, err := uses.ResolveRelative(src, ref, aliases)
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
