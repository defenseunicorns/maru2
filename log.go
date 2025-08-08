// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/goccy/go-yaml"
	"github.com/muesli/termenv"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

func printScript(logger *log.Logger, lang, script string) {
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

func printBuiltin(logger *log.Logger, builtin v0.With) {
	b, err := yaml.MarshalWithOptions(v0.Step{With: builtin}, yaml.Indent(2), yaml.IndentSequence(true))
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
		logger.Printf("%s", string(b))
		return
	}

	logger.Printf("%s", strings.TrimSpace(buf.String()))
}
