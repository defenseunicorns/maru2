// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package uses provides a cache+clients for storing and retrieving remote workflows.
package uses

import (
	"context"
	"io"
	"net/url"
)

// DefaultFileName is the default file name to use when a path resolves to "."
const DefaultFileName = "tasks.yaml"

// DefaultVersion is the default version to use when a version is not specified
const DefaultVersion = "main"

// QualifierTokenFromEnv is the qualifier for the token to use when fetching a package
const QualifierTokenFromEnv = "token-from-env"

// QualifierBaseURL is the qualifier for the base URL to use when fetching a package
const QualifierBaseURL = "base-url"

// QualifierTask is the qualifier for the task to use when fetching a package
const QualifierTask = "task"

// OCIQueryParamPlainHTTP is the query param for the OCI client to use plain HTTP
const OCIQueryParamPlainHTTP = "plain-http"

// OCIQueryParamInsecureSkipTLSVerify is the query param for the OCI client to allow for an insecure HTTPS connection
const OCIQueryParamInsecureSkipTLSVerify = "insecure-skip-tls-verify"

// Fetcher fetches a file from a remote location.
type Fetcher interface {
	Fetch(context.Context, *url.URL) (io.ReadCloser, error)
}
