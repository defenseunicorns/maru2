// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/package-url/packageurl-go"
)

// ResolveURL resolves a URI relative to a previous URI.
// It handles different schemes (file, http, https, pkg) and resolves relative paths.
func ResolveURL(p, u string) (string, error) {
	prev, err := url.Parse(p)
	if err != nil {
		return "", err
	}

	uri, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	if uri.Scheme == "" {
		return "", fmt.Errorf("must contain a scheme: %q", uri)
	}

	if uri.Opaque == "." {
		return "", fmt.Errorf("invalid relative path \".\"")
	}

	if prev.Scheme == "" {
		return "", fmt.Errorf("must contain a scheme: %q", prev)
	}

	// file -> http(s) or pkg
	if prev.Scheme == "file" && (uri.Scheme == "https" || uri.Scheme == "http" || uri.Scheme == "pkg") {
		return u, nil
	}

	// http(s) -> http(s)
	if (prev.Scheme == "https" || prev.Scheme == "http") && (uri.Scheme == "https" || uri.Scheme == "http") {
		return u, nil
	}

	// pkg -> pkg
	if prev.Scheme == "pkg" && uri.Scheme == "pkg" {
		return u, nil
	}

	// file -> file
	if prev.Scheme == "file" && uri.Scheme == "file" {
		dir := filepath.Dir(prev.Opaque)
		if dir != "." {
			next := &url.URL{
				Scheme:   "file",
				Opaque:   filepath.Join(dir, uri.Opaque),
				RawQuery: uri.RawQuery,
			}
			if next.Opaque == "." {
				next.Opaque = DefaultFileName
			}
			return next.String(), nil
		}
		return u, nil
	}

	// http(s) -> file (assumes relative path) = http(s) + relative path
	if (prev.Scheme == "https" || prev.Scheme == "http") && uri.Scheme == "file" {
		next := *prev // https://github.com/golang/go/issues/38351
		next.Path = filepath.Join(filepath.Dir(prev.Path), uri.Opaque)
		if next.Path == "." || next.Path == "/" {
			next.Path = "/" + DefaultFileName
		}
		return next.String(), nil
	}

	// pkg -> file (assumes relative path) = pkg + relative path
	if prev.Scheme == "pkg" && uri.Scheme == "file" {
		pURL, err := packageurl.FromString(p)
		if err != nil {
			return "", err
		}
		pURL.Subpath = filepath.Join(filepath.Dir(pURL.Subpath), uri.Opaque)
		if pURL.Subpath == "." {
			pURL.Subpath = DefaultFileName
		}
		if pURL.Version == "" {
			pURL.Version = "main"
		}

		if taskName := uri.Query().Get("task"); taskName != "" {
			qm := pURL.Qualifiers.Map()
			qm["task"] = taskName
			pURL.Qualifiers = packageurl.QualifiersFromMap(qm)
		}

		return pURL.String(), nil
	}

	return "", fmt.Errorf("unsupported scheme: %q", uri.Scheme)
}
