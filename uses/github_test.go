// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"
)

func TestGitHubFetcher(t *testing.T) {
	t.Run("basic fetch", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping tests that require network access")
		}

		uses := "pkg:github/noxsios/vai@main?task=echo#testdata/simple.yaml"

		ctx := log.WithContext(t.Context(), log.New(io.Discard))

		client, err := NewGitHubClient("", "")
		require.NoError(t, err)

		rc, err := client.Fetch(ctx, uses)
		require.NoError(t, err)

		b, err := io.ReadAll(rc)
		require.NoError(t, err)

		require.Equal(t, `# yaml-language-server: $schema=../vai.schema.json

echo:
  - run: |
      echo "$MESSAGE"
    with:
      message: input
`, string(b))
	})

	t.Run("environment variables", func(t *testing.T) {
		// Test with default token env
		_, err := NewGitHubClient("", "")
		require.NoError(t, err)

		// Test with custom token env that doesn't exist
		customEnv := "CUSTOM_GITHUB_TOKEN"
		_, err = NewGitHubClient("", customEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), customEnv)

		// Test with custom token env that exists
		t.Setenv(customEnv, "dummy-token")
		client, err := NewGitHubClient("", customEnv)
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("base url", func(t *testing.T) {
		// Test with invalid base URL
		_, err := NewGitHubClient(":%invalid", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid base URL")

		// Test with valid base URL
		baseURL := "https://github.example.com"
		client, err := NewGitHubClient(baseURL, "")
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the base URL was set correctly
		actualBaseURL := client.client.BaseURL.String()
		expectedBaseURL := baseURL
		if !strings.HasSuffix(expectedBaseURL, "/") {
			expectedBaseURL += "/"
		}
		require.Equal(t, expectedBaseURL, actualBaseURL)
	})
}
