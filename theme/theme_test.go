// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package theme

import (
	"os"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	t.Parallel()

	t.Run("returns blank theme when missing", func(t *testing.T) {
		t.Parallel()

		actual := FromContext(t.Context())
		require.Equal(t, colorprofile.NoTTY, actual.ColorProfile)
		require.Equal(t, blankColor, actual.Debug)
		require.NotNil(t, actual.LogStyles)
	})

	t.Run("returns stored theme", func(t *testing.T) {
		t.Parallel()

		expected := Theme{
			Debug:        lipgloss.Color("#112233"),
			Info:         lipgloss.Color("#223344"),
			Warn:         lipgloss.Color("#334455"),
			Error:        lipgloss.Color("#445566"),
			Fatal:        lipgloss.Color("#556677"),
			Green:        lipgloss.Color("#667788"),
			Gray:         lipgloss.Color("#778899"),
			LogStyles:    log.DefaultStyles(),
			ColorProfile: colorprofile.ANSI256,
			ChromaStyle:  "test-style",
		}

		actual := FromContext(WithContext(t.Context(), expected))
		require.Equal(t, expected, actual)
	})
}

func TestNewStringWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	testTheme := Theme{ColorProfile: colorprofile.NoTTY}

	writer := testTheme.NewStringWriter(&buf)
	n, err := writer.WriteString("\x1b[31mred\x1b[m")

	require.NoError(t, err)
	require.Equal(t, len("\x1b[31mred\x1b[m"), n)
	require.Equal(t, "red", buf.String())

	profileWriter, ok := writer.(*colorprofile.Writer)
	require.True(t, ok)
	require.Equal(t, colorprofile.NoTTY, profileWriter.Profile)
	require.Same(t, &buf, profileWriter.Forward)
}

func TestDefaultTheme(t *testing.T) {
	t.Setenv("TTY_FORCE", "1")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")

	testTheme := DefaultTheme(&strings.Builder{}, &strings.Builder{})

	require.Equal(t, lipgloss.Color("#7aa2f7"), testTheme.Debug)
	require.Equal(t, lipgloss.Color("#7dcfff"), testTheme.Info)
	require.Equal(t, lipgloss.Color("#e0af68"), testTheme.Warn)
	require.Equal(t, lipgloss.Color("#f7768e"), testTheme.Error)
	require.Equal(t, lipgloss.Color("#bb9af7"), testTheme.Fatal)
	require.Equal(t, lipgloss.Color("#9ece6a"), testTheme.Green)
	require.Equal(t, lipgloss.Color("#3a3943"), testTheme.Gray)
	require.Equal(t, colorprofile.TrueColor, testTheme.ColorProfile)
	require.Equal(t, "tokyonight-moon", testTheme.ChromaStyle)
	require.NotNil(t, testTheme.LogStyles)
	require.Equal(t, testTheme.Debug, testTheme.LogStyles.Levels[log.DebugLevel].GetForeground())
	require.Equal(t, testTheme.Info, testTheme.LogStyles.Levels[log.InfoLevel].GetForeground())
	require.Equal(t, testTheme.Warn, testTheme.LogStyles.Levels[log.WarnLevel].GetForeground())
	require.Equal(t, testTheme.Error, testTheme.LogStyles.Levels[log.ErrorLevel].GetForeground())
	require.Equal(t, testTheme.Fatal, testTheme.LogStyles.Levels[log.FatalLevel].GetForeground())
}

func TestDefaultThemeWithFiles(t *testing.T) {
	in, err := os.CreateTemp(t.TempDir(), "in")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, in.Close())
	})

	out, err := os.CreateTemp(t.TempDir(), "out")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, out.Close())
	})

	testTheme := DefaultTheme(in, out)

	require.Equal(t, lipgloss.Color("#7aa2f7"), testTheme.Debug)
	require.Equal(t, "tokyonight-moon", testTheme.ChromaStyle)
}

func TestBlankTheme(t *testing.T) {
	t.Parallel()

	testTheme := BlankTheme()

	require.Equal(t, colorprofile.NoTTY, testTheme.ColorProfile)
	require.Equal(t, blankColor, testTheme.Debug)
	require.Equal(t, blankColor, testTheme.Info)
	require.Equal(t, blankColor, testTheme.Warn)
	require.Equal(t, blankColor, testTheme.Error)
	require.Equal(t, blankColor, testTheme.Fatal)
	require.Equal(t, blankColor, testTheme.Green)
	require.Equal(t, blankColor, testTheme.Gray)
	require.Empty(t, testTheme.ChromaStyle)
	require.Equal(t, log.DefaultStyles(), testTheme.LogStyles)
}
