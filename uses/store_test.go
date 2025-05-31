// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(fs afero.Fs) error
		expectedErr string
		validate    func(t *testing.T, s *Store)
	}{
		{
			name: "new store without existing index",
			validate: func(t *testing.T, s *Store) {
				assert.NotNil(t, s.index)
				assert.Empty(t, s.index)

				content, err := afero.ReadFile(s.fs, IndexFileName)
				require.NoError(t, err)
				assert.Equal(t, "{}", string(content))
			},
		},
		{
			name: "new store with existing valid index",
			setup: func(fs afero.Fs) error {
				return afero.WriteFile(fs, IndexFileName, []byte(`{"https://example.com": {"Size": 10, "Hex": "abcd1234"}}`), 0644)
			},
			validate: func(t *testing.T, s *Store) {
				assert.NotNil(t, s.index)
				assert.Len(t, s.index, 1)
				assert.Contains(t, s.index, "https://example.com")
				assert.Equal(t, int64(10), s.index["https://example.com"].Size)
				assert.Equal(t, "abcd1234", s.index["https://example.com"].Hex)
			},
		},
		{
			name: "new store with existing invalid index",
			setup: func(fs afero.Fs) error {
				return afero.WriteFile(fs, IndexFileName, []byte(`invalid json`), 0644)
			},
			expectedErr: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "error creating index file",
			setup: func(_ afero.Fs) error {
				// No setup needed, we'll use a read-only filesystem
				return nil
			},
			expectedErr: "operation not permitted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			if tc.setup != nil {
				err := tc.setup(fs)
				require.NoError(t, err)
			}

			if tc.name == "error creating index file" {
				fs = afero.NewReadOnlyFs(fs)
			}

			store, err := NewStore(fs)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, store)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, store)

			if tc.validate != nil {
				tc.validate(t, store)
			}
		})
	}
}

func TestStoreFetch(t *testing.T) {
	testCases := []struct {
		name        string
		index       map[string]Descriptor
		files       map[string][]byte
		uri         string
		expectedErr string
		expected    string
	}{
		{
			name: "fetch existing workflow",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "1234abcd",
				},
			},
			files: map[string][]byte{
				"1234abcd": []byte("hello world!"),
			},
			uri:      "https://example.com/workflow",
			expected: "hello world!",
		},
		{
			name: "fetch with query params - should ignore them",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "1234abcd",
				},
			},
			files: map[string][]byte{
				"1234abcd": []byte("hello world!"),
			},
			uri:      "https://example.com/workflow?param=value",
			expected: "hello world!",
		},
		{
			name:        "fetch non-existent workflow",
			index:       map[string]Descriptor{},
			uri:         "https://example.com/non-existent",
			expectedErr: "descriptor not found",
		},
		{
			name: "fetch with missing file",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "1234abcd",
				},
			},
			files:       map[string][]byte{},
			uri:         "https://example.com/workflow",
			expectedErr: "open 1234abcd: file does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			store := &Store{
				index: tc.index,
				fs:    fs,
			}

			for name, content := range tc.files {
				err := afero.WriteFile(fs, name, content, 0644)
				require.NoError(t, err)
			}

			uri, err := url.Parse(tc.uri)
			require.NoError(t, err)

			reader, err := store.Fetch(t.Context(), uri)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, reader)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, reader)

			content, err := io.ReadAll(reader)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(content))
		})
	}
}

func TestStoreStore(t *testing.T) {
	testCases := []struct {
		name         string
		initialIndex map[string]Descriptor
		uri          string
		content      string
		expectedErr  string
		validate     func(t *testing.T, s *Store, contentHex string)
	}{
		{
			name:         "store new workflow",
			initialIndex: map[string]Descriptor{},
			uri:          "https://example.com/workflow",
			content:      "hello world!",
			validate: func(t *testing.T, s *Store, contentHex string) {
				assert.Len(t, s.index, 1)
				desc, exists := s.index["https://example.com/workflow"]
				assert.True(t, exists)
				assert.Equal(t, int64(12), desc.Size)
				assert.Equal(t, contentHex, desc.Hex)

				content, err := afero.ReadFile(s.fs, contentHex)
				require.NoError(t, err)
				assert.Equal(t, "hello world!", string(content))

				indexContent, err := afero.ReadFile(s.fs, IndexFileName)
				require.NoError(t, err)
				assert.Contains(t, string(indexContent), contentHex)
				assert.Contains(t, string(indexContent), "https://example.com/workflow")
			},
		},
		{
			name:         "store workflow with query params - should ignore them",
			initialIndex: map[string]Descriptor{},
			uri:          "https://example.com/workflow?param=value",
			content:      "hello params!",
			validate: func(t *testing.T, s *Store, _ string) {
				assert.Len(t, s.index, 1)
				_, exists := s.index["https://example.com/workflow"]
				assert.True(t, exists)
			},
		},
		{
			name: "update existing workflow",
			initialIndex: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "old_hash",
				},
			},
			uri:     "https://example.com/workflow",
			content: "updated content",
			validate: func(t *testing.T, s *Store, contentHex string) {
				assert.Len(t, s.index, 1)
				desc := s.index["https://example.com/workflow"]
				assert.Equal(t, int64(15), desc.Size)
				assert.Equal(t, contentHex, desc.Hex)
				assert.NotEqual(t, "old_hash", desc.Hex)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			store := &Store{
				index: tc.initialIndex,
				fs:    fs,
			}

			err := afero.WriteFile(fs, IndexFileName, []byte("{}"), 0644)
			require.NoError(t, err)

			uri, err := url.Parse(tc.uri)
			require.NoError(t, err)

			rc := io.NopCloser(bytes.NewReader([]byte(tc.content)))
			err = store.Store(rc, uri)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)

			hasher := sha256.New()
			_, err = hasher.Write([]byte(tc.content))
			require.NoError(t, err)
			contentHash := fmt.Sprintf("%x", hasher.Sum(nil))

			if tc.validate != nil {
				tc.validate(t, store, contentHash)
			}
		})
	}
}

func TestStoreExists(t *testing.T) {
	testCases := []struct {
		name        string
		index       map[string]Descriptor
		files       map[string][]byte
		uri         string
		expected    bool
		expectedErr string
	}{
		{
			name: "workflow exists",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9",
				},
			},
			files: map[string][]byte{
				"7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9": []byte("hello world!"),
			},
			uri:      "https://example.com/workflow",
			expected: true,
		},
		{
			name: "workflow exists with query params - should ignore them",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9",
				},
			},
			files: map[string][]byte{
				"7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9": []byte("hello world!"),
			},
			uri:      "https://example.com/workflow?param=value",
			expected: true,
		},
		{
			name:     "workflow does not exist",
			index:    map[string]Descriptor{},
			uri:      "https://example.com/non-existent",
			expected: false,
		},
		{
			name: "descriptor exists but file missing",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "1234abcd",
				},
			},
			files:       map[string][]byte{},
			uri:         "https://example.com/workflow",
			expectedErr: "descriptor exists in index, but no corresponding file was found, possible cache corruption: 1234abcd",
		},
		{
			name: "size mismatch",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 20, // Wrong size
					Hex:  "1234abcd",
				},
			},
			files: map[string][]byte{
				"1234abcd": []byte("hello world!"), // Actual size is 12
			},
			uri:         "https://example.com/workflow",
			expectedErr: "size mismatch, expected 20, got 12",
		},
		{
			name: "hash mismatch",
			index: map[string]Descriptor{
				"https://example.com/workflow": {
					Size: 12,
					Hex:  "wrong_hash", // Wrong hash
				},
			},
			files: map[string][]byte{
				"wrong_hash": []byte("hello world!"),
			},
			uri:         "https://example.com/workflow",
			expectedErr: "hash mismatch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			store := &Store{
				index: tc.index,
				fs:    fs,
			}

			for name, content := range tc.files {
				err := afero.WriteFile(fs, name, content, 0644)
				require.NoError(t, err)
			}

			uri, err := url.Parse(tc.uri)
			require.NoError(t, err)

			exists, err := store.Exists(uri)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, exists)
		})
	}
}
