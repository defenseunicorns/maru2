// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v92/github"
	"github.com/package-url/packageurl-go"
)

// GitHubClient is a client for fetching files from GitHub
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHub client
//
// Uses auth token from tokenEnv > GITHUB_ENV > no auth token
func NewGitHubClient(client *http.Client, base string, tokenEnv string) (*GitHubClient, error) {
	var opts []github.ClientOptionsFunc
	if client != nil {
		opts = append(opts, github.WithHTTPClient(client))
	}

	if tokenEnv == "" {
		tokenEnv = "GITHUB_TOKEN"
	}

	token, ok := os.LookupEnv(tokenEnv)
	if tokenEnv != "GITHUB_TOKEN" && !ok {
		return nil, fmt.Errorf("token environment variable %s is not set", tokenEnv)
	}

	if ok {
		opts = append(opts, github.WithAuthToken(token))
	}

	if base != "" {
		baseURL, err := url.Parse(base)
		if err != nil {
			return nil, fmt.Errorf("invalid base URL: %w", err)
		}

		if !strings.HasSuffix(baseURL.Path, "/") {
			baseURL.Path += "/"
		}
		b := baseURL.String()
		opts = append(opts, github.WithURLs(&b, nil)) // not using upload url currently so leaving nil, this may need to change to WithEnterpriseURLs
	}

	c, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &GitHubClient{client: c}, nil
}

// Fetch downloads a file from GitHub
func (g *GitHubClient) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	if uri == nil {
		return nil, fmt.Errorf("uri is nil")
	}

	pURL, err := packageurl.FromString(uri.String())
	if err != nil {
		return nil, err
	}

	if pURL.Type != packageurl.TypeGithub {
		return nil, fmt.Errorf("purl type is not %q: %q", packageurl.TypeGithub, pURL.Type)
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
