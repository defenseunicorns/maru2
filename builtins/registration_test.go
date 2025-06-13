// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBuiltin struct {
	ExecuteFunc func(ctx context.Context) (map[string]any, error)
}

func (m mockBuiltin) Execute(ctx context.Context) (map[string]any, error) {
	if m.ExecuteFunc == nil {
		return map[string]any{"result": "default"}, nil
	}
	return m.ExecuteFunc(ctx)
}

func TestRegister(t *testing.T) {
	// Don't run this test in parallel to avoid race conditions with other tests

	tests := []struct {
		name             string
		builtinName      string
		existingName     bool
		registrationFunc func() Builtin
		expectedError    string
	}{
		{
			name:         "register new builtin",
			builtinName:  "test-builtin",
			existingName: false,
			registrationFunc: func() Builtin {
				return &mockBuiltin{
					ExecuteFunc: func(_ context.Context) (map[string]any, error) {
						return map[string]any{"result": "test"}, nil
					},
				}
			},
			expectedError: "",
		},
		{
			name:         "register duplicate builtin",
			builtinName:  "duplicate-builtin",
			existingName: true,
			registrationFunc: func() Builtin {
				return &mockBuiltin{
					ExecuteFunc: func(_ context.Context) (map[string]any, error) {
						return map[string]any{"result": "test"}, nil
					},
				}
			},
			expectedError: "\"duplicate-builtin\" is already registered",
		},
		{
			name:         "register with empty name",
			builtinName:  "",
			existingName: false,
			registrationFunc: func() Builtin {
				return &mockBuiltin{
					ExecuteFunc: func(_ context.Context) (map[string]any, error) {
						return map[string]any{"result": "test"}, nil
					},
				}
			},
			expectedError: "builtin name cannot be empty",
		},
		{
			name:             "register with nil function",
			builtinName:      "nil-func",
			existingName:     false,
			registrationFunc: nil,
			expectedError:    "registration function cannot be nil",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			if tc.existingName {
				err := Register(tc.builtinName, func() Builtin {
					return mockBuiltin{
						ExecuteFunc: func(_ context.Context) (map[string]any, error) {
							return map[string]any{"result": "first"}, nil
						},
					}
				})
				require.NoError(t, err)
			}

			err := Register(tc.builtinName, tc.registrationFunc)

			if tc.expectedError == "" {
				require.NoError(t, err)

				builtin := Get(tc.builtinName)
				require.NotNil(t, builtin)

				result, execErr := builtin.Execute(t.Context())
				require.NoError(t, execErr)
				assert.Equal(t, "test", result["result"])
			} else {
				require.EqualError(t, err, tc.expectedError)
			}

			_register.Lock()
			delete(_registrations, tc.builtinName)
			_register.Unlock()
		})
	}
}

func TestConcurrentOperations(t *testing.T) {
	done := make(chan bool)

	for i := range 5 {
		go func(id int) {
			name := fmt.Sprintf("concurrent-test-%d", id)
			err := Register(name, func() Builtin {
				return &mockBuiltin{
					ExecuteFunc: func(_ context.Context) (map[string]any, error) {
						return nil, nil
					},
				}
			})
			assert.NoError(t, err)

			builtin := Get(name)
			assert.NotNil(t, builtin)

			builtinNames := Names()
			assert.Contains(t, builtinNames, name)

			done <- true
		}(i)
	}

	for range 5 {
		<-done
	}

	_register.Lock()
	for i := range 5 {
		delete(_registrations, fmt.Sprintf("concurrent-test-%d", i))
	}
	_register.Unlock()
}
