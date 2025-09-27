// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestParseOutputFile(t *testing.T) {
	testCases := []struct {
		name        string
		rs          io.ReadSeeker
		expected    map[string]string
		expectedErr string
		initialRead int
	}{
		{
			name:     "empty file",
			rs:       strings.NewReader(""),
			expected: map[string]string{},
		},
		{
			name: "single key value pair",
			rs:   strings.NewReader("a=b"),
			expected: map[string]string{
				"a": "b",
			},
		},
		{
			name: "multiple key value pair",
			rs: strings.NewReader(`
foo=bar
a=b`),
			expected: map[string]string{
				"a":   "b",
				"foo": "bar",
			},
		},
		{
			name: "invalid multiline value",
			rs: strings.NewReader(`
a=b
multiline<<1
2
3`),
			expected:    nil,
			expectedErr: "invalid syntax: multiline value not terminated",
		},
		{
			name: "missing delimiter",
			rs: strings.NewReader(`
a=b
multiline<<
2`),
			expected:    nil,
			expectedErr: "invalid syntax: missing delimiter after '<<'",
		},
		{
			name: "non-delimited multiline value",
			rs: strings.NewReader(`
a=b
multiline
2`),
			expected:    nil,
			expectedErr: "invalid syntax: non-delimited multiline value",
		},
		{
			name: "multiline value with delimiter",
			rs: strings.NewReader(`
a=b
multiline<<EOF
1
2
3
EOF
c=d`),
			expected: map[string]string{
				"a":         "b",
				"c":         "d",
				"multiline": "1\n2\n3",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.initialRead != 0 {
				_, err := tc.rs.Seek(0, tc.initialRead)
				require.NoError(t, err)
			}

			outputs, err := ParseOutput(tc.rs)
			if err != nil {
				require.EqualError(t, err, tc.expectedErr)
			}
			require.Len(t, outputs, len(tc.expected))
			for k, v := range tc.expected {
				require.Equal(t, v, outputs[k])
			}
		})
	}

	t.Run("output dne", func(t *testing.T) {
		tmp := t.TempDir()
		f, err := os.Create(filepath.Join(tmp, "output.txt"))
		t.Cleanup(func() {
			_ = f.Close()
		})
		require.NoError(t, err)
		require.NoError(t, f.Close())
		err = os.Remove(filepath.Join(tmp, "output.txt"))
		require.NoError(t, err)
		outputs, err := ParseOutput(f)
		require.Nil(t, outputs)
		require.ErrorIs(t, err, os.ErrClosed)
	})

	t.Run("output hits size limit", func(t *testing.T) {
		tmp := t.TempDir()
		f, err := os.Create(filepath.Join(tmp, "output.txt"))
		t.Cleanup(func() {
			_ = f.Close()
		})
		require.NoError(t, err)
		err = f.Truncate(51 << 20) // sparse 50+ MB
		require.NoError(t, err)
		outputs, err := ParseOutput(f)
		require.Nil(t, outputs)
		require.EqualError(t, err, "output file too large")
	})

	t.Run("fail to seek", func(t *testing.T) {
		fsys := afero.NewMemMapFs()
		f, err := fsys.Create("output.txt")
		t.Cleanup(func() {
			_ = f.Close()
		})
		require.NoError(t, err)
		// deliberately close
		require.NoError(t, f.Close())

		outputs, err := ParseOutput(f)
		require.Nil(t, outputs)
		require.ErrorContains(t, err, afero.ErrFileClosed.Error())
	})
}
