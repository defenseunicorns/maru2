// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package config

import (
	"fmt"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPolicy(t *testing.T) {
	t.Run("constants", func(t *testing.T) {
		assert.Equal(t, FetchPolicy("always"), FetchPolicyAlways)
		assert.Equal(t, FetchPolicy("if-not-present"), FetchPolicyIfNotPresent)
		assert.Equal(t, FetchPolicy("never"), FetchPolicyNever)
		assert.Equal(t, FetchPolicyIfNotPresent, DefaultFetchPolicy)
	})

	t.Run("available policies", func(t *testing.T) {
		policies := AvailablePolicies()
		assert.Len(t, policies, 3)
		assert.Contains(t, policies, string(FetchPolicyAlways))
		assert.Contains(t, policies, string(FetchPolicyIfNotPresent))
		assert.Contains(t, policies, string(FetchPolicyNever))
	})

	t.Run("pflag value interface", func(t *testing.T) {
		// Test String() method
		var policy FetchPolicy = FetchPolicyAlways
		assert.Equal(t, "always", policy.String())

		// Test Type() method
		assert.Equal(t, "string", policy.Type())

		// Test Set() with valid values - test all valid options
		err := policy.Set("always")
		assert.NoError(t, err)
		assert.Equal(t, FetchPolicyAlways, policy)

		err = policy.Set("if-not-present")
		assert.NoError(t, err)
		assert.Equal(t, FetchPolicyIfNotPresent, policy)

		err = policy.Set("never")
		assert.NoError(t, err)
		assert.Equal(t, FetchPolicyNever, policy)

		// Test invalid Set() operation
		err = policy.Set("invalid")
		assert.Error(t, err)
		assert.Equal(t, "invalid fetch policy: invalid", err.Error())
		assert.Equal(t, FetchPolicyNever, policy, "Policy should remain unchanged after invalid set")

		// Test pflag.Value interface compliance
		var flagValue pflag.Value = &policy
		assert.NotNil(t, flagValue)
	})

	t.Run("set method edge cases", func(t *testing.T) {
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
				assert.Error(t, err)
				require.EqualError(t, err, tc.expectedErr)
			})
		}
	})
}
