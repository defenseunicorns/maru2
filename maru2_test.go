// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"io"
	"net/url"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := log.WithContext(t.Context(), log.New(io.Discard))
	with := With{}

	// Create test fetcher service
	svc, err := uses.NewFetcherService(nil, nil)
	require.NoError(t, err)

	dummyOrigin := &url.URL{
		Scheme: "file",
		Path:   "test",
	}

	// simple happy path
	_, err = Run(ctx, helloWorldWorkflow, "", with, dummyOrigin, false, svc)
	require.NoError(t, err)

	// fast failure for 404
	_, err = Run(ctx, helloWorldWorkflow, "does not exist", with, dummyOrigin, false, svc)
	require.EqualError(t, err, "task \"does not exist\" not found")
}
