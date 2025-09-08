// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package keycloak_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/defenseunicorns/maru2/cmd"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"maru2": func() {
			code := cmd.Main()
			os.Exit(code)
		},
	})
}

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Setup: func(env *testscript.Env) error {
			env.Setenv("NO_COLOR", "true")
			env.Setenv("HOME", filepath.Join(env.WorkDir, "home"))
			return nil
		},
		RequireUniqueNames: true,
		// UpdateScripts:      true,
	})
}
