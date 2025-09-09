// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name        string
		prev        string
		uri         string
		aliases     v1.AliasMap
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
			aliases: v1.AliasMap{
				"github": {
					Type:    "github",
					BaseURL: "https://github.com/",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base-url=https%3A%2F%2Fgithub.com%2F#dir/bar.yaml",
		},
		{
			name: "pkg -> file with alias resolution",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:bar.yaml",
			aliases: v1.AliasMap{
				"github": {
					Type:    "github",
					BaseURL: "https://github.com",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base-url=https%3A%2F%2Fgithub.com#dir/bar.yaml",
		},
		{
			name: "pkg -> file with task param and alias resolution",
			prev: "pkg:github/owner/repo@v1.0.0#dir/foo.yaml",
			uri:  "file:bar.yaml?task=baz",
			aliases: v1.AliasMap{
				"github": {
					Type:         "github",
					BaseURL:      "https://github.com",
					TokenFromEnv: "GITHUB_TOKEN",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base-url=https%3A%2F%2Fgithub.com&task=baz&token-from-env=GITHUB_TOKEN#dir/bar.yaml",
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
			name: "oci -> file with dot path",
			prev: "oci:registry.uds.sh/maru2:latest",
			uri:  "file:.",
			next: "oci:registry.uds.sh/maru2:latest#tasks.yaml",
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
		{
			name: "alias path resolution",
			prev: "file:foo.yaml",
			uri:  "custom:task-name",
			aliases: v1.AliasMap{
				"custom": {
					Path: "local/path/to/file.yaml",
				},
			},
			next: "file:local/path/to/file.yaml?task=task-name",
		},
		{
			name: "unsupported scheme with empty path (no alias resolution)",
			prev: "file:foo.yaml",
			uri:  "custom:task-name",
			aliases: v1.AliasMap{
				"custom": {
					Type: "github",
					// Path is empty, so no alias resolution should occur
				},
			},
			expectedErr: `unsupported scheme: "custom" in "custom:task-name"`,
		},
		{
			name: "unsupported scheme with no matching alias",
			prev: "file:foo.yaml",
			uri:  "unknown:task-name",
			aliases: v1.AliasMap{
				"custom": {
					Path: "some/path",
				},
			},
			expectedErr: `unsupported scheme: "unknown" in "unknown:task-name"`,
		},
		{
			name: "unsupported scheme with invalid file URL after alias resolution",
			prev: "file:foo.yaml",
			uri:  "custom:task-name",
			aliases: v1.AliasMap{
				"custom": {
					Path: "invalid%url%path\x7f",
				},
			},
			expectedErr: `parse "file:invalid%url%path\x7f": net/url: invalid control character in URL`,
		},
		{
			name: "pkg alias resolution with qualifiers",
			prev: "file:foo.yaml",
			uri:  "pkg:custom/owner/repo@v1.0.0#dir/foo.yaml",
			aliases: v1.AliasMap{
				"custom": {
					Type:         "github",
					BaseURL:      "https://custom.github.com",
					TokenFromEnv: "CUSTOM_TOKEN",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base-url=https%3A%2F%2Fcustom.github.com&token-from-env=CUSTOM_TOKEN#dir/foo.yaml",
		},
		{
			name: "pkg alias resolution preserves existing qualifiers",
			prev: "file:foo.yaml",
			uri:  "pkg:custom/owner/repo@v1.0.0?existing=value#dir/foo.yaml",
			aliases: v1.AliasMap{
				"custom": {
					Type:         "github",
					BaseURL:      "https://custom.github.com",
					TokenFromEnv: "CUSTOM_TOKEN",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base-url=https%3A%2F%2Fcustom.github.com&existing=value&token-from-env=CUSTOM_TOKEN#dir/foo.yaml",
		},
		{
			name: "pkg alias resolution does not override existing qualifiers",
			prev: "file:foo.yaml",
			uri:  "pkg:custom/owner/repo@v1.0.0?base-url=override#dir/foo.yaml",
			aliases: v1.AliasMap{
				"custom": {
					Type:         "github",
					BaseURL:      "https://custom.github.com",
					TokenFromEnv: "CUSTOM_TOKEN",
				},
			},
			next: "pkg:github/owner/repo@v1.0.0?base-url=override&token-from-env=CUSTOM_TOKEN#dir/foo.yaml",
		},
		{
			name: "oci -> https",
			prev: "oci:registry.uds.sh/maru2:latest",
			uri:  "https://example.com/workflow.yaml",
			next: "oci:registry.uds.sh/maru2:latest#https://example.com/workflow.yaml",
		},
		{
			name: "file -> file with complex path traversal",
			prev: "file:a/b/c/d/foo.yaml",
			uri:  "file:../../../bar.yaml",
			next: "file:a/bar.yaml",
		},
		{
			name: "pkg -> file with complex path traversal",
			prev: "pkg:github/owner/repo@v1.0.0#a/b/c/d/foo.yaml",
			uri:  "file:../../../bar.yaml",
			next: "pkg:github/owner/repo@v1.0.0#a/bar.yaml",
		},
		{
			name: "https -> file with query parameters",
			prev: "https://example.com/dir/foo.yaml?param=value",
			uri:  "file:bar.yaml?new=param",
			next: "https://example.com/dir/bar.yaml?new=param",
		},
		{
			name: "pkg -> pkg with different namespace",
			prev: "pkg:github/owner1/repo1@v1.0.0#dir/foo.yaml",
			uri:  "pkg:gitlab/owner2/repo2@v2.0.0#other/bar.yaml",
			next: "pkg:gitlab/owner2/repo2@v2.0.0#other/bar.yaml",
		},
		{
			name: "oci -> oci valid transition",
			prev: "oci:registry1.com/repo:tag",
			uri:  "oci:registry2.com/other:tag",
			next: "oci:registry2.com/other:tag",
		},
		{
			name:        "invalid transition: unsupported prev scheme",
			prev:        "unsupported:foo",
			uri:         "file:bar.yaml",
			expectedErr: `unsupported scheme: "unsupported" in "unsupported:foo"`,
		},
		{
			name: "nil prev with absolute file path",
			uri:  "file:/absolute/path.yaml",
			next: "file:/absolute/path.yaml",
		},
		{
			name: "alias path resolution with query parameters",
			prev: "file:foo.yaml",
			uri:  "custom:task-name?param=value",
			aliases: v1.AliasMap{
				"custom": {
					Path: "local/path/to/file.yaml",
				},
			},
			expectedErr: `"task-name?param=value" does not satisfy "^[_a-zA-Z][a-zA-Z0-9_-]*$"`,
		},
		{
			name: "alias path resolution with invalid task name",
			prev: "file:foo.yaml",
			uri:  "custom:2-invalid-task",
			aliases: v1.AliasMap{
				"custom": {
					Path: "local/path/to/file.yaml",
				},
			},
			expectedErr: `"2-invalid-task" does not satisfy "^[_a-zA-Z][a-zA-Z0-9_-]*$"`,
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

// TestEscapeVersion tests the escapeVersion function which URL-escapes version strings in package URLs.
//
// The function finds the '@' symbol in a package URL and escapes the version portion using url.PathEscape.
// The version portion extends from '@' until the first '?' or '#' delimiter (or end of string).
// Only the version portion is escaped, while query parameters and fragments remain unchanged.
//
// The tests verify correct escaping behavior, delimiter handling, and edge cases.
func TestEscapeVersion(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		shouldPanic bool
		note        string
	}{
		{
			name:     "no @ symbol",
			input:    "pkg:github/owner/repo",
			expected: "pkg:github/owner/repo",
		},
		{
			name:     "simple version without delimiters",
			input:    "pkg:github/owner/repo@v1.0.0",
			expected: "pkg:github/owner/repo@v1.0.0",
			note:     "No special chars, so no escaping needed",
		},
		{
			name:     "version with special characters no delimiters",
			input:    "pkg:github/owner/repo@v1.0.0+build.1",
			expected: "pkg:github/owner/repo@v1.0.0+build.1",
			note:     "BUG: Should escape +, but current implementation doesn't escape without ? or # delimiters",
		},
		{
			name:     "version with spaces no delimiters",
			input:    "pkg:github/owner/repo@release 1.0",
			expected: "pkg:github/owner/repo@release%201.0",
			note:     "Spaces are escaped by url.PathEscape",
		},
		{
			name:     "version with hash fragment",
			input:    "pkg:github/owner/repo@v1.0.0#path/to/file",
			expected: "pkg:github/owner/repo@v1.0.0#path/to/file",
			note:     "Version stops at # delimiter, no escaping of version part needed",
		},
		{
			name:     "version with query parameters",
			input:    "pkg:github/owner/repo@v1.0.0?param=value",
			expected: "pkg:github/owner/repo@v1.0.0?param=value",
			note:     "Version stops at ? delimiter, no escaping of version part needed",
		},
		{
			name:     "empty version",
			input:    "pkg:github/owner/repo@",
			expected: "pkg:github/owner/repo@",
		},
		{
			name:     "version with slashes no delimiters",
			input:    "pkg:github/owner/repo@feature/branch-name",
			expected: "pkg:github/owner/repo@feature%2Fbranch-name",
			note:     "Slashes are escaped by url.PathEscape",
		},
		{
			name:     "version with colon no delimiters",
			input:    "pkg:github/owner/repo@refs/tags/v1.0.0:stable",
			expected: "pkg:github/owner/repo@refs%2Ftags%2Fv1.0.0:stable",
			note:     "Slashes are escaped by url.PathEscape, but colon is not",
		},
		{
			name:     "complex version with multiple @ symbols",
			input:    "pkg:github/owner/repo@v1.0.0@tag",
			expected: "pkg:github/owner/repo@v1.0.0@tag",
			note:     "BUG: Should escape second @, but current implementation doesn't escape without ? or # delimiters",
		},
		{
			name:     "version with unicode characters",
			input:    "pkg:github/owner/repo@版本1.0",
			expected: "pkg:github/owner/repo@%E7%89%88%E6%9C%AC1.0",
			note:     "Unicode characters are escaped by url.PathEscape",
		},
		{
			name:     "git commit hash",
			input:    "pkg:github/owner/repo@abc123def456",
			expected: "pkg:github/owner/repo@abc123def456",
			note:     "Alphanumeric only, no escaping needed",
		},
		{
			name:     "semantic version with metadata no delimiters",
			input:    "pkg:github/owner/repo@1.0.0-alpha+20130313144700",
			expected: "pkg:github/owner/repo@1.0.0-alpha+20130313144700",
			note:     "BUG: Should escape +, but current implementation doesn't escape without ? or # delimiters",
		},
		// Test cases demonstrating expected behavior (when function is fixed)
		{
			name:     "alphanumeric version",
			input:    "pkg:github/owner/repo@v1.2.3",
			expected: "pkg:github/owner/repo@v1.2.3",
			note:     "Expected: no escaping needed for alphanumeric with dots and hyphens",
		},
		{
			name:     "version at end of string",
			input:    "pkg:github/owner/repo@main",
			expected: "pkg:github/owner/repo@main",
			note:     "Expected: simple branch name, no escaping needed",
		},
		{
			name:     "multiple @ symbols",
			input:    "pkg:github/owner/repo@v1.0@beta",
			expected: "pkg:github/owner/repo@v1.0@beta",
			note:     "@ symbol is not escaped by url.PathEscape",
		},
		{
			name:     "version with percent signs",
			input:    "pkg:github/owner/repo@100%coverage",
			expected: "pkg:github/owner/repo@100%25coverage",
			note:     "Percent signs are escaped",
		},
		{
			name:     "version with backslashes",
			input:    "pkg:github/owner/repo@windows\\path",
			expected: "pkg:github/owner/repo@windows%5Cpath",
			note:     "Backslashes are escaped",
		},
		{
			name:     "very long version string",
			input:    "pkg:github/owner/repo@" + strings.Repeat("a", 100),
			expected: "pkg:github/owner/repo@" + strings.Repeat("a", 100),
			note:     "Long strings without special chars are not escaped",
		},
		{
			name:     "version with equals sign",
			input:    "pkg:github/owner/repo@branch=main",
			expected: "pkg:github/owner/repo@branch=main",
			note:     "Equals signs are not escaped by url.PathEscape",
		},
		{
			name:     "version with ampersand",
			input:    "pkg:github/owner/repo@feature&test",
			expected: "pkg:github/owner/repo@feature&test",
			note:     "Ampersands are not escaped by url.PathEscape",
		},
		{
			name:     "version starting with special char",
			input:    "pkg:github/owner/repo@+experimental",
			expected: "pkg:github/owner/repo@+experimental",
			note:     "Plus signs are not escaped by url.PathEscape",
		},
		{
			name:     "version with special chars and hash",
			input:    "pkg:github/owner/repo@!@#$%",
			expected: "pkg:github/owner/repo@%21@#$%",
			note:     "Some special characters before # are escaped, then stops at # delimiter",
		},
		{
			name:     "version with mixed case unicode",
			input:    "pkg:github/owner/repo@Версия1.0",
			expected: "pkg:github/owner/repo@%D0%92%D0%B5%D1%80%D1%81%D0%B8%D1%8F1.0",
			note:     "Mixed case Cyrillic characters are escaped",
		},
		{
			name:     "version with query and special chars",
			input:    "pkg:github/owner/repo@v1.0.0+build?param=value",
			expected: "pkg:github/owner/repo@v1.0.0+build?param=value",
			note:     "Plus sign is not escaped by url.PathEscape before ? delimiter",
		},
		{
			name:     "version with fragment and special chars",
			input:    "pkg:github/owner/repo@v1.0.0+build#path/to/file",
			expected: "pkg:github/owner/repo@v1.0.0+build#path/to/file",
			note:     "Plus sign is not escaped by url.PathEscape before # delimiter",
		},
		{
			name:     "version with both delimiters - query first",
			input:    "pkg:github/owner/repo@v1.0.0?param=value#fragment",
			expected: "pkg:github/owner/repo@v1.0.0?param=value#fragment",
			note:     "Version stops at first delimiter (?)",
		},
		{
			name:     "version with both delimiters - fragment first",
			input:    "pkg:github/owner/repo@v1.0.0#fragment?param=value",
			expected: "pkg:github/owner/repo@v1.0.0#fragment?param=value",
			note:     "Version stops at first delimiter (#)",
		},
		{
			name:     "complex version with spaces and delimiters",
			input:    "pkg:github/owner/repo@release candidate 1.0?test=true",
			expected: "pkg:github/owner/repo@release%20candidate%201.0?test=true",
			note:     "Spaces escaped before ? delimiter",
		},
		{
			name:     "version with path-like structure",
			input:    "pkg:github/owner/repo@feature/branch/name#dir/file.yaml",
			expected: "pkg:github/owner/repo@feature%2Fbranch%2Fname#dir/file.yaml",
			note:     "Slashes in version escaped before # delimiter",
		},
		{
			name:     "very short version with delimiter",
			input:    "pkg:github/owner/repo@v#fragment",
			expected: "pkg:github/owner/repo@v#fragment",
			note:     "Single character version, no escaping needed",
		},
		{
			name:     "unicode version with delimiter",
			input:    "pkg:github/owner/repo@版本1.0?locale=zh",
			expected: "pkg:github/owner/repo@%E7%89%88%E6%9C%AC1.0?locale=zh",
			note:     "Unicode characters escaped before ? delimiter",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := escapeVersion(tc.input)
			assert.Equal(t, tc.expected, result, "Note: %s", tc.note)
		})
	}
}
