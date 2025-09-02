// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/olareg/olareg"
	olaregcfg "github.com/olareg/olareg/config"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/require"
)

func TestPublishE2E(t *testing.T) {
	r := olareg.New(olaregcfg.Config{
		Storage: olaregcfg.ConfigStorage{
			StoreType: olaregcfg.StoreMem,
		},
	})
	s := httptest.NewServer(r)
	t.Cleanup(func() {
		s.Close()
		_ = r.Close()
	})

	serverURL, err := url.Parse(s.URL)
	require.NoError(t, err)

	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("..", "testdata", "publish"),
		Setup: func(env *testscript.Env) error {
			env.Setenv("NO_COLOR", "true")
			env.Setenv("REGISTRY", serverURL.Host)
			env.Setenv("HOME", filepath.Join(env.WorkDir, "home"))
			return nil
		},
		RequireUniqueNames: true,
		// UpdateScripts:      true,
	})
}
