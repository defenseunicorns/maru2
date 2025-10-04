// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd_test

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/rogpeppe/go-internal/testscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		UpdateScripts:      os.Getenv("UPDATE_SCRIPTS") == "true",
	})
}

func TestIsTerminal(t *testing.T) {
	assert.True(t, cmd.IsTerminal(os.Stdout))
	assert.True(t, cmd.IsTerminal(os.Stderr))
	tmp := t.TempDir()
	f, err := os.CreateTemp(tmp, "")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = f.Close()
	})
	assert.False(t, cmd.IsTerminal(f))
	assert.False(t, cmd.IsTerminal(nil))
	assert.False(t, cmd.IsTerminal(&strings.Builder{}))
}

func TestExplainRedirect(t *testing.T) {
	tmp := t.TempDir()

	tasksYamlPath := filepath.Join(tmp, "tasks.yaml")
	content := `schema-version: v1

aliases:
  local:
    path: ./local.yaml
  remote:
    type: github
    base-url: https://api.github.com
    token-from-env: GITHUB_TOKEN

tasks:
  default:
    description: Default task with all features
    collapse: true
    inputs:
      required-param:
        description: A required parameter
        required: true
      optional-param:
        description: An optional parameter
        required: false
        default: "default-value"
      env-param:
        description: Parameter with env default
        required: false
        default-from-env: HOME
      validated-param:
        description: Parameter with validation
        required: false
        validate: "^[a-z]+$"
      deprecated-param:
        description: Old parameter
        required: false
        deprecated-message: Use required-param instead
    steps:
      - uses: echo
      - uses: builtin:fetch
        with:
          url: https://example.com

  echo:
    steps:
      - run: echo "hello"
`
	err := os.WriteFile(tasksYamlPath, []byte(content), 0o755)
	require.NoError(t, err)

	root := cmd.NewRootCmd()
	root.SetArgs([]string{"--from", tasksYamlPath, "--explain"})
	// temporarily always say yes
	curr := cmd.IsTerminal
	t.Cleanup(func() {
		cmd.IsTerminal = curr
	})
	cmd.IsTerminal = func(io.Writer) bool {
		return true
	}

	sb := strings.Builder{}
	root.SetOut(&sb)
	err = root.Execute()
	require.NoError(t, err)

	expected := []string{
		"",
		"                                                                                                  ",
		"  │ for schema version v1                                                                         ",
		"  │                                                                                               ",
		"  │ https://raw.githubusercontent.com/defenseunicorns/maru2/main/schema/v1/schema.json            ",
		"                                                                                                  ",
		"  ## Aliases                                                                                      ",
		"                                                                                                  ",
		"  Shortcuts for referencing remote repositories and local files:                                  ",
		"                                                                                                  ",
		"   Name             │ Type             │ Details                                                  ",
		"  ──────────────────┼──────────────────┼────────────────────────────────────────────────────────  ",
		"   local            │ Local File       │ ./local.yaml                                             ",
		"   remote           │ Package URL      │ github at https://api.github.com (auth: $GITHUB_TOKEN)   ",
		"                                                                                                  ",
		"  ## Tasks                                                                                        ",
		"                                                                                                  ",
		"  ### default (Default Task)                                                                      ",
		"                                                                                                  ",
		"  Default task with all features                                                                  ",
		"                                                                                                  ",
		"  Output will be grouped in CI environments (GitHub Actions, GitLab CI)                           ",
		"                                                                                                  ",
		"  Input Parameters:                                                                               ",
		"                                                                                                  ",
		"   Name             │ Description                │ Required │ Default   │ Validati… │ Notes       ",
		"  ──────────────────┼────────────────────────────┼──────────┼───────────┼───────────┼───────────  ",
		"   deprecated-param │ Old parameter              │ No       │ -         │ -         │ ⚠️          ",
		"                    │                            │          │           │           │ Deprecate   ",
		"                    │                            │          │           │           │ d: Use      ",
		"                    │                            │          │           │           │ required-   ",
		"                    │                            │          │           │           │ param       ",
		"                    │                            │          │           │           │ instead     ",
		"   env-param        │ Parameter with env default │ No       │ $HOME     │ -         │ -           ",
		"   optional-param   │ An optional parameter      │ No       │ default-  │ -         │ -           ",
		"                    │                            │          │ value     │           │             ",
		"   required-param   │ A required parameter       │ Yes      │ -         │ -         │ -           ",
		"   validated-param  │ Parameter with validation  │ No       │ -         │ ^[a-z]+$  │ -           ",
		"                                                                                                  ",
		"  Uses:                                                                                           ",
		"                                                                                                  ",
		"  • echo                                                                                          ",
		"  • builtin:fetch                                                                                 ",
		"                                                                                                  ",
		"  ### echo                                                                                        ",
		"",
		"",
		"",
	}

	require.Equal(t, strings.Join(expected, "\n"), ansi.Strip(sb.String()))
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
