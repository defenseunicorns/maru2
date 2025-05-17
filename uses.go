// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/package-url/packageurl-go"
)

// ExecuteUses executes a task from a given URI.
func ExecuteUses(ctx context.Context, u string, with With, prev *url.URL, dry bool, svc *uses.FetcherService) (map[string]any, error) {
	logger := log.FromContext(ctx)
	logger.Debug("using", "task", u)

	uri, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "" {
		return nil, fmt.Errorf("must contain a scheme: %q", u)
	}

	if prev.Scheme == "" {
		return nil, fmt.Errorf("must contain a scheme: %q", prev)
	}

	var next *url.URL

	if uri.Scheme == "file" {
		switch prev.Scheme {
		case "http", "https":
			// turn relative paths into absolute references
			next = prev
			next.Path = filepath.Join(filepath.Dir(prev.Path), uri.Opaque)
			if next.Path == "." {
				next.Path = DefaultFileName
			}
		case "pkg":
			pURL, err := packageurl.FromString(prev.String())
			if err != nil {
				return nil, err
			}
			// turn relative paths into absolute references
			pURL.Subpath = filepath.Join(filepath.Dir(pURL.Subpath), uri.Opaque)
			if pURL.Subpath == "." {
				pURL.Subpath = DefaultFileName
			}
			next, _ = url.Parse(pURL.String())
		default:
			dir := filepath.Dir(prev.Opaque)
			if dir != "." {
				next = &url.URL{
					Scheme:   uri.Scheme,
					Opaque:   filepath.Join(dir, uri.Opaque),
					RawQuery: uri.RawQuery,
				}
				if next.Opaque == "." {
					next.Opaque = DefaultFileName
				}
			}
		}

		if next != nil {
			logger.Debug("merged", "previous", prev, "uses", u, "next", next)
			u = next.String()
		}
	}

	if next == nil {
		next, _ = url.Parse(u)
	}

	if uri.Scheme == "pkg" {
		// dogsledding the error here since we know it's a package URL
		pURL, _ := packageurl.FromString(u)
		if pURL.Subpath == "" {
			pURL.Subpath = DefaultFileName
		}
		if pURL.Version == "" {
			pURL.Version = "main"
		}
		u = pURL.String()
	}

	fetcher, err := svc.GetFetcher(uri, prev)
	if err != nil {
		return nil, err
	}

	logger.Debug("chosen", "fetcher", fmt.Sprintf("%T", fetcher))

	rc, err := fetcher.Fetch(ctx, u)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	wf, err := ReadAndValidate(rc)
	if err != nil {
		return nil, err
	}

	taskName := uri.Query().Get("task")

	return Run(ctx, wf, taskName, with, next, dry, svc)
}
