// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package cmd

import (
	"github.com/charmbracelet/log"

	"github.com/defenseunicorns/maru2"
)

// DefaultStyles returns the default styles.
func DefaultStyles() *log.Styles {
	styles := log.DefaultStyles()

	styles.Levels[log.DebugLevel] = styles.Levels[log.DebugLevel].Foreground(maru2.DebugColor)
	styles.Levels[log.InfoLevel] = styles.Levels[log.InfoLevel].Foreground(maru2.InfoColor)
	styles.Levels[log.WarnLevel] = styles.Levels[log.WarnLevel].Foreground(maru2.WarnColor)
	styles.Levels[log.ErrorLevel] = styles.Levels[log.ErrorLevel].Foreground(maru2.ErrorColor)
	styles.Levels[log.FatalLevel] = styles.Levels[log.FatalLevel].Foreground(maru2.FatalColor)

	return styles
}
