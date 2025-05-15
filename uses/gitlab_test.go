// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitLabFetcher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping tests that require network access")
	}

	uses := "pkg:gitlab/noxsios/vai@main?task=hello-world#vai.yaml"

	ctx := context.Background()

	client, err := NewGitLabClient("")
	require.NoError(t, err)

	rc, err := client.Fetch(ctx, uses)
	require.NoError(t, err)

	b, err := io.ReadAll(rc)
	require.NoError(t, err)

	require.Equal(t, `# yaml-language-server: $schema=vai.schema.json

hello-world:
  - run: echo "Hello, World!"
`, string(b))
}
