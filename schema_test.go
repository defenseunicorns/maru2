// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

func TestWorkflowSchema(t *testing.T) {
	t.Parallel()
	t.Run(v0.SchemaVersion, func(t *testing.T) {
		t.Parallel()
		schema := WorkflowSchema(v0.SchemaVersion)
		b, err := json.Marshal(schema)
		require.NoError(t, err)

		current, err := os.ReadFile("schema/v0/schema.json")
		require.NoError(t, err)

		assert.JSONEq(t, string(current), string(b))
	})
	t.Run(v1.SchemaVersion, func(t *testing.T) {
		t.Parallel()
		schema := WorkflowSchema(v1.SchemaVersion)
		b, err := json.Marshal(schema)
		require.NoError(t, err)

		current, err := os.ReadFile("schema/v1/schema.json")
		require.NoError(t, err)

		assert.JSONEq(t, string(current), string(b))
	})
	t.Run("meta", func(t *testing.T) {
		t.Parallel()
		schema := WorkflowSchema("")
		b, err := json.Marshal(schema)
		require.NoError(t, err)

		current, err := os.ReadFile("maru2.schema.json")
		require.NoError(t, err)

		assert.JSONEq(t, string(current), string(b))
	})
	t.Run("meta schema contains v1 schema at correct location", func(t *testing.T) {
		t.Parallel()
		metaSchema := WorkflowSchema("")

		require.NotNil(t, metaSchema.If, "meta schema should have 'if' condition")
		require.NotNil(t, metaSchema.Then, "meta schema should have 'then' branch")
		require.NotNil(t, metaSchema.Else, "meta schema should have 'else' branch")

		v1Schema := WorkflowSchema(v1.SchemaVersion)
		v1SchemaBytes, err := json.Marshal(v1Schema)
		require.NoError(t, err)

		thenSchemaBytes, err := json.Marshal(metaSchema.Then)
		require.NoError(t, err)

		assert.JSONEq(t, string(v1SchemaBytes), string(thenSchemaBytes), "v1 schema should be in the 'then' branch of meta schema")

		require.NotNil(t, metaSchema.If.Properties, "meta schema 'if' should have properties")
		schemaVersionProp, ok := metaSchema.If.Properties.Get("schema-version")
		require.True(t, ok, "meta schema 'if' should have schema-version property")
		require.NotNil(t, schemaVersionProp, "meta schema 'if' should check schema-version property")
		assert.Contains(t, schemaVersionProp.Enum, "v1", "meta schema 'if' condition should check for 'v1'")
	})
	t.Run("meta schema contains v0 schema at correct location", func(t *testing.T) {
		metaSchema := WorkflowSchema("")

		require.NotNil(t, metaSchema.Else, "meta schema should have 'else' branch")
		require.NotNil(t, metaSchema.Else.If, "meta schema 'else' should have nested 'if' condition")
		require.NotNil(t, metaSchema.Else.Then, "meta schema 'else' should have nested 'then' branch")

		v0Schema := WorkflowSchema(v0.SchemaVersion)
		v0SchemaBytes, err := json.Marshal(v0Schema)
		require.NoError(t, err)

		elseThenSchemaBytes, err := json.Marshal(metaSchema.Else.Then)
		require.NoError(t, err)

		assert.JSONEq(t, string(v0SchemaBytes), string(elseThenSchemaBytes), "v0 schema should be in the nested 'then' branch of meta schema")

		require.NotNil(t, metaSchema.Else.If.Properties, "meta schema nested 'if' should have properties")
		schemaVersionProp, ok := metaSchema.Else.If.Properties.Get("schema-version")
		require.True(t, ok, "meta schema nested 'if' should have schema-version property")
		require.NotNil(t, schemaVersionProp, "meta schema nested 'if' should check schema-version property")
		assert.Contains(t, schemaVersionProp.Enum, "v0", "meta schema nested 'if' condition should check for 'v0'")
	})
}
