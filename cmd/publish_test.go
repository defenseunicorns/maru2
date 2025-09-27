// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olareg/olareg"
	olaregcfg "github.com/olareg/olareg/config"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/defenseunicorns/maru2/cmd"
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
		UpdateScripts:      os.Getenv("UPDATE_SCRIPTS") == "true",
	})
}

func TestEmbeddedPublishVersion(t *testing.T) {
	embed := &cobra.Command{Use: "test-embed"}
	embed.AddCommand(cmd.NewPublishCmd())
	sb := strings.Builder{}
	embed.SetOut(&sb)
	embed.SetArgs([]string{"maru2-publish", "--version"})
	require.NoError(t, embed.Execute())
	assert.Equal(t, "(devel)\n", sb.String())
}
