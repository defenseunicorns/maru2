// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderedInputs(t *testing.T) {
	testCases := []struct {
		name     string
		inputs   InputMap
		expected []string
	}{
		{
			name:     "nil",
			inputs:   nil,
			expected: []string{},
		},
		{
			name:     "empty",
			inputs:   InputMap{},
			expected: []string{},
		},
		{
			name: "single input",
			inputs: InputMap{
				"name": InputParameter{},
			},
			expected: []string{"name"},
		},
		{
			name: "multiple inputs - sorted order",
			inputs: InputMap{
				"zebra": InputParameter{},
				"alpha": InputParameter{},
				"beta":  InputParameter{},
			},
			expected: []string{"alpha", "beta", "zebra"},
		},
		{
			name: "inputs with similar names",
			inputs: InputMap{
				"input-2":  InputParameter{},
				"input-10": InputParameter{},
				"input-1":  InputParameter{},
			},
			expected: []string{"input-1", "input-10", "input-2"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := make([]string, 0)
			for name := range tc.inputs.OrderedSeq() {
				got = append(got, name)
			}
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("partial iteration", func(t *testing.T) {
		inputs := InputMap{
			"zebra": InputParameter{},
			"alpha": InputParameter{},
			"beta":  InputParameter{},
			"gamma": InputParameter{},
		}

		got := make([]string, 0)
		for name := range inputs.OrderedSeq() {
			got = append(got, name)
			if len(got) == 2 {
				break
			}
		}

		expected := []string{"alpha", "beta"}
		assert.Equal(t, expected, got)
	})
}
