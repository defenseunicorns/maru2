// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"maps"
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
)

// ExecuteUses executes a task from a given URI.
func ExecuteUses(ctx context.Context, svc *uses.FetcherService, pkgAliases map[string]config.Alias, u string, with With, prev *url.URL, dry bool) (map[string]any, error) {
	aliases := svc.PkgAliases()
	maps.Copy(aliases, pkgAliases)

	next, err := uses.ResolveRelative(prev, u, aliases)
	if err != nil {
		return nil, err
	}

	wf, err := Fetch(ctx, svc, next)
	if err != nil {
		return nil, err
	}

	taskName := next.Query().Get(uses.QualifierTask)

	return Run(ctx, svc, wf, taskName, with, next, dry)
}

func Fetch(ctx context.Context, svc *uses.FetcherService, uri *url.URL) (Workflow, error) {
	logger := log.FromContext(ctx)

	fetcher, err := svc.GetFetcher(uri)
	if err != nil {
		return Workflow{}, err
	}

	logger.Debug("fetching", "url", uri, "fetcher", fmt.Sprintf("%T", fetcher))

	rc, err := fetcher.Fetch(ctx, uri)
	if err != nil {
		return Workflow{}, err
	}
	defer rc.Close()

	return ReadAndValidate(rc)
}
