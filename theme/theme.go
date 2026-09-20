// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package theme controls terminal coloring and styles in maru2
package theme

import (
	"context"
	"image/color"
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

// Theme controls (almost) all coloring and styles
//
// still some one-offs that I'm figuring out how to best handle downsampling/NO_COLOR
type Theme struct {
	Debug     color.Color
	Info      color.Color
	Warn      color.Color
	Error     color.Color
	Fatal     color.Color
	Green     color.Color
	Gray      color.Color
	LogStyles *log.Styles

	ColorProfile colorprofile.Profile
	ChromaStyle  string
}

var blankColor = lipgloss.Color("")

// DefaultTheme returns the default theme with tokyonight colors.
func DefaultTheme(in, out io.Writer) Theme {
	hasDarkBG := true // follow lipgloss default behavior
	inFile, inIsFile := in.(term.File)
	outFile, outIsFile := out.(term.File)

	if inIsFile && outIsFile {
		hasDarkBG = lipgloss.HasDarkBackground(inFile, outFile)
	}

	lightDark := func(light, dark color.Color) color.Color {
		return lipgloss.LightDark(hasDarkBG)(light, dark)
	}

	chromaStyle := "tokyonight-day"
	if hasDarkBG {
		chromaStyle = "tokyonight-moon"
	}

	theme := Theme{
		Debug: lightDark(
			lipgloss.Color("#2e7de9"), // tokyonight-day blue
			lipgloss.Color("#7aa2f7"), // tokyonight blue
		),
		Info: lightDark(
			lipgloss.Color("#007197"), // tokyonight-day cyan
			lipgloss.Color("#7dcfff"), // tokyonight cyan
		),
		Warn: lightDark(
			lipgloss.Color("#8c6c3e"), // tokyonight-day amber/yellow
			lipgloss.Color("#e0af68"), // tokyonight amber/yellow
		),
		Error: lightDark(
			lipgloss.Color("#f52a65"), // tokyonight-day red
			lipgloss.Color("#f7768e"), // tokyonight red
		),
		Fatal: lightDark(
			lipgloss.Color("#9854f1"), // tokyonight-day magenta (deep red alternative)
			lipgloss.Color("#bb9af7"), // tokyonight magenta (deep red alternative)
		),
		Green: lightDark(
			lipgloss.Color("#587539"), // tokyonight-day green
			lipgloss.Color("#9ece6a"), // tokyonight green
		),
		Gray: lightDark(
			lipgloss.Color("#c5c6bC"),
			lipgloss.Color("#3a3943"),
		),
		ColorProfile: colorprofile.Detect(out, os.Environ()),
		ChromaStyle:  chromaStyle,
	}

	styles := log.DefaultStyles()

	styles.Levels[log.DebugLevel] = styles.Levels[log.DebugLevel].Foreground(theme.Debug)
	styles.Levels[log.InfoLevel] = styles.Levels[log.InfoLevel].Foreground(theme.Info)
	styles.Levels[log.WarnLevel] = styles.Levels[log.WarnLevel].Foreground(theme.Warn)
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].Foreground(theme.Error)
	styles.Levels[log.FatalLevel] = styles.Levels[log.FatalLevel].Foreground(theme.Fatal)

	theme.LogStyles = styles

	return theme
}

// BlankTheme is a minimal theme w/ blank colors and default styles
//
// Mainly used for tests
func BlankTheme() Theme {
	return Theme{
		Debug:        blankColor,
		Info:         blankColor,
		Warn:         blankColor,
		Error:        blankColor,
		Fatal:        blankColor,
		Green:        blankColor,
		Gray:         blankColor,
		LogStyles:    log.DefaultStyles(),
		ColorProfile: colorprofile.NoTTY,
	}
}

// NewStringWriter is similar in function to colorprofile.NewWriter
func (t Theme) NewStringWriter(wr io.Writer) io.StringWriter {
	return &colorprofile.Writer{Forward: wr, Profile: t.ColorProfile}
}

type contextKey struct{}

// WithContext returns a new context with the given theme embedded.
func WithContext(ctx context.Context, t Theme) context.Context {
	return context.WithValue(ctx, contextKey{}, t)
}

// FromContext returns the Theme stored in ctx, or BlankTheme() if none is set.
func FromContext(ctx context.Context) Theme {
	if t, ok := ctx.Value(contextKey{}).(Theme); ok {
		return t
	}
	return BlankTheme()
}
