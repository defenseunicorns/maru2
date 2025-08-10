// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name        string
		prev        string
		uri         string
		aliases     v0.AliasMap
		next        string
		expectedErr string
	}{
		{
			name: "http -> file",
			prev: "http://example.com/dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "http://example.com/dir/foo.yaml",
		},
		{
			name: "https -> file",
			prev: "https://example.com/dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "https://example.com/dir/foo.yaml",
		},
		{
			name: "http -> file with double dot path",
			prev: "http://example.com/dir/bar.yaml",
			uri:  "file:..",
			next: "http://example.com/tasks.yaml",
		},
		{
			name: "http -> file with dot path",
			prev: "http://example.com/dir/bar.yaml",
			uri:  "file:.",
			next: "http://example.com/dir",
		},
		{
			name: "pkg -> file",
			prev: "pkg:github/owner/repo@main#dir/bar.yaml",
			uri:  "file:foo.yaml",
			next: "pkg:github/owner/repo@main#dir/foo.yaml",
		},
		{
			name:        "nil prev: invalid pkg url", // https://raw.githubusercontent.com/package-url/purl-spec/master/test-suite-data.json
			uri:         "pkg:EnterpriseLibrary.Common@6.0.1304",
			expectedErr: "purl is missing type or name",
		},
		{
			name: "pkg -> file with dot path",
			prev: "pkg:github/owner/repo@main#dir/bar.yaml",
			uri:  "file:.",
			next: "pkg:github/owner/repo@main#dir",
		},
		{
			name: "file -> file",
			prev: "file:foo.yaml",
			uri:  "file:bar.yaml",
			next: "file:bar.yaml",
		},
		{
			name: "file -> file with dot path",
			prev: "file:foo.yaml",
			uri:  "file:.",
			next: "file:.",
		},
		{
			name: "http -> http",
			prev: "http://example.com/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "http -> pkg",
			prev: "http://example.com/foo.yaml",
			uri:  "pkg:github/owner/repo",
			next: "pkg:github/owner/repo@main#tasks.yaml",
		},
		{
			name: "http -> file with task param",
			prev: "http://127.0.0.1:43951/foo.yaml",
			uri:  "file:bar/baz.yaml?task=baz",
			next: "http://127.0.0.1:43951/bar/baz.yaml?task=baz",
		},
		{
			name: "pkg -> pkg with no subpath",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo",
			next: "pkg:github/owner/repo@main#tasks.yaml",
		},
		{
			name: "pkg -> pkg with no version",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo#dir/foo.yaml",
			next: "pkg:github/owner/repo@main#dir/foo.yaml",
		},
		{
			name: "pkg -> pkg with version and subpath",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			next: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
		},
		{
			name: "pkg -> pkg with task param",
			prev: "file:/dir/bar.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/foo.yaml?task=bar",
			next: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml?task=bar",
		},
		{
			name: "pkg -> file with path traversal",
			prev: "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			uri:  "file:../tasks/foo.yaml",
			next: "pkg:github/owner/repo@v1.0.0#tasks/foo.yaml",
		},
		{
			name: "pkg -> file with path traversal and task param",
			prev: "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			uri:  "file:../tasks/foo.yaml?task=bar",
			next: "pkg:github/owner/repo@v1.0.0?task=bar#tasks/foo.yaml",
		},
		{
			name:        "file -> file with invalid uri parse",
			prev:        "file:foo.yaml",
			uri:         "http://invalid%url",
			expectedErr: "parse \"http://invalid%url\": invalid URL escape \"%ur\"",
		},
		{
			name:        "file -> file with no scheme",
			prev:        "file:foo.yaml",
			uri:         "no-scheme",
			expectedErr: `unsupported scheme: "" in "no-scheme"`,
		},
		{
			name:        "file with no scheme -> file",
			prev:        "no-scheme",
			uri:         "file:foo.yaml",
			expectedErr: `unsupported scheme: "" in "no-scheme"`,
		},
		{
			name: "file -> file with directory path",
			prev: "file:dir/foo.yaml",
			uri:  "file:bar.yaml",
			next: "file:dir/bar.yaml",
		},
		{
			name: "file -> file",
			prev: "file://dir/foo.yaml",
			uri:  "file:bar.yaml",
			next: "file:bar.yaml",
		},
		{
			name: "file -> file with directory path and dot replacement",
			prev: "file:dir/foo.yaml",
			uri:  "file:.",
			next: "file:dir",
		},
		{
			name: "pkg -> pkg",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "pkg:github/owner/repo2@v2.0.0#dir/bar.yaml",
			next: "pkg:github/owner/repo2@v2.0.0#dir/bar.yaml",
		},
		{
			name: "pkg -> http",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "file -> http",
			prev: "file:dir/foo.yaml",
			uri:  "http://example.com/bar.yaml",
			next: "http://example.com/bar.yaml",
		},
		{
			name: "file -> https",
			prev: "file:dir/foo.yaml",
			uri:  "https://example.com/bar.yaml",
			next: "https://example.com/bar.yaml",
		},
		{
			name: "file -> pkg",
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
			name: "file -> file with dot replacement in next.Opaque",
			prev: "file:dir/foo.yaml",
			uri:  "file:.",
			next: "file:dir",
		},
		{
			name: "nil prev: pkg",
			uri:  "pkg:github/owner/repo",
			next: "pkg:github/owner/repo@main#tasks.yaml",
		},
		{
			name: "pkg -> file with subpath",
			prev: "pkg:github/owner/repo@v1.0.0#.",
			uri:  "file:foo.yaml",
			next: "pkg:github/owner/repo@v1.0.0#foo.yaml",
		},
		{
			name: "pkg -> file with empty version",
			prev: "pkg:github/owner/repo#dir/foo.yaml",
			uri:  "file:bar.yaml",
			next: "pkg:github/owner/repo@main#dir/bar.yaml",
		},
		{
			name: "file -> file with directory path and dot opaque",
			prev: "file:dir/foo.yaml",
			uri:  "file:.",
			next: "file:dir",
		},
		{
			name: "pkg -> file with dot subpath replacement",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:../.",
			next: "pkg:github/owner/repo@v1.0.0#tasks.yaml",
		},
		{
			name: "pkg -> file up one dir",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:..",
			next: "pkg:github/owner/repo@v1.0.0#tasks.yaml",
		},
		{
			name: "file -> file with next.Opaque equals dot",
			prev: "file:dir/foo.yaml",
			uri:  "file:.",
			next: "file:dir",
		},
		{
			name: "file -> pkg with alias resolution",
			prev: "file:dir/foo.yaml",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/bar.yaml",
			aliases: v0.AliasMap{
				"github": {
					Type: "github",
					Base: "https://github.com/",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base=https%3A%2F%2Fgithub.com%2F#dir/bar.yaml",
		},
		{
			name: "pkg -> file with alias resolution",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:bar.yaml",
			aliases: v0.AliasMap{
				"github": {
					Type: "github",
					Base: "https://github.com",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base=https%3A%2F%2Fgithub.com#dir/bar.yaml",
		},
		{
			name: "pkg -> file with task param and alias resolution",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:bar.yaml?task=baz",
			aliases: v0.AliasMap{
				"github": {
					Type:         "github",
					Base:         "https://github.com",
					TokenFromEnv: "GITHUB_TOKEN",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base=https%3A%2F%2Fgithub.com&task=baz&token-from-env=GITHUB_TOKEN#dir/bar.yaml",
		},
		{
			name:        "pkg -> file with invalid package URL",
			prev:        "pkg:invalid",
			uri:         "file:foo.yaml",
			expectedErr: "purl is missing type or name",
		},
		{
			name: "file -> file with next.Opaque as dot nested",
			prev: "file:dir/sub/subdir/foo.yaml",
			uri:  "file:..",
			next: "file:dir/sub", // only time a join doesn't result in a .yaml
		},
		{
			name: "file -> file with next.Opaque as dot",
			prev: "file:dir/foo.yaml",
			uri:  "file:..",
			next: "file:tasks.yaml",
		},
		{
			name: "nil prev: file",
			uri:  "file:foo/bar.yaml",
			next: "file:foo/bar.yaml",
		},
		{
			name: "nil prev: file without scheme",
			uri:  "foo/bar.yaml",
			next: "file:foo/bar.yaml",
		},
		{
			name: "nil prev: abs file without scheme",
			uri:  "/foo/bar.yaml",
			next: "file:/foo/bar.yaml",
		},
		{
			name:        "file without scheme",
			uri:         "foo/bar.yaml",
			expectedErr: "unsupported scheme: \"\" in \"foo/bar.yaml\"",
		},
		{
			name: "relative file -> abs file",
			prev: "file:foo/bar.yaml",
			uri:  "file:/",
			next: "file:/",
		},
		{
			name: "oci -> file",
			prev: "oci:registry.uds.sh/maru2:latest",
			uri:  "file:foo.yaml",
			next: "oci:registry.uds.sh/maru2:latest#foo.yaml",
		},
		{
			name: "oci -> nested",
			prev: "oci:registry.uds.sh/maru2:latest#foo.yaml",
			uri:  "file:dir/foo.yaml",
			next: "oci:registry.uds.sh/maru2:latest#dir/foo.yaml",
		},
		{
			name: "oci -> pkg",
			prev: "oci:registry.uds.sh/maru2:latest",
			uri:  "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			next: "oci:registry.uds.sh/maru2:latest#pkg:github/owner/repo@v1.0.0%23dir/foo.yaml",
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
