// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns/maru2/config"
	"github.com/package-url/packageurl-go"
)

// ResolveRelative resolves a URI relative to a previous URI.
// It handles different schemes (file, http, https, pkg) and resolves relative paths.
func ResolveRelative(prev *URI, u string, pkgAliases map[string]config.Alias) (*URI, error) {
	uri, err := Parse(u)
	if err != nil {
		return nil, err
	}

	if uri.Scheme == "" {
		return nil, fmt.Errorf("must contain a scheme: %q", uri)
	}

	if uri.Opaque == "." {
		return nil, fmt.Errorf("invalid relative path \".\"")
	}

	if uri.Scheme == "file" && (uri.Path == "" && uri.Opaque == "") {
		return nil, fmt.Errorf("invalid path %q", uri)
	}

	if uri.Scheme == "file" && strings.HasPrefix(uri.Path, "/") {
		return nil, fmt.Errorf("absolute path %q", uri)
	}

	switch {
	case
		// nil -> anything
		prev == nil,
		// file -> https, http, pkg
		prev.Scheme == "file" && (uri.Scheme == "https" || uri.Scheme == "http" || uri.Scheme == "pkg"),
		// https, http -> https, http
		(prev.Scheme == "https" || prev.Scheme == "http") && (uri.Scheme == "https" || uri.Scheme == "http"),
		// pkg -> pkg
		prev.Scheme == "pkg" && uri.Scheme == "pkg",
		// https, http -> pkg
		(prev.Scheme == "https" || prev.Scheme == "http") && uri.Scheme == "pkg":

		if uri.Scheme == "pkg" {
			pURL, err := packageurl.FromString(u)
			if err != nil {
				return nil, err
			}

			if pURL.Subpath == "" {
				pURL.Subpath = DefaultFileName
			}
			if pURL.Version == "" {
				pURL.Version = DefaultVersion
			}
			resolvedPURL, isAlias := ResolveAlias(pURL, pkgAliases)
			if isAlias {
				return Parse(resolvedPURL.String())
			}
			return Parse(pURL.String())
		}
		return uri, nil

	// file -> file
	case prev.Scheme == "file" && uri.Scheme == "file":
		dir := filepath.Dir(prev.URL.Opaque)
		if dir != "." {
			next := &URI{
				URL: &url.URL{
					Scheme:   "file",
					Opaque:   filepath.Join(dir, uri.Opaque),
					RawQuery: uri.RawQuery,
				},
			}
			if next.Opaque == "." {
				next.Opaque = DefaultFileName
			}
			return next, nil
		}
		return uri, nil

	// http(s) -> file
	case (prev.Scheme == "https" || prev.Scheme == "http") && uri.Scheme == "file":
		next := *prev // https://github.com/golang/go/issues/38351
		next.Path = filepath.Join(filepath.Dir(prev.Path), uri.Opaque)
		if next.Path == "." || next.Path == "/" {
			next.Path = "/" + DefaultFileName
		}
		next.RawQuery = uri.RawQuery
		return &next, nil

	// pkg -> file
	case prev.Scheme == "pkg" && uri.Scheme == "file":
		pURL, err := packageurl.FromString(prev.String())
		if err != nil {
			return nil, err
		}

		pURL.Subpath = filepath.Join(filepath.Dir(pURL.Subpath), uri.Opaque)
		if pURL.Subpath == "." {
			pURL.Subpath = DefaultFileName
		}
		if pURL.Version == "" {
			pURL.Version = DefaultVersion
		}

		qm := pURL.Qualifiers.Map()
		qm[QualifierTask] = uri.Query().Get(QualifierTask)
		pURL.Qualifiers = packageurl.QualifiersFromMap(qm)

		resolvedPURL, isAlias := ResolveAlias(pURL, pkgAliases)
		if isAlias {
			pURL = resolvedPURL
		}

		return Parse(pURL.String())
	}

	// This should be unreachable
	return nil, fmt.Errorf("unable to resolve %q to %q", prev, uri)
}
