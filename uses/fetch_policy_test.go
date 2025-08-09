// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPolicy(t *testing.T) {
	t.Run("constants", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, FetchPolicyAlways, FetchPolicy("always"))
		assert.Equal(t, FetchPolicyIfNotPresent, FetchPolicy("if-not-present"))
		assert.Equal(t, FetchPolicyNever, FetchPolicy("never"))
		assert.Equal(t, FetchPolicyIfNotPresent, DefaultFetchPolicy)
	})

	t.Run("available policies", func(t *testing.T) {
		t.Parallel()
		policies := AvailablePolicies()
		assert.Len(t, policies, 3)
		assert.Contains(t, policies, string(FetchPolicyAlways))
		assert.Contains(t, policies, string(FetchPolicyIfNotPresent))
		assert.Contains(t, policies, string(FetchPolicyNever))
	})

	t.Run("pflag value interface", func(t *testing.T) {
		t.Parallel()
		// Test String() method
		var policy = FetchPolicyAlways
		assert.Equal(t, "always", policy.String())

		// Test Type() method
		assert.Equal(t, "string", policy.Type())

		// Test Set() with valid values - test all valid options
		err := policy.Set("always")
		require.NoError(t, err)
		assert.Equal(t, FetchPolicyAlways, policy)

		err = policy.Set("if-not-present")
		require.NoError(t, err)
		assert.Equal(t, FetchPolicyIfNotPresent, policy)

		err = policy.Set("never")
		require.NoError(t, err)
		assert.Equal(t, FetchPolicyNever, policy)

		// Test invalid Set() operation
		err = policy.Set("invalid")
		require.EqualError(t, err, "invalid fetch policy: invalid")
		assert.Equal(t, FetchPolicyNever, policy, "Policy should remain unchanged after invalid set")
	})

	t.Run("set method edge cases", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			value       string
			expectedErr string
		}{
			{value: "", expectedErr: "invalid fetch policy: "},
			{value: " never ", expectedErr: "invalid fetch policy:  never "},
			{value: "Never", expectedErr: "invalid fetch policy: Never"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("set_%s", tc.value), func(t *testing.T) {
				var policy FetchPolicy
				err := policy.Set(tc.value)
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedErr)
			})
		}
	})

	t.Run("JSON schema", func(t *testing.T) {
		t.Parallel()

		golden := `{"type":"string","enum":["always","if-not-present","never"],"description":"Policy for fetching resources"}`

		reflector := jsonschema.Reflector{DoNotReference: true}
		fetchPolicySchema := reflector.Reflect(FetchPolicy(""))
		fetchPolicySchema.Version = ""
		fetchPolicySchema.ID = ""

		b, err := json.Marshal(fetchPolicySchema)
		require.NoError(t, err)

		assert.JSONEq(t, golden, string(b))
	})
}
