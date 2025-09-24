// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// DefaultStyles returns the default styles.
func DefaultStyles() *log.Styles {
	styles := log.DefaultStyles()

	// https://github.com/charmbracelet/vhs/blob/main/themes.json
	styles.Levels[log.DebugLevel] = styles.Levels[log.DebugLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#2e7de9", // tokyonight-day blue
		Dark:  "#7aa2f7", // tokyonight blue
	})
	styles.Levels[log.InfoLevel] = styles.Levels[log.InfoLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#007197", // tokyonight-day cyan
		Dark:  "#7dcfff", // tokyonight cyan
	})
	styles.Levels[log.WarnLevel] = styles.Levels[log.WarnLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#8c6c3e", // tokyonight-day amber/yellow
		Dark:  "#e0af68", // tokyonight amber/yellow
	})
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#f52a65", // tokyonight-day red
		Dark:  "#f7768e", // tokyonight red
	})
	styles.Levels[log.FatalLevel] = styles.Levels[log.FatalLevel].Foreground(lipgloss.AdaptiveColor{
		Light: "#9854f1", // tokyonight-day magenta (deep red alternative)
		Dark:  "#bb9af7", // tokyonight magenta (deep red alternative)
	})

	return styles
}

var (
	FaintStyle = lipgloss.NewStyle().Faint(true)

	Green = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#587539", // tokyonight-day green
		Dark:  "#9ece6a", // tokyonight green
	})
)
