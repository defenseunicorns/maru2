// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/defenseunicorns/maru2/cmd"
	"github.com/rogpeppe/go-internal/testscript"
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
		Dir: filepath.Join("..", "testdata"),
		Setup: func(env *testscript.Env) error {
			env.Setenv("NO_COLOR", "true")
			return nil
		},
	})
}
