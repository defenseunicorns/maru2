// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package uses provides clients for retrieving remote workflows.
package uses

import (
	"context"
	"io"

	"github.com/package-url/packageurl-go"
)

// DefaultFileName is the default file name to use when a path resolves to "."
const DefaultFileName = "tasks.yaml"

// DefaultVersion is the default version to use when a version is not specified
const DefaultVersion = "main"

// QualifierTokenFromEnv is the qualifier for the token to use when fetching a package
const QualifierTokenFromEnv = "token-from-env"

// QualifierBaseURL is the qualifier for the base URL to use when fetching a package
const QualifierBaseURL = "base"

// QualifierTask is the qualifier for the task to use when fetching a package
const QualifierTask = "task"

// Fetcher fetches a file from a remote location.
type Fetcher interface {
	Fetch(context.Context, *URI) (io.ReadCloser, error)
}

// PackageAliasMapper handles mapping package URL aliases to their resolved forms
type PackageAliasMapper interface {
	ResolveAlias(packageurl.PackageURL) (packageurl.PackageURL, bool)
}
