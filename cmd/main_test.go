// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"os"
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
		"maru2-publish": func() {
			code := cmd.PublishMain()
			os.Exit(code)
		},
	})
}
