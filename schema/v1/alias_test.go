// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

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

func TestOrderedAliases(t *testing.T) {
	testCases := []struct {
		name     string
		aliases  AliasMap
		expected []string
	}{
		{
			name:     "nil",
			aliases:  nil,
			expected: []string{},
		},
		{
			name:     "empty",
			aliases:  AliasMap{},
			expected: []string{},
		},
		{
			name: "single alias - local",
			aliases: AliasMap{
				"local": Alias{},
			},
			expected: []string{"local"},
		},
		{
			name: "single alias - remote",
			aliases: AliasMap{
				"gh": Alias{},
			},
			expected: []string{"gh"},
		},
		{
			name: "multiple aliases - sorted order",
			aliases: AliasMap{
				"zebra": Alias{},
				"alpha": Alias{},
				"beta":  Alias{},
			},
			expected: []string{"alpha", "beta", "zebra"},
		},
		{
			name: "aliases with similar names",
			aliases: AliasMap{
				"task-2":  Alias{},
				"task-10": Alias{},
				"task-1":  Alias{},
			},
			expected: []string{"task-1", "task-10", "task-2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make([]string, 0)
			for name := range tc.aliases.OrderedSeq() {
				got = append(got, name)
			}
			assert.Equal(t, tc.expected, got)
		})
	}
}
