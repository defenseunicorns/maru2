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

// Fetcher fetches a file from a remote location.
type Fetcher interface {
	Fetch(context.Context, string) (io.ReadCloser, error)
}
