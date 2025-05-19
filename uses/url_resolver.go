// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/package-url/packageurl-go"
)

// ResolveURL resolves a URI relative to a previous URI.
// It handles different schemes (file, http, https, pkg) and resolves relative paths.
func ResolveURL(p, u string, resolvers ...AliasResolver) (string, error) {
	prev, err := url.Parse(p)
	if err != nil {
		return "", err
	}

	uri, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	if err := validateURIs(prev, uri); err != nil {
		return "", err
	}

	switch {
	// file -> https, http, pkg
	case prev.Scheme == "file" && (uri.Scheme == "https" || uri.Scheme == "http" || uri.Scheme == "pkg"),
		// https, http -> https, http
		(prev.Scheme == "https" || prev.Scheme == "http") && (uri.Scheme == "https" || uri.Scheme == "http"),
		// pkg -> pkg
		prev.Scheme == "pkg" && uri.Scheme == "pkg",
		// https, http -> pkg
		(prev.Scheme == "https" || prev.Scheme == "http") && uri.Scheme == "pkg":

		if uri.Scheme == "pkg" {
			pURL, err := packageurl.FromString(u)
			if err != nil {
				return "", err
			}
			if pURL.Subpath == "" {
				pURL.Subpath = DefaultFileName
			}
			if pURL.Version == "" {
				pURL.Version = DefaultVersion
			}
			for _, resolver := range resolvers {
				if resolver == nil {
					continue
				}
				resolvedPURL, isAlias := resolver.ResolveAlias(pURL)
				if isAlias {
					return resolvedPURL.String(), nil
				}
			}
			return pURL.String(), nil
		}
		return u, nil

	// file -> file
	case prev.Scheme == "file" && uri.Scheme == "file":
		return resolveFileToFile(prev, uri)

	// http(s) -> file
	case (prev.Scheme == "https" || prev.Scheme == "http") && uri.Scheme == "file":
		return resolveHTTPToFile(prev, uri)

	// pkg -> file
	case prev.Scheme == "pkg" && uri.Scheme == "file":
		return resolvePkgToFile(p, uri, resolvers...)
	}

	// This should be unreachable
	return "", fmt.Errorf("unable to resolve %q to %q", p, u)
}

func validateURIs(prev, uri *url.URL) error {
	if uri.Scheme == "" {
		return fmt.Errorf("must contain a scheme: %q", uri)
	}

	if !slices.Contains([]string{"file", "http", "https", "pkg"}, uri.Scheme) {
		return fmt.Errorf("unsupported scheme: %q", uri.Scheme)
	}

	if uri.Opaque == "." {
		return fmt.Errorf("invalid relative path \".\"")
	}

	if uri.Scheme == "file" && (uri.Path == "" && uri.Opaque == "") {
		return fmt.Errorf("invalid path %q", uri)
	}

	if uri.Scheme == "file" && strings.HasPrefix(uri.Path, "/") {
		return fmt.Errorf("absolute path %q", uri)
	}

	if prev.Scheme == "" {
		return fmt.Errorf("must contain a scheme: %q", prev)
	}

	return nil
}

func resolveFileToFile(prev, uri *url.URL) (string, error) {
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
	return uri.String(), nil
}

func resolveHTTPToFile(prev, uri *url.URL) (string, error) {
	next := *prev // https://github.com/golang/go/issues/38351
	next.Path = filepath.Join(filepath.Dir(prev.Path), uri.Opaque)
	if next.Path == "." || next.Path == "/" {
		next.Path = "/" + DefaultFileName
	}
	next.RawQuery = uri.RawQuery
	return next.String(), nil
}

func resolvePkgToFile(p string, uri *url.URL, resolvers ...AliasResolver) (string, error) {
	pURL, err := packageurl.FromString(p)
	if err != nil {
		return "", err
	}
	pURL.Subpath = filepath.Join(filepath.Dir(pURL.Subpath), uri.Opaque)
	if pURL.Subpath == "." {
		pURL.Subpath = DefaultFileName
	}
	if pURL.Version == "" {
		pURL.Version = DefaultVersion
	}

	if taskName := uri.Query().Get(QualifierTask); taskName != "" {
		qm := pURL.Qualifiers.Map()
		qm[QualifierTask] = taskName
		pURL.Qualifiers = packageurl.QualifiersFromMap(qm)
	}

	for _, resolver := range resolvers {
		if resolver == nil {
			continue
		}
		resolvedPURL, isAlias := resolver.ResolveAlias(pURL)
		if isAlias {
			pURL = resolvedPURL
		}
	}

	return pURL.String(), nil
}
