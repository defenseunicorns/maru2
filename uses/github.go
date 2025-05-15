// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/go-github/v62/github"
	"github.com/package-url/packageurl-go"
)

// GitHubClient is a client for fetching files from GitHub
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient() *GitHubClient {
	client := github.NewClient(nil)

	token, ok := os.LookupEnv("GITHUB_TOKEN")
	if ok {
		client = client.WithAuthToken(token)
	}
	return &GitHubClient{client}
}

// Fetch the file
func (g *GitHubClient) Fetch(ctx context.Context, uses string) (io.ReadCloser, error) {
	pURL, err := packageurl.FromString(uses)
	if err != nil {
		return nil, err
	}

	rc, resp, err := g.client.Repositories.DownloadContents(ctx, pURL.Namespace, pURL.Name, pURL.Subpath, &github.RepositoryContentGetOptions{
		Ref: pURL.Version,
	})

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s: %s", pURL, resp.Status)
	}

	return rc, nil
}
