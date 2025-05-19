// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"io"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitLabFetcher(t *testing.T) {
	t.Run("basic fetch", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping tests that require network access")
		}

		uses := "pkg:gitlab/noxsios/vai@main?task=hello-world#vai.yaml"

		ctx := log.WithContext(t.Context(), log.New(io.Discard))

		client, err := NewGitLabClient("", "")
		require.NoError(t, err)

		rc, err := client.Fetch(ctx, "file:foo.yaml")
		require.EqualError(t, err, `purl scheme is not "pkg": file`)
		assert.Nil(t, rc)

		rc, err = client.Fetch(ctx, "pkg:github/foo.yaml")
		require.EqualError(t, err, `purl type is not "gitlab": "github"`)
		assert.Nil(t, rc)

		rc, err = client.Fetch(ctx, uses)
		require.NoError(t, err)

		b, err := io.ReadAll(rc)
		require.NoError(t, err)

		assert.Equal(t, `# yaml-language-server: $schema=vai.schema.json

hello-world:
  - run: echo "Hello, World!"
`, string(b))
	})

	t.Run("environment variables", func(t *testing.T) {
		// Test with default token env
		_, err := NewGitLabClient("", "")
		require.NoError(t, err)

		// Test with custom token env that doesn't exist
		customEnv := "CUSTOM_GITLAB_TOKEN"
		_, err = NewGitLabClient("", customEnv)
		require.Error(t, err)
		assert.Contains(t, err.Error(), customEnv)

		// Test with custom token env that exists
		t.Setenv(customEnv, "dummy-token")
		client, err := NewGitLabClient("", customEnv)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("base url", func(t *testing.T) {
		// Test with default base URL
		client, err := NewGitLabClient("", "")
		require.NoError(t, err)
		assert.NotNil(t, client)

		// Test with custom base URL
		baseURL := "https://gitlab.example.com"
		client, err = NewGitLabClient(baseURL, "")
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}
