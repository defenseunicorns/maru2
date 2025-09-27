// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/ansi"
	"github.com/goccy/go-yaml"
	"github.com/muesli/termenv"
	"github.com/spf13/cast"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

// Environment variables used to determine what CI environment (if any) maru2 is in
//
// https://docs.github.com/en/actions/reference/workflows-and-actions/variables
//
// https://docs.gitlab.com/ci/variables/predefined_variables/
const (
	GitHubActionsEnvVar = "GITHUB_ACTIONS"
	GitLabCIEnvVar      = "GITLAB_CI"
)

var isGitHubActions = sync.OnceValue(func() bool {
	return os.Getenv(GitHubActionsEnvVar) == "true"
})

var isGitLabCI = sync.OnceValue(func() bool {
	return os.Getenv(GitLabCIEnvVar) == "true"
})

// The terminal colors maru2 uses, derived from
// https://github.com/charmbracelet/vhs/blob/main/themes.json
var (
	DebugColor = lipgloss.AdaptiveColor{
		Light: "#2e7de9", // tokyonight-day blue
		Dark:  "#7aa2f7", // tokyonight blue
	}
	InfoColor = lipgloss.AdaptiveColor{
		Light: "#007197", // tokyonight-day cyan
		Dark:  "#7dcfff", // tokyonight cyan
	}
	WarnColor = lipgloss.AdaptiveColor{
		Light: "#8c6c3e", // tokyonight-day amber/yellow
		Dark:  "#e0af68", // tokyonight amber/yellow
	}
	ErrorColor = lipgloss.AdaptiveColor{
		Light: "#f52a65", // tokyonight-day red
		Dark:  "#f7768e", // tokyonight red
	}
	FatalColor = lipgloss.AdaptiveColor{
		Light: "#9854f1", // tokyonight-day magenta (deep red alternative)
		Dark:  "#bb9af7", // tokyonight magenta (deep red alternative)
	}
	GreenColor = lipgloss.AdaptiveColor{
		Light: "#587539", // tokyonight-day green
		Dark:  "#9ece6a", // tokyonight green
	}
	GrayColor = lipgloss.AdaptiveColor{
		Light: "#c5c6bC",
		Dark:  "#3a3943",
	}
)

// TaskList is a prettier way to display a workflow's entrypoints from a CLI's perspective
type TaskList struct {
	col0max int
	rows    [][2]string
}

// NewDetailedTaskList renders a table detailing a workflow and all aliased workflows tasks
//
// The formatting is inspired by `just --list`
func NewDetailedTaskList(ctx context.Context, svc *uses.FetcherService, origin *url.URL, wf v1.Workflow) (*TaskList, error) {
	t := &TaskList{}
	for name, task := range wf.Tasks.OrderedSeq() {
		var comment string
		if desc := task.Description; desc != "" {
			comment = "# " + desc
		}

		msg := strings.Builder{}
		msg.WriteString(name)

		renderInputMap(&msg, task.Inputs)

		t.Row(msg.String(), comment)
	}

	for name, alias := range wf.Aliases.OrderedSeq() {
		if alias.Path != "" {
			next, err := uses.ResolveRelative(origin, strings.Join([]string{"file", alias.Path}, ":"), wf.Aliases)
			if err != nil {
				return nil, err
			}
			aliasedWF, err := Fetch(ctx, svc, next)
			if err != nil {
				return nil, err
			}
			for n, task := range aliasedWF.Tasks.OrderedSeq() {
				var comment string
				if desc := task.Description; desc != "" {
					comment = "# " + desc
				}

				msg := strings.Builder{}
				msg.WriteString((fmt.Sprintf("%s:%s", name, n)))

				renderInputMap(&msg, task.Inputs)

				t.Row(msg.String(), comment)
			}
		}
	}

	return t, nil
}

// Row appends a row to the list
func (tl *TaskList) Row(col0, col1 string) {
	tl.col0max = max(tl.col0max, ansi.StringWidth(col0))

	tl.rows = append(tl.rows, [2]string{col0, col1})
}

// String implements fmt.Stringers
func (tl *TaskList) String() string {
	sb := strings.Builder{}

	cutoff := 50

	for _, row := range tl.rows {
		col0, col1 := row[0], row[1]

		col0len := ansi.StringWidth(col0)
		text0 := lipgloss.NewStyle().MarginLeft(4).Render(col0)
		text1 := lipgloss.NewStyle().Foreground(InfoColor).Render(col1)

		sb.WriteString(text0)

		if col0len > cutoff {
			sb.WriteString(text1 + "\n")
		} else {
			numspaces := min(50-col0len, tl.col0max-col0len)
			sb.WriteString(strings.Repeat(" ", numspaces) + text1 + "\n")
		}
	}

	return sb.String()
}

func renderInputMap(w *strings.Builder, inputs v1.InputMap) {
	faint := lipgloss.NewStyle().Faint(true)
	blue := lipgloss.NewStyle().Foreground(DebugColor)
	amber := lipgloss.NewStyle().Foreground(WarnColor)
	green := lipgloss.NewStyle().Foreground(GreenColor)
	gray := lipgloss.NewStyle().Foreground(GrayColor)

	for n, input := range inputs.OrderedSeq() {
		w.WriteString(faint.Render(" -w "))
		if input.Default != nil {
			w.WriteString(blue.Render(n))
			w.WriteString("=")

			if input.DefaultFromEnv != "" {
				w.WriteString(green.Render(fmt.Sprintf(`"${%s:-%s}"`, input.DefaultFromEnv, cast.ToString(input.Default))))
			} else {
				w.WriteString(green.Render(fmt.Sprintf("'%s'", cast.ToString(input.Default))))
			}
			continue
		}

		if input.DefaultFromEnv != "" {
			w.WriteString(blue.Render(n))
			w.WriteString("=")
			w.WriteString(green.Render(fmt.Sprintf(`"$%s"`, input.DefaultFromEnv)))
			continue
		}

		if input.Required != nil && !*input.Required {
			w.WriteString(gray.Render(n))
			w.WriteString("=")
			continue
		}
		w.WriteString(amber.Render(n))
		w.WriteString("=")
	}
}

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

	gray := lipgloss.NewStyle().Background(GrayColor)

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
	if taskName == "" || wr == nil { // printing functions are best effort styled in order to not get in the way of true execution which should be catching these cases
		// no-op that prevents nil reference
		return func() {}
	}

	// https://docs.gitlab.com/ci/jobs/job_logs/#expand-and-collapse-job-log-sections
	if isGitLabCI() {
		if header == "" {
			header = taskName
		}
		_, _ = fmt.Fprintf(wr, `\e[0Ksection_start:%d:%s[collapsed=true]\r\e[0K%s`, time.Now().Unix(), taskName, header)
		_, _ = fmt.Fprintln(wr)
		return func() {
			_, _ = fmt.Fprintf(wr, `\e[0Ksection_end:%d:%s\r\e[0K`, time.Now().Unix(), taskName)
			_, _ = fmt.Fprintln(wr)
		}
	}

	// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#grouping-log-lines
	if isGitHubActions() {
		_, _ = fmt.Fprint(wr, "::group::")
		_, _ = fmt.Fprint(wr, taskName)
		if header != "" {
			_, _ = fmt.Fprintf(wr, ": %s", header)
		}
		_, _ = fmt.Fprintln(wr)
		return func() {
			_, _ = fmt.Fprintln(wr, `::endgroup::`)
		}
	}

	// no-op that prevents nil reference
	return func() {}
}
