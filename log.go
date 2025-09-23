// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/goccy/go-yaml"
	"github.com/muesli/termenv"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

const (
	GITHUB_ACTIONS_ENV_VAR = "GITHUB_ACTIONS"
	GITLAB_CI_ENV_VAR      = "GITLAB_CI"
)

var isGitHubActions = sync.OnceValue(func() bool {
	return os.Getenv(GITHUB_ACTIONS_ENV_VAR) == "true"
})

var isGitLabCI = sync.OnceValue(func() bool {
	return os.Getenv(GITLAB_CI_ENV_VAR) == "true"
})

// printScript renders shell script content with syntax highlighting
//
// Uses chroma for syntax highlighting with adaptive color schemes (light/dark theme support)
// Falls back to plain text output when NO_COLOR is set or highlighting fails
func printScript(logger *log.Logger, lang, script string) {
	if logger.GetLevel() > log.InfoLevel {
		return
	}

	script = strings.TrimSpace(script)

	if termenv.EnvNoColor() {
		// this is essentially the same behavior/rendering as make
		logger.Print(script)
		return
	}

	if lang == "" {
		lang = "shell"
	}

	var buf strings.Builder
	style := "tokyonight-day"
	if lipgloss.HasDarkBackground() {
		style = "tokyonight-moon"
	}
	if err := quick.Highlight(&buf, script, lang, "terminal256", style); err != nil {
		logger.Debugf("failed to highlight: %v", err)
		for line := range strings.SplitSeq(script, "\n") {
			logger.Printf("  %s", line)
		}
		return
	}

	color := lipgloss.AdaptiveColor{
		Light: "#c5c6bC",
		Dark:  "#3a3943",
	}
	gray := lipgloss.NewStyle().Background(color)

	prefix := gray.Render(" ")

	for line := range strings.SplitSeq(buf.String(), "\n") {
		logger.Printf("%s %s", prefix, line)
	}
}

// printBuiltin renders builtin task configuration with syntax highlighting
//
// Marshals the builtin With map as YAML and applies syntax highlighting for better readability
// Used in dry-run mode to preview builtin task execution without running commands
func printBuiltin(logger *log.Logger, builtin schema.With) {
	if logger.GetLevel() > log.InfoLevel {
		return
	}

	b, err := yaml.MarshalWithOptions(v1.Step{With: builtin}, yaml.Indent(2), yaml.IndentSequence(true))
	if err != nil {
		logger.Debugf("failed to marshal builtin: %v", err)
		return
	}

	if termenv.EnvNoColor() {
		logger.Printf("%s", strings.TrimSpace(string(b)))
		return
	}

	style := "tokyonight-day"
	if lipgloss.HasDarkBackground() {
		style = "tokyonight-moon"
	}

	var buf strings.Builder

	if err := quick.Highlight(&buf, string(b), "yaml", "terminal256", style); err != nil {
		logger.Debugf("failed to highlight: %v", err)
		logger.Printf("%s", strings.TrimSpace(string(b)))
		return
	}

	logger.Printf("%s", strings.TrimSpace(buf.String()))
}

func printGroup(wr io.Writer, taskName string, header string) func() {
	isGitHub := isGitHubActions()
	isGitLab := isGitLabCI()

	// https://docs.gitlab.com/ci/jobs/job_logs/#expand-and-collapse-job-log-sections
	if isGitLab {
		if header == "" {
			header = taskName
		}
		_, _ = fmt.Fprintf(wr, `\e[0Ksection_start:%d:%s[collapsed=true]\r\e[0K%s`, time.Now().Unix(), taskName, header)
		return func() {
			_, _ = fmt.Fprintf(wr, `\e[0Ksection_end:%d:%s\r\e[0K`, time.Now().Unix(), taskName)
		}
	}

	// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#grouping-log-lines
	if isGitHub {
		_, _ = fmt.Fprintf(wr, `::group::%s: %s\n`, taskName, header)
		return func() {
			_, _ = fmt.Fprintln(wr, `::endgroup::`)
		}
	}

	// no-op that prevents nil reference
	return func() {}
}
