// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPFetcher(t *testing.T) {
	ctx := log.WithContext(t.Context(), log.New(io.Discard))
	hw := `echo: [run: "Hello, World!"]`

	handler := func(w http.ResponseWriter, r *http.Request) {
		// handle /hello-world.yaml
		if r.URL.Path == "/hello-world.yaml" {
			_, _ = w.Write([]byte(hw))
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}
	s1 := httptest.NewTLSServer(http.HandlerFunc(handler))
	t.Cleanup(func() {
		s1.Close()
	})

	s2 := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(func() {
		s2.Close()
	})

	f := func(server *httptest.Server) {
		fetcher := NewHTTPFetcher(server.Client())

		rc, err := fetcher.Fetch(ctx, server.URL+"/hello-world.yaml")
		require.NoError(t, err)

		b, err := io.ReadAll(rc)
		require.NoError(t, err)

		assert.Equal(t, string(b), hw)

		rc, err = fetcher.Fetch(ctx, server.URL)
		require.EqualError(t, err, fmt.Sprintf("failed to fetch %s: 404 Not Found", server.URL))
		assert.Nil(t, rc)

		server.Close()
		rc, err = fetcher.Fetch(ctx, server.URL+"/hello-world.yaml")
		require.EqualError(t, err, fmt.Sprintf("Get \"%s/hello-world.yaml\": dial tcp %s: connect: connection refused", server.URL, server.Listener.Addr()))
		assert.Nil(t, rc)
	}

	f(s1)
	f(s2)
}
