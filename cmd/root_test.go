// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/assert"

	"github.com/defenseunicorns/maru2/cmd"
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

func TestParseExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: 0,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: 1,
		},
		{
			name:     "command exit code 0",
			err:      exec.Command("true").Run(),
			expected: 0,
		},
		{
			name:     "command exit code 1",
			err:      exec.Command("false").Run(),
			expected: 1,
		},
		{
			name:     "command exit code 42",
			err:      exec.Command("sh", "-c", "exit 42").Run(),
			expected: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := cmd.ParseExitCode(tt.err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
