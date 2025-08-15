// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"encoding/json"
	"os"
	"testing"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowSchema(t *testing.T) {
	t.Run(v0.SchemaVersion, func(t *testing.T) {
		schema := WorkflowSchema(v0.SchemaVersion)
		b, err := json.Marshal(schema)
		require.NoError(t, err)

		current, err := os.ReadFile("schema/v0/schema.json")
		require.NoError(t, err)

		assert.JSONEq(t, string(current), string(b))
	})
	t.Run("meta", func(t *testing.T) {
		schema := WorkflowSchema("")
		b, err := json.Marshal(schema)
		require.NoError(t, err)

		current, err := os.ReadFile("maru2.schema.json")
		require.NoError(t, err)

		assert.JSONEq(t, string(current), string(b))
	})
}
