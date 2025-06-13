// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalStore(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(fs afero.Fs) error
		expectedErr string
		validate    func(t *testing.T, s *LocalStore)
	}{
		{
			name: "new store without existing index",
			validate: func(t *testing.T, s *LocalStore) {
				assert.NotNil(t, s.index)
				assert.Empty(t, s.index)

				content, err := afero.ReadFile(s.fs, IndexFileName)
				require.NoError(t, err)
				assert.Empty(t, string(content))
			},
		},
		{
			name: "new store with existing valid index",
			setup: func(fs afero.Fs) error {
				return afero.WriteFile(fs, IndexFileName, []byte(`https://example.com h1:7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9 10`), 0644)
			},
			validate: func(t *testing.T, s *LocalStore) {
				assert.NotNil(t, s.index)
				assert.Len(t, s.index, 1)
				assert.Contains(t, s.index, "https://example.com")
				assert.Equal(t, int64(10), s.index["https://example.com"].Size)
				assert.Equal(t, "7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9", s.index["https://example.com"].Hex)
			},
		},
		{
			name: "new store with existing invalid index",
			setup: func(fs afero.Fs) error {
				return afero.WriteFile(fs, IndexFileName, []byte(`invalid txt`), 0644)
			},
			expectedErr: "invalid line format",
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

			store, err := NewLocalStore(fs)

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

func TestLocalStoreFetch(t *testing.T) {
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

			store := &LocalStore{
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

func TestLocalStoreStore(t *testing.T) {
	testCases := []struct {
		name         string
		initialIndex map[string]Descriptor
		uri          string
		content      string
		expectedErr  string
		validate     func(t *testing.T, s *LocalStore, contentHex string)
	}{
		{
			name:         "store new workflow",
			initialIndex: map[string]Descriptor{},
			uri:          "https://example.com/workflow",
			content:      "hello world!",
			validate: func(t *testing.T, s *LocalStore, contentHex string) {
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
			validate: func(t *testing.T, s *LocalStore, _ string) {
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
			validate: func(t *testing.T, s *LocalStore, contentHex string) {
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

			store := &LocalStore{
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
			contentHash := hex.EncodeToString(hasher.Sum(nil))

			if tc.validate != nil {
				tc.validate(t, store, contentHash)
			}
		})
	}
}

func TestLocalStoreExists(t *testing.T) {
	testCases := []struct {
		name        string
		index       map[string]Descriptor
		files       map[string]string
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
			files: map[string]string{
				"7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9": "hello world!",
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
			files: map[string]string{
				"7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9": "hello world!",
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
			files:       map[string]string{},
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
			files: map[string]string{
				"1234abcd": "hello world!", // Actual size is 12
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
			files: map[string]string{
				"wrong_hash": "hello world!",
			},
			uri:         "https://example.com/workflow",
			expectedErr: "hash mismatch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()

			store := &LocalStore{
				index: tc.index,
				fs:    fs,
			}

			for name, content := range tc.files {
				err := afero.WriteFile(fs, name, []byte(content), 0644)
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

func TestLocalStoreGC(t *testing.T) {
	fs := afero.NewMemMapFs()
	store, err := NewLocalStore(fs)
	require.NoError(t, err)

	err = store.Store(strings.NewReader("hello world!"), &url.URL{Scheme: "https", Host: "example.com", Path: "/workflow"})
	require.NoError(t, err)

	assert.Len(t, store.index, 1)
	wf1 := store.index["https://example.com/workflow"].Hex
	require.NotEmpty(t, wf1)

	indexContent, err := afero.ReadFile(fs, IndexFileName)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/workflow h1:7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9 12\n", string(indexContent))

	err = store.GC()
	require.NoError(t, err)

	_, err = fs.Stat(wf1)
	require.NoError(t, err)

	indexContentAfterGC, err := afero.ReadFile(fs, IndexFileName)
	require.NoError(t, err)
	assert.Equal(t, indexContent, indexContentAfterGC)

	unusedFile := "unused123"
	err = afero.WriteFile(fs, unusedFile, []byte("unused content"), 0644)
	require.NoError(t, err)

	_, err = fs.Stat(unusedFile)
	require.NoError(t, err)

	err = store.GC()
	require.NoError(t, err)

	_, err = fs.Stat(wf1)
	require.NoError(t, err)

	_, err = fs.Stat(unusedFile)
	require.ErrorIs(t, err, os.ErrNotExist)

	indexContentAfterRemoval, err := afero.ReadFile(fs, IndexFileName)
	require.NoError(t, err)
	assert.Equal(t, string(indexContent), string(indexContentAfterRemoval))

	_, err = fs.Stat(IndexFileName)
	require.NoError(t, err)

	err = fs.Mkdir("testdir", 0755)
	require.NoError(t, err)

	err = store.GC()
	require.NoError(t, err)

	fi, err := fs.Stat("testdir")
	require.NoError(t, err)
	require.True(t, fi.IsDir())

	err = store.Store(strings.NewReader("new content"), &url.URL{Scheme: "https", Host: "example.com", Path: "/new-workflow"})
	require.NoError(t, err)

	wf2 := store.index["https://example.com/new-workflow"].Hex
	require.NotEmpty(t, wf2)

	updatedIndexContent, err := afero.ReadFile(fs, IndexFileName)
	require.NoError(t, err)
	assert.Equal(t, `https://example.com/new-workflow h1:fe32608c9ef5b6cf7e3f946480253ff76f24f4ec0678f3d0f07f9844cbff9601 11
https://example.com/workflow h1:7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9 12
`, string(updatedIndexContent))

	err = store.GC()
	require.NoError(t, err)

	_, err = fs.Stat(wf1)
	require.NoError(t, err)

	_, err = fs.Stat(wf2)
	require.NoError(t, err)

	err = store.Store(strings.NewReader("more"), &url.URL{Scheme: "https", Host: "example.com", Path: "/workflow"})
	require.NoError(t, err)

	err = store.GC()
	require.NoError(t, err)

	assert.Len(t, store.index, 2)

	updatedIndexContent, err = afero.ReadFile(fs, IndexFileName)
	require.NoError(t, err)
	assert.Equal(t, `https://example.com/new-workflow h1:fe32608c9ef5b6cf7e3f946480253ff76f24f4ec0678f3d0f07f9844cbff9601 11
https://example.com/workflow h1:187897ce0afcf20b50ba2b37dca84a951b7046f29ed5ab94f010619f69d6e189 4
`, string(updatedIndexContent))

	_, err = fs.Stat(wf1)
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = fs.Stat(wf2)
	require.NoError(t, err)
}
