// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name        string
		prev        string
		uri         string
		next        string
		expectedErr string
	}{
		{
			name: "file with http previous",
			prev: "http://example.com/dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "http://example.com/dir/foo.yaml",
		},
		{
			name: "file with https previous",
			prev: "https://example.com/dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "https://example.com/dir/foo.yaml",
		},
		{
			name: "file with double dot path and http previous",
			prev: "http://example.com/dir/bar.yaml",
			uri:  "file:..",
			next: "http://example.com/tasks.yaml",
		},
		{
			name:        "file with dot path and http previous",
			prev:        "http://example.com/dir/bar.yaml",
			uri:         "file:.",
			next:        "",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "file with pkg previous",
			prev: "pkg:github/owner/repo@main#dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "pkg:github/owner/repo@main#dir/foo.yaml",
		},
		{
			name:        "invalid pkg url",
			uri:         "file:foo.yaml",
			prev:        "pkg://invalid%url",
			expectedErr: "parse \"pkg://invalid%url\": invalid URL escape \"%ur\"",
		},
		{
			name:        "file with dot path and pkg previous",
			prev:        "pkg:github/owner/repo@main#dir/bar.yaml",
			uri:         "file:.",
			next:        "",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "file with file previous",
			prev: "file:foo.yaml",
			uri:  "file:bar.yaml",
			next: "file:bar.yaml",
		},
		{
			name:        "file with dot path and file previous",
			prev:        "file:foo.yaml",
			uri:         "file:.",
			next:        "",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "http with any previous",
			prev: "http://example.com/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "pkg with no subpath",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo",
			next: "pkg:github/owner/repo",
		},
		{
			name: "pkg with no version",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo#dir/foo.yaml",
			next: "pkg:github/owner/repo#dir/foo.yaml",
		},
		{
			name: "pkg with version and subpath",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			next: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
		},
		{
			name: "pkg with task param",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/foo.yaml?task=bar",
			next: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml?task=bar",
		},
		{
			name: "pkg with path traversal",
			prev: "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			uri:  "file:../tasks/foo.yaml",
			next: "pkg:github/owner/repo@v1.0.0#tasks/foo.yaml",
		},
		{
			name: "pkg with path traversal and task param",
			prev: "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			uri:  "file:../tasks/foo.yaml?task=bar",
			next: "pkg:github/owner/repo@v1.0.0?task=bar#tasks/foo.yaml",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			next, err := ResolveURL(tc.prev, tc.uri)

			if tc.expectedErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.next, next)
		})
	}
}
