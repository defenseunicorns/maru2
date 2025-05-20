// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/goccy/go-yaml"
	"github.com/muesli/termenv"
)

// very side effect heavy
// should rethink this
func printScript(ctx context.Context, prefix, script string) {
	logger := log.FromContext(ctx)
	script = strings.TrimSpace(script)

	if termenv.EnvNoColor() {
		for line := range strings.SplitSeq(script, "\n") {
			logger.Printf("%s %s", prefix, line)
		}
		return
	}

	var buf strings.Builder
	style := "tokyonight-day"
	if lipgloss.HasDarkBackground() {
		style = "tokyonight-moon"
	}
	lang := "shell"
	if prefix == ">" {
		lang = "go"
	}
	if err := quick.Highlight(&buf, script, lang, "terminal256", style); err != nil {
		logger.Debugf("failed to highlight: %v", err)
		for line := range strings.SplitSeq(script, "\n") {
			logger.Printf("%s %s", prefix, line)
		}
		return
	}

	for line := range strings.SplitSeq(buf.String(), "\n") {
		logger.Printf("%s %s", prefix, line)
	}
}

func printBuiltin(ctx context.Context, builtin With) error {
	logger := log.FromContext(ctx)

	b, err := yaml.MarshalWithOptions(Step{
		With: builtin,
	}, yaml.Indent(2), yaml.IndentSequence(true))
	if err != nil {
		return err
	}

	if termenv.EnvNoColor() {
		logger.Printf("%s", strings.TrimSpace(string(b)))
		return nil
	}

	style := "tokyonight-day"
	if lipgloss.HasDarkBackground() {
		style = "tokyonight-moon"
	}

	lang := "yaml"

	var buf strings.Builder

	if err := quick.Highlight(&buf, string(b), lang, "terminal256", style); err != nil {
		logger.Debugf("failed to highlight: %v", err)
		logger.Printf("%s", string(b))
		return err
	}

	logger.Printf("%s", strings.TrimSpace(buf.String()))

	return nil
}
