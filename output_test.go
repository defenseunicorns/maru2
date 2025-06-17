// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"strings"
	"testing"

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
}
