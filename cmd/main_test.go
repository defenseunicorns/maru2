// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"fmt"
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
		"envsubst": func() {
			envsubst()
		},
	})
}

// envsubst replicates similar functionality to https://man7.org/linux/man-pages/man1/envsubst.1.html
// but instead operates on file paths and edits in-place so tests don't need to rename files
func envsubst() {
	paths := os.Args[1:]

	fatal := func(err error) {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fatal(err)
		}
		out := os.ExpandEnv(string(data))
		err = os.WriteFile(path, []byte(out), 0644)
		if err != nil {
			fatal(err)
		}
	}
	os.Exit(0)
}
