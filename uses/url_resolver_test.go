// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"strings"
	"testing"

	"github.com/defenseunicorns/maru2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name        string
		prev        string
		uri         string
		aliases     map[string]config.Alias
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
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "file with pkg previous",
			prev: "pkg:github/owner/repo@main#dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "pkg:github/owner/repo@main#dir/foo.yaml",
		},
		{
			name:        "nil prev with invalid pkg url", // https://raw.githubusercontent.com/package-url/purl-spec/master/test-suite-data.json
			uri:         "pkg:EnterpriseLibrary.Common@6.0.1304",
			expectedErr: "purl is missing type or name",
		},
		{
			name:        "file with dot path and pkg previous",
			prev:        "pkg:github/owner/repo@main#dir/bar.yaml",
			uri:         "file:.",
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
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "http with any previous",
			prev: "http://example.com/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "http to pkg",
			prev: "http://example.com/foo.yaml",
			uri:  "pkg:github/owner/repo",
			next: "pkg:github/owner/repo@main#tasks.yaml",
		},
		{
			name: "http with task param",
			prev: "http://127.0.0.1:43951/foo.yaml",
			uri:  "file:bar/baz.yaml?task=baz",
			next: "http://127.0.0.1:43951/bar/baz.yaml?task=baz",
		},
		{
			name: "pkg with no subpath",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo",
			next: "pkg:github/owner/repo@main#tasks.yaml",
		},
		{
			name: "pkg with no version",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo#dir/foo.yaml",
			next: "pkg:github/owner/repo@main#dir/foo.yaml",
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
		{
			name:        "invalid uri parse",
			prev:        "file:foo.yaml",
			uri:         "http://invalid%url",
			expectedErr: "parse \"http://invalid%url\": invalid URL escape \"%ur\"",
		},
		{
			name:        "uri without scheme",
			prev:        "file:foo.yaml",
			uri:         "no-scheme",
			expectedErr: `unsupported scheme: "" in "no-scheme"`,
		},
		{
			name: "prev without scheme",
			prev: "no-scheme",
			uri:  "file:foo.yaml",
			next: "file:foo.yaml",
		},
		{
			name: "file to file with directory path",
			prev: "file:dir/foo.yaml",
			uri:  "file:bar.yaml",
			next: "file:dir/bar.yaml",
		},
		{
			name: "file to file",
			prev: "file://dir/foo.yaml",
			uri:  "file:bar.yaml",
			next: "file:dir/bar.yaml",
		},
		{
			name:        "file to file with directory path and dot replacement",
			prev:        "file:dir/foo.yaml",
			uri:         "file:.",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "pkg to pkg",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "pkg:github/owner/repo2@v2.0.0#dir/bar.yaml",
			next: "pkg:github/owner/repo2@v2.0.0#dir/bar.yaml",
		},
		{
			name: "pkg to http",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "file to http",
			prev: "file:dir/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "file to https",
			prev: "file:dir/foo.yaml",
			uri:  "https://example.com/bar.yaml",
			next: "https://example.com/bar.yaml",
		},
		{
			name: "file to pkg",
			prev: "file:dir/foo.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			next: "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
		},
		{
			name:        "unsupported scheme",
			prev:        "file:dir/foo.yaml",
			uri:         "ftp://example.com/bar.yaml",
			expectedErr: `unsupported scheme: "ftp" in "ftp://example.com/bar.yaml"`,
		},
		{
			name:        "file to file with dot replacement in next.Opaque",
			prev:        "file:dir/foo.yaml",
			uri:         "file:.",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name:        "nil prev with error in packageurl.FromString",
			uri:         "pkg://invalid%url",
			expectedErr: "parse \"pkg://invalid%url\": invalid URL escape \"%ur\"",
		},
		{
			name: "pkg to file with dot subpath",
			prev: "pkg:github/owner/repo@v1.0.0#.",
			uri:  "file:foo.yaml",
			next: "pkg:github/owner/repo@v1.0.0#foo.yaml",
		},
		{
			name: "pkg to file with empty version",
			prev: "pkg:github/owner/repo#dir/foo.yaml",
			uri:  "file:bar.yaml",
			next: "pkg:github/owner/repo@main#dir/bar.yaml",
		},
		{
			name:        "file to file with directory path and dot opaque",
			prev:        "file:dir/foo.yaml",
			uri:         "file:.",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "pkg to file with dot subpath replacement",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:../.",
			next: "pkg:github/owner/repo@v1.0.0#tasks.yaml",
		},
		{
			name: "pkg to file up one dir",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:..",
			next: "pkg:github/owner/repo@v1.0.0#tasks.yaml",
		},
		{
			name:        "file to file with next.Opaque equals dot",
			prev:        "file:dir/foo.yaml",
			uri:         "file:.",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name:        "file to file with dir path and dot replacement",
			prev:        "file:dir/foo.yaml",
			uri:         "file:.",
			expectedErr: "invalid relative path \".\"",
		},
		{
			name: "pkg with alias resolution",
			prev: "file:dir/foo.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			aliases: map[string]config.Alias{
				"github": {
					Type: "github",
					Base: "github/aliased/repo@v2.0.0",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base=github%2Faliased%2Frepo%40v2.0.0#dir/bar.yaml",
		},
		{
			name: "pkg to file with alias resolution",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:bar.yaml",
			aliases: map[string]config.Alias{
				"github": {
					Type: "github",
					Base: "github/aliased/repo@v2.0.0",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base=github%2Faliased%2Frepo%40v2.0.0#dir/bar.yaml",
		},
		{
			name: "pkg to file with task param and alias resolution",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:bar.yaml?task=baz",
			aliases: map[string]config.Alias{
				"github": {
					Type: "github",
					Base: "github/aliased/repo@v2.0.0",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base=github%2Faliased%2Frepo%40v2.0.0&task=baz#dir/bar.yaml",
		},
		{
			name:        "pkg to file with invalid package URL",
			prev:        "pkg:invalid",
			uri:         "file:foo.yaml",
			expectedErr: "purl is missing type or name",
		},
		{
			name: "file to file with next.Opaque as dot nested",
			prev: "file:dir/sub/subdir/foo.yaml",
			uri:  "file:..",
			next: "file:dir/sub", // only time a join doesn't result in a .yaml
		},
		{
			name: "file to file with next.Opaque as dot",
			prev: "file:dir/foo.yaml",
			uri:  "file:..",
			next: "file:tasks.yaml",
		},
		{
			name: "nil prev with file",
			uri:  "file:foo/bar.yaml",
			next: "file:foo/bar.yaml",
		},
		{
			name: "nil prev with file without scheme",
			uri:  "foo/bar.yaml",
			next: "file://foo/bar.yaml",
		},
		{
			name: "file to file with next.Opaque equals dot",
			prev: "file:foo/bar.yaml",
			uri:  "file:.",
			next: "file:foo/tasks.yaml",
		},
		{
			name: "relative file to abs file",
			prev: "file:foo/bar.yaml",
			uri:  "file:/",
			next: "file:/",
		},
		{
			name:        "invalid path",
			prev:        "file:",
			uri:         "file://",
			expectedErr: "invalid path \"file:\"",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.prev)
			require.NoError(t, err)

			if strings.HasPrefix(tc.name, "nil prev") {
				u = nil
			}

			next, err := ResolveRelative(u, tc.uri, tc.aliases)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, next)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, next.Scheme)
				assert.Equal(t, tc.next, next.String())
			}
		})
	}
}
