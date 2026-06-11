package theme

import (
	"image/color"
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/term"
)

type Theme struct {
	Debug color.Color
	Info  color.Color
	Warn  color.Color
	Error color.Color
	Fatal color.Color
	Green color.Color
	Gray  color.Color

	ColorProfile colorprofile.Profile
	ChromaStyle  string
}

// DefaultTheme returns the default theme with tokyonight colors.
func DefaultTheme(in, out io.Writer) *Theme {
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

	return &Theme{
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
}

// DefaultStyles returns the default log styles using the default theme.
func DefaultStyles(in, out io.Writer) *log.Styles {
	t := DefaultTheme(in, out)
	styles := log.DefaultStyles()

	styles.Levels[log.DebugLevel] = styles.Levels[log.DebugLevel].Foreground(t.Debug)
	styles.Levels[log.InfoLevel] = styles.Levels[log.InfoLevel].Foreground(t.Info)
	styles.Levels[log.WarnLevel] = styles.Levels[log.WarnLevel].Foreground(t.Warn)
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].Foreground(t.Error)
	styles.Levels[log.FatalLevel] = styles.Levels[log.FatalLevel].Foreground(t.Fatal)

	return styles
}

func BlankTheme() *Theme {
	return &Theme{
		Debug:        lipgloss.Color("#2e7de9"), // tokyonight-day blue
		Info:         lipgloss.Color("#007197"), // tokyonight-day cyan
		Warn:         lipgloss.Color("#8c6c3e"), // tokyonight-day amber/yellow
		Error:        lipgloss.Color("#f52a65"), // tokyonight-day red
		Fatal:        lipgloss.Color("#9854f1"), // tokyonight-day magenta (deep red alternative)
		Green:        lipgloss.Color("#587539"), // tokyonight-day green
		Gray:         lipgloss.Color("#c5c6bC"),
		ColorProfile: colorprofile.ASCII, // disable color despite setting colors up above
		ChromaStyle:  "tokyonight-day",
	}
}
