// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses_test

import (
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/olareg/olareg"
	olaregcfg "github.com/olareg/olareg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/config"
	"github.com/defenseunicorns/maru2/uses"
)

func TestOCIClient(t *testing.T) {
	r1 := olareg.New(olaregcfg.Config{
		Storage: olaregcfg.ConfigStorage{
			StoreType: olaregcfg.StoreMem,
			RootDir:   "./testdata", // serve content from testdata, writes only apply to memory
		},
	})
	s1 := httptest.NewServer(r1)
	t.Cleanup(func() {
		s1.Close()
		_ = r1.Close()
	})

	r2 := olareg.New(olaregcfg.Config{
		Storage: olaregcfg.ConfigStorage{
			StoreType: olaregcfg.StoreMem,
			RootDir:   "./testdata", // serve content from testdata, writes only apply to memory
		},
	})
	s2 := httptest.NewTLSServer(r2)
	t.Cleanup(func() {
		s2.Close()
		_ = r2.Close()
	})

	// not testing context cancellation at this time
	ctx := log.WithContext(t.Context(), log.New(io.Discard))

	seed := func(server *httptest.Server) {
		tmp := t.TempDir()
		t.Chdir(tmp)
		err := os.WriteFile(uses.DefaultFileName, []byte(`inputs:
  text:
    description: Text to echo
    default: "Hello, world!"
    required: true

tasks:
  echo:
    - run: echo "${{ input "text" }}"`), 0700)
		require.NoError(t, err)

		serverURL, err := url.Parse(server.URL)
		require.NoError(t, err)
		registry := serverURL.Host
		isPlainHTTP := serverURL.Scheme == "http"

		dst, err := remote.NewRepository(fmt.Sprintf("%s/workflow-1:latest", registry))
		require.NoError(t, err)
		dst.PlainHTTP = isPlainHTTP
		dst.Client = &auth.Client{
			Client: server.Client(),
		}

		err = maru2.Publish(ctx, &config.Config{}, dst, []string{uses.DefaultFileName})
		require.NoError(t, err)
	}

	f := func(server *httptest.Server) {
		serverURL, err := url.Parse(server.URL)
		require.NoError(t, err)
		registry := serverURL.Host
		isPlainHTTP := serverURL.Scheme == "http"
		httpClient := server.Client()

		// not testing insecureskiptls yet?
		client, err := uses.NewOCIClient(httpClient, false, isPlainHTTP)
		require.NoError(t, err)

		uri, err := url.Parse(fmt.Sprintf("oci:%s/workflow-1:latest", registry))
		require.NoError(t, err)

		rc, err := client.Fetch(ctx, uri)
		require.NoError(t, err)

		tru := true
		wf, err := maru2.Read(rc)
		require.NoError(t, err)
		assert.Equal(t, maru2.Workflow{
			Inputs: maru2.InputMap{"text": maru2.InputParameter{
				Description: "Text to echo",
				Default:     "Hello, world!",
				Required:    &tru,
			}},
			Tasks: maru2.TaskMap{"echo": maru2.Task{{
				Run: `echo "${{ input "text" }}"`,
			}}},
			Aliases: map[string]config.Alias{},
		}, wf)
	}
	seed(s1)
	f(s1)
	seed(s2)
	f(s2)
}
