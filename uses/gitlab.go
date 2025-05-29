// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/package-url/packageurl-go"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitLabClient is a client for fetching files from GitLab
type GitLabClient struct {
	client *gitlab.Client
}

// NewGitLabClient creates a new GitLab client
func NewGitLabClient(client *http.Client, base string, tokenEnv string) (*GitLabClient, error) {
	if tokenEnv == "" {
		tokenEnv = "GITLAB_TOKEN"
	}

	token, ok := os.LookupEnv(tokenEnv)
	if tokenEnv != "GITLAB_TOKEN" && !ok {
		return nil, fmt.Errorf("token environment variable %s is not set", tokenEnv)
	}

	if base == "" {
		base = "https://gitlab.com"
	}

	opts := []gitlab.ClientOptionFunc{
		gitlab.WithBaseURL(base),
	}

	if client != nil {
		opts = append(opts, gitlab.WithHTTPClient(client))
	}

	c, err := gitlab.NewClient(token, opts...)
	if err != nil {
		return nil, err
	}
	return &GitLabClient{c}, nil
}

// Fetch downloads a file from GitLab
func (g *GitLabClient) Fetch(ctx context.Context, uri *URI) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("uri is nil")
	}

	pURL, err := packageurl.FromString(uri.String())
	if err != nil {
		return nil, err
	}

	if pURL.Type != packageurl.TypeGitlab {
		return nil, fmt.Errorf("purl type is not %q: %q", packageurl.TypeGitlab, pURL.Type)
	}

	pid := pURL.Namespace + "/" + pURL.Name
	b, resp, err := g.client.RepositoryFiles.GetRawFile(pid, pURL.Subpath, &gitlab.GetRawFileOptions{
		Ref: &pURL.Version,
	}, gitlab.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: %s", pURL, resp.Status)
	}

	return io.NopCloser(bytes.NewReader(b)), nil
}
