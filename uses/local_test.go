// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalFetcher(t *testing.T) {
	testCases := []struct {
		name        string
		uses        string
		rc          io.ReadCloser
		expectedErr string
	}{
		{
			name: "file exists",
			uses: "file:foo.yaml",
			rc:   io.NopCloser(strings.NewReader("hello, world")),
		},
		{
			name:        "file does not exist",
			uses:        "file:baz.yaml",
			expectedErr: "open baz.yaml: file does not exist",
		},
		{
			name:        "is a directory",
			uses:        "file:bar",
			expectedErr: `read bar: is a directory`,
		},
		{
			name:        "bad scheme",
			uses:        "http://foo.com/bar.yaml",
			expectedErr: `scheme is not "file" or empty`,
		},
		{
			name: "empty file",
			uses: "file:zab.yaml",
			rc:   io.NopCloser(strings.NewReader("")),
		},
		{
			name:        "nil uri",
			expectedErr: "uri is nil",
		},
	}

	fs := afero.NewMemMapFs()

	err := afero.WriteFile(fs, "foo.yaml", []byte("hello, world"), 0o644)
	require.NoError(t, err)

	err = fs.Mkdir("bar", 0o755)
	require.NoError(t, err)

	f, err := fs.Create("zab.yaml")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fetcher := NewLocalFetcher(fs)
			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			u, err := url.Parse(tc.uses)
			require.NoError(t, err)

			if tc.name == "nil uri" {
				u = nil
			}

			rc, err := fetcher.Fetch(ctx, u)
			if tc.expectedErr != "" {
				assert.Nil(t, rc)
				require.EqualError(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				b1, err := io.ReadAll(tc.rc)
				require.NoError(t, err)

				b2, err := io.ReadAll(rc)
				require.NoError(t, err)

				assert.Equal(t, string(b1), string(b2))
			}
		})
	}

	t.Run("context is pre-cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		fetcher := NewLocalFetcher(afero.NewMemMapFs())

		rc, err := fetcher.Fetch(ctx, &url.URL{})
		assert.Nil(t, rc)
		require.ErrorIs(t, err, context.Canceled)
	})
}
