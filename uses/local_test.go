// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalFetcher(t *testing.T) {
	testCases := []struct {
		name string
		uses string
		rc   io.ReadCloser

		expectedFetchErr string
	}{
		{
			name: "file exists",
			uses: "file:foo.yaml",
			rc:   io.NopCloser(strings.NewReader("hello, world")),
		},
		{
			name:             "file does not exist",
			uses:             "file:baz.yaml",
			expectedFetchErr: "open baz.yaml: file does not exist",
		},
		{
			name:             "invalid uri",
			uses:             "$%#",
			expectedFetchErr: `parse "$%": invalid URL escape "%"`,
		},
		{
			name:             "bad scheme",
			uses:             "http://foo.com/bar.yaml",
			expectedFetchErr: `scheme is not "file"`,
		},
	}

	fs := afero.NewMemMapFs()

	err := afero.WriteFile(fs, "foo.yaml", []byte("hello, world"), 0644)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := NewLocalFetcher(fs)
			ctx := log.WithContext(t.Context(), log.New(io.Discard))

			rc, err := fetcher.Fetch(ctx, tc.uses)
			if tc.expectedFetchErr != "" {
				assert.Nil(t, rc)
				require.EqualError(t, err, tc.expectedFetchErr)
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
}
