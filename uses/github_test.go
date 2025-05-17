// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubFetcher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping tests that require network access")
	}

	uses := "pkg:github/noxsios/vai@main?task=echo#testdata/simple.yaml"

	ctx := context.Background()

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
}
