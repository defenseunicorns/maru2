// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := log.WithContext(t.Context(), log.New(io.Discard))
	with := With{}

	// simple happy path
	_, err := Run(ctx, helloWorldWorkflow, "", with, "file:test", false)
	require.NoError(t, err)

	// fast failure for 404
	_, err = Run(ctx, helloWorldWorkflow, "does not exist", with, "file:test", false)
	require.EqualError(t, err, "task \"does not exist\" not found")
}
