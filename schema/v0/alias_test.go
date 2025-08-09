// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasSchema(t *testing.T) {
	t.Parallel()
	f, err := os.Open("schema.json")
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)

	var schema map[string]any
	require.NoError(t, json.Unmarshal(data, &schema))

	curr := schema["properties"].(map[string]any)["aliases"].(map[string]any)["additionalProperties"]
	b, err := json.Marshal(curr)
	require.NoError(t, err)

	reflector := jsonschema.Reflector{DoNotReference: true}
	aliasSchema := reflector.Reflect(Alias{})
	aliasSchema.Version = ""
	aliasSchema.ID = ""
	b2, err := json.Marshal(aliasSchema)
	require.NoError(t, err)

	assert.JSONEq(t, string(b), string(b2))
}
