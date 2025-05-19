// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package uses provides clients for retrieving remote workflows.
package uses

import (
	"context"
	"io"
)

// DefaultFileName is the default file name to use when a path resolves to "."
const DefaultFileName = "tasks.yaml"

// QualifierTokenFromEnv is the qualifier for the token to use when fetching a package
const QualifierTokenFromEnv = "token-from-env"

// QualifierBaseURL is the qualifier for the base URL to use when fetching a package
const QualifierBaseURL = "base"

// Fetcher fetches a file from a remote location.
type Fetcher interface {
	Fetch(context.Context, string) (io.ReadCloser, error)
}
