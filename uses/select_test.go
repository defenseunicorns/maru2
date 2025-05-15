// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestSelectFetcher(t *testing.T) {
	defaultPrev := "file:tmp/test"

	tests := []struct {
		name        string
		uri         string
		prev        string
		want        Fetcher
		expectedErr string
	}{
		{
			name:        "file",
			uri:         "file:tmp/test",
			prev:        defaultPrev,
			want:        NewLocalFetcher(afero.NewOsFs()),
			expectedErr: "",
		},
		{
			name:        "file with http prev",
			uri:         "file:tmp/test",
			prev:        "http://example.com",
			want:        NewHTTPFetcher(),
			expectedErr: "",
		},
		{
			name:        "file with pkg prev",
			uri:         "file:tmp/test",
			prev:        "pkg:github/noxsios/vai",
			want:        NewGitHubClient(),
			expectedErr: "",
		},
		{
			name:        "abs file",
			uri:         "file:///tmp/test",
			prev:        defaultPrev,
			want:        NewLocalFetcher(afero.NewOsFs()),
			expectedErr: "",
		},
		{
			name:        "http",
			uri:         "http://example.com",
			prev:        defaultPrev,
			want:        NewHTTPFetcher(),
			expectedErr: "",
		},
		{
			name:        "https",
			uri:         "https://example.com",
			prev:        defaultPrev,
			want:        NewHTTPFetcher(),
			expectedErr: "",
		},
		{
			name:        "pkg-github",
			uri:         "pkg:github/noxsios/vai",
			prev:        defaultPrev,
			want:        NewGitHubClient(),
			expectedErr: "",
		},
		{
			name:        "unsupported scheme",
			uri:         "ftp://example.com",
			prev:        defaultPrev,
			want:        nil,
			expectedErr: `unsupported scheme: "ftp"`,
		},
		{
			name:        "unsupported previous scheme",
			uri:         "file:tmp/test",
			prev:        "ftp://example.com",
			want:        nil,
			expectedErr: `unsupported scheme: "ftp"`,
		},
		{
			name:        "unsupported type",
			uri:         "pkg:unsupported/noxsios/vai",
			prev:        defaultPrev,
			want:        nil,
			expectedErr: `unsupported type: "unsupported"`,
		},
		{
			name:        "unsupported previous type",
			uri:         "file:tmp/test",
			prev:        "pkg:unsupported/noxsios/vai",
			want:        nil,
			expectedErr: `unsupported type: "unsupported"`,
		},
		{
			name:        "invalid previous package-url",
			uri:         "file:tmp/test",
			prev:        "pkg:",
			want:        nil,
			expectedErr: "purl is missing type or name",
		},
		{
			name:        "invalid package-url",
			uri:         "pkg:",
			prev:        defaultPrev,
			want:        nil,
			expectedErr: "purl is missing type or name",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			uri, err := url.Parse(tt.uri)
			require.NoError(t, err)

			previous, err := url.Parse(tt.prev)
			require.NoError(t, err)

			got, err := SelectFetcher(uri, previous)
			if err != nil {
				require.EqualError(t, err, tt.expectedErr)
			}

			require.Equal(t, tt.want, got)
		})
	}

	t.Run("pkg-gitlab", func(t *testing.T) {
		testCases := []struct {
			name string
			uri  string
			prev string
			base string
		}{
			{
				name: "default",
				uri:  "pkg:gitlab/noxsios/vai",
				prev: defaultPrev,
				base: "",
			},
			{
				name: "default from previous",
				uri:  defaultPrev,
				prev: "pkg:gitlab/noxsios/vai",
				base: "",
			},
			{
				name: "gitlab.com",
				uri:  "pkg:gitlab/noxsios/vai",
				prev: defaultPrev,
				base: "https://gitlab.com",
			},
			{
				name: "custom",
				uri:  "pkg:gitlab/noxsios/vai?base=https://gitlab.example.com",
				prev: defaultPrev,
				base: "https://gitlab.example.com",
			},
		}

		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				uri, err := url.Parse(tc.uri)
				require.NoError(t, err)

				previous, err := url.Parse(tc.prev)
				require.NoError(t, err)

				want, err := NewGitLabClient(tc.base)
				require.NoError(t, err)

				got, err := SelectFetcher(uri, previous)
				require.NoError(t, err)
				require.IsType(t, want, got)
				if got, ok := got.(*GitLabClient); ok && tc.base != "" {
					require.Equal(t, want.client.BaseURL(), got.client.BaseURL())
				} else {
					require.Equal(t, "https://gitlab.com/api/v4/", got.client.BaseURL().String())
				}
			})
		}
	})
}
