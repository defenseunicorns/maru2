// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: filepath.Join("..", "testdata"),
		Setup: func(env *testscript.Env) error {
			env.Setenv("NO_COLOR", "true")
			env.Setenv("HOME", filepath.Join(env.WorkDir, "home"))
			return nil
		},
		RequireUniqueNames: true,
		// UpdateScripts:      true,
	})
}
