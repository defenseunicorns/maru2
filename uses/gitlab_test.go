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
		t.Parallel()
		if testing.Short() {
			t.Skip("skipping tests that require network access")
		}

		ctx := log.WithContext(t.Context(), log.New(io.Discard))

		client, err := NewGitLabClient(nil, "", "")
		require.NoError(t, err)

		rc, err := client.Fetch(ctx, nil)
		assert.Nil(t, rc)
		require.EqualError(t, err, `uri is nil`)

		u, err := ResolveRelative(nil, "file:foo.yaml", nil)
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		require.EqualError(t, err, `purl scheme is not "pkg": "file"`)
		assert.Nil(t, rc)

		u, err = ResolveRelative(nil, "pkg:github/foo.yaml", nil)
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		require.EqualError(t, err, `purl type is not "gitlab": "github"`)
		assert.Nil(t, rc)

		u, err = ResolveRelative(nil, "pkg:gitlab/noxsios/vai@main?task=hello-world#vai.yaml", nil)
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		require.NoError(t, err)

		b, err := io.ReadAll(rc)
		require.NoError(t, err)

		assert.Equal(t, `# yaml-language-server: $schema=vai.schema.json

hello-world:
  - run: echo "Hello, World!"
`, string(b))

		u, err = ResolveRelative(nil, "pkg:gitlab/noxsios/vai@foo%2Fbar?task=hello-world#vai.yaml", nil)
		require.NoError(t, err)

		rc, err = client.Fetch(ctx, u)
		require.NoError(t, err)

		b, err = io.ReadAll(rc)
		require.NoError(t, err)

		assert.Equal(t, `# yaml-language-server: $schema=vai.schema.json

hello-world:
  - run: echo "Hello, World!"
`, string(b))
	})

	t.Run("environment variables", func(t *testing.T) {
		_, err := NewGitLabClient(nil, "", "")
		require.NoError(t, err)

		customEnv := "CUSTOM_GITLAB_TOKEN"
		_, err = NewGitLabClient(nil, "", customEnv)
		require.Error(t, err)
		assert.Contains(t, err.Error(), customEnv)

		t.Setenv(customEnv, "dummy-token")
		client, err := NewGitLabClient(nil, "", customEnv)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("base url", func(t *testing.T) {
		t.Parallel()
		client, err := NewGitLabClient(nil, "", "")
		require.NoError(t, err)
		assert.NotNil(t, client)

		assert.Equal(t, "https://gitlab.com/api/v4/", client.client.BaseURL().String())

		baseURL := "https://gitlab.example.com/"
		client, err = NewGitLabClient(nil, baseURL, "")
		require.NoError(t, err)
		assert.NotNil(t, client)

		assert.Equal(t, baseURL+"api/v4/", client.client.BaseURL().String())
	})
}
