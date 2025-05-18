// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/uses"
)

// ExecuteUses executes a task from a given URI.
func ExecuteUses(ctx context.Context, svc *uses.FetcherService, u string, with With, prev string, dry bool) (map[string]any, error) {
	logger := log.FromContext(ctx)
	logger.Debug("using", "task", u)

	next, err := uses.ResolveURL(u, prev)
	if err != nil {
		return nil, err
	}

	nextURI, err := url.Parse(next)
	if err != nil {
		return nil, err
	}

	prevURI, err := url.Parse(prev)
	if err != nil {
		return nil, err
	}

	logger.Debug("resolved", "next", next)

	fetcher, err := svc.GetFetcher(nextURI, prevURI)
	if err != nil {
		return nil, err
	}

	logger.Debug("chosen", "fetcher", fmt.Sprintf("%T", fetcher))

	rc, err := fetcher.Fetch(ctx, next)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	wf, err := ReadAndValidate(rc)
	if err != nil {
		return nil, err
	}

	taskName := nextURI.Query().Get("task")

	return Run(ctx, svc, wf, taskName, with, next, dry)
}
