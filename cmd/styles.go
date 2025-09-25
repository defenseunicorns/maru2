// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var (
	// https://github.com/charmbracelet/vhs/blob/main/themes.json
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
)

// DefaultStyles returns the default styles.
func DefaultStyles() *log.Styles {
	styles := log.DefaultStyles()

	styles.Levels[log.DebugLevel] = styles.Levels[log.DebugLevel].Foreground(DebugColor)
	styles.Levels[log.InfoLevel] = styles.Levels[log.InfoLevel].Foreground(InfoColor)
	styles.Levels[log.WarnLevel] = styles.Levels[log.WarnLevel].Foreground(WarnColor)
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].Foreground(ErrorColor)
	styles.Levels[log.FatalLevel] = styles.Levels[log.FatalLevel].Foreground(FatalColor)

	return styles
}
