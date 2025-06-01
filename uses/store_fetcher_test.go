// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/defenseunicorns/maru2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFetcher implements the Fetcher interface for testing
type mockFetcher struct {
	fetchFunc  func(ctx context.Context, uri *url.URL) (io.ReadCloser, error)
	fetchCalls int
}

func (m *mockFetcher) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	m.fetchCalls++
	if m.fetchFunc == nil {
		return nil, fmt.Errorf("fetchFunc not implemented")
	}
	return m.fetchFunc(ctx, uri)
}

// mockStorage implements the Storage interface for testing
type mockStorage struct {
	fetchFunc   func(ctx context.Context, uri *url.URL) (io.ReadCloser, error)
	existsFunc  func(uri *url.URL) (bool, error)
	storeFunc   func(r io.Reader, uri *url.URL) error
	fetchCalls  int
	existsCalls int
	storeCalls  int
}

func (m *mockStorage) Fetch(ctx context.Context, uri *url.URL) (io.ReadCloser, error) {
	m.fetchCalls++
	if m.fetchFunc == nil {
		return nil, fmt.Errorf("fetchFunc not implemented")
	}
	return m.fetchFunc(ctx, uri)
}

func (m *mockStorage) Exists(uri *url.URL) (bool, error) {
	m.existsCalls++
	if m.existsFunc == nil {
		return false, fmt.Errorf("existsFunc not implemented")
	}
	return m.existsFunc(uri)
}

func (m *mockStorage) Store(r io.Reader, uri *url.URL) error {
	m.storeCalls++
	if m.storeFunc == nil {
		return fmt.Errorf("storeFunc not implemented")
	}
	return m.storeFunc(r, uri)
}

func TestStoreFetcher(t *testing.T) {
	testCases := []struct {
		name            string
		policy          config.FetchPolicy
		setup           func(source *mockFetcher, store *mockStorage)
		uri             string
		expected        string
		expectedErr     string
		verifyCallCount func(t *testing.T, source *mockFetcher, store *mockStorage)
	}{
		{
			name:   "FetchPolicyNever: always fetch from store",
			policy: config.FetchPolicyNever,
			setup: func(_ *mockFetcher, store *mockStorage) {
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from store")), nil
				}
			},
			uri:      "https://example.com/workflow",
			expected: "from store",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 0, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 0, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyNever: store fetch error",
			policy: config.FetchPolicyNever,
			setup: func(_ *mockFetcher, store *mockStorage) {
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return nil, errors.New("store fetch error")
				}
			},
			uri:         "https://example.com/workflow",
			expectedErr: "store fetch error",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 0, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 0, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyIfNotPresent: exists in store",
			policy: config.FetchPolicyIfNotPresent,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.existsFunc = func(_ *url.URL) (bool, error) {
					return true, nil
				}
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from store")), nil
				}
			},
			uri:      "https://example.com/workflow",
			expected: "from store",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 0, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 1, store.existsCalls)
				assert.Equal(t, 0, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyIfNotPresent: store exists check error",
			policy: config.FetchPolicyIfNotPresent,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.existsFunc = func(_ *url.URL) (bool, error) {
					return false, errors.New("exists error")
				}
				store.storeFunc = func(_ io.Reader, _ *url.URL) error {
					return nil
				}
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from store after fetch")), nil
				}
			},
			uri:      "https://example.com/workflow",
			expected: "from store after fetch",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 1, store.existsCalls)
				assert.Equal(t, 1, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyIfNotPresent: exists but fetch from store fails",
			policy: config.FetchPolicyIfNotPresent,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.existsFunc = func(_ *url.URL) (bool, error) {
					return true, nil
				}
				// Store fetch implementation is done via counter in the test setup
				store.storeFunc = func(_ io.Reader, _ *url.URL) error {
					return nil
				}
			},
			uri:      "https://example.com/workflow",
			expected: "from store after fetch",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 2, store.fetchCalls)
				assert.Equal(t, 1, store.existsCalls)
				assert.Equal(t, 1, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyIfNotPresent: not in store",
			policy: config.FetchPolicyIfNotPresent,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.existsFunc = func(_ *url.URL) (bool, error) {
					return false, nil
				}
				store.storeFunc = func(_ io.Reader, _ *url.URL) error {
					return nil
				}
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from store after fetch")), nil
				}
			},
			uri:      "https://example.com/workflow",
			expected: "from store after fetch",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 1, store.existsCalls)
				assert.Equal(t, 1, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyAlways: always fetch from source and update store",
			policy: config.FetchPolicyAlways,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.storeFunc = func(_ io.Reader, _ *url.URL) error {
					return nil
				}
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from store after fetch")), nil
				}
			},
			uri:      "https://example.com/workflow",
			expected: "from store after fetch",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 1, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyAlways: source fetch error",
			policy: config.FetchPolicyAlways,
			setup: func(source *mockFetcher, _ *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return nil, errors.New("source fetch error")
				}
			},
			uri:         "https://example.com/workflow",
			expectedErr: "source fetch error",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 0, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 0, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyAlways: store error",
			policy: config.FetchPolicyAlways,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.storeFunc = func(_ io.Reader, _ *url.URL) error {
					return errors.New("store error")
				}
			},
			uri:         "https://example.com/workflow",
			expectedErr: "store error",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 0, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 1, store.storeCalls)
			},
		},
		{
			name:   "FetchPolicyAlways: store fetch error after store",
			policy: config.FetchPolicyAlways,
			setup: func(source *mockFetcher, store *mockStorage) {
				source.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return io.NopCloser(strings.NewReader("from source")), nil
				}
				store.storeFunc = func(_ io.Reader, _ *url.URL) error {
					return nil
				}
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					return nil, errors.New("store fetch error")
				}
			},
			uri:         "https://example.com/workflow",
			expectedErr: "store fetch error",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 1, source.fetchCalls)
				assert.Equal(t, 1, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 1, store.storeCalls)
			},
		},
		{
			name:        "unsupported fetch policy",
			policy:      "invalid",
			uri:         "https://example.com/workflow",
			expectedErr: "unsupported fetch policy: invalid",
			verifyCallCount: func(t *testing.T, source *mockFetcher, store *mockStorage) {
				assert.Equal(t, 0, source.fetchCalls)
				assert.Equal(t, 0, store.fetchCalls)
				assert.Equal(t, 0, store.existsCalls)
				assert.Equal(t, 0, store.storeCalls)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := &mockFetcher{}
			store := &mockStorage{}

			if tc.setup != nil {
				tc.setup(source, store)
			}

			// For tests involving fetch from store fails then succeeds
			if tc.name == "FetchPolicyIfNotPresent: exists but fetch from store fails" {
				var fetchCount int
				store.fetchFunc = func(_ context.Context, _ *url.URL) (io.ReadCloser, error) {
					fetchCount++
					if fetchCount == 1 {
						return nil, errors.New("store fetch error")
					}
					return io.NopCloser(strings.NewReader("from store after fetch")), nil
				}
			}
			fetcher := &StoreFetcher{
				Source: source,
				Store:  store,
				Policy: tc.policy,
			}

			uri, err := url.Parse(tc.uri)
			require.NoError(t, err)

			rc, err := fetcher.Fetch(t.Context(), uri)

			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
				assert.Nil(t, rc)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, rc)

			content, err := io.ReadAll(rc)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(content))

			if tc.verifyCallCount != nil {
				tc.verifyCallCount(t, source, store)
			}
		})
	}
}
