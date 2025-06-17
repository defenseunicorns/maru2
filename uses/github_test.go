// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubFetcher(t *testing.T) {
	t.Run("basic fetch", func(t *testing.T) {
		t.Parallel()
		if testing.Short() {
			t.Skip("skipping tests that require network access")
		}

		ctx := log.WithContext(t.Context(), log.New(io.Discard))

		client, err := NewGitHubClient(nil, "", "")
		require.NoError(t, err)

		rc, err := client.Fetch(ctx, nil)
		assert.Nil(t, rc)
		require.EqualError(t, err, `uri is nil`)

		u, err := url.Parse("file:foo.yaml")
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		assert.Nil(t, rc)
		require.EqualError(t, err, `purl scheme is not "pkg": "file"`)

		u, err = url.Parse("pkg:gitlab/foo.yaml")
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		assert.Nil(t, rc)
		require.EqualError(t, err, `purl type is not "github": "gitlab"`)

		u, err = url.Parse("pkg:github/noxsios/vai@main?task=echo#testdata/simple.yaml")
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		require.NoError(t, err)

		b, err := io.ReadAll(rc)
		require.NoError(t, err)

		assert.Equal(t, `# yaml-language-server: $schema=../vai.schema.json

echo:
  - run: |
      echo "$MESSAGE"
    with:
      message: input
`, string(b))
	})

	t.Run("environment variables", func(t *testing.T) {
		_, err := NewGitHubClient(nil, "", "")
		require.NoError(t, err)

		customEnv := "CUSTOM_GITHUB_TOKEN"
		_, err = NewGitHubClient(nil, "", customEnv)
		require.Error(t, err)
		assert.Contains(t, err.Error(), customEnv)

		t.Setenv(customEnv, "dummy-token")
		client, err := NewGitHubClient(nil, "", customEnv)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("base url", func(t *testing.T) {
		t.Parallel()
		_, err := NewGitHubClient(nil, ":%invalid", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid base URL")
		baseURL := "https://github.example.com"
		client, err := NewGitHubClient(nil, baseURL, "")
		require.NoError(t, err)
		assert.NotNil(t, client)

		actualBaseURL := client.client.BaseURL.String()
		expectedBaseURL := baseURL
		if !strings.HasSuffix(expectedBaseURL, "/") {
			expectedBaseURL += "/"
		}
		assert.Equal(t, expectedBaseURL, actualBaseURL)
	})
}
