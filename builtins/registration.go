// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import (
	"context"
	"slices"
)

// Builtin is a simple interface, only implementable on structs due to how the with re-parsing logic works
type Builtin interface {
	Execute(ctx context.Context) (map[string]any, error)
}

var builtinFactories = map[string]func() Builtin{
	"echo":          func() Builtin { return &echo{} },
	"fetch":         func() Builtin { return &fetch{} },
	"wacky-structs": func() Builtin { return &wackyStructs{} },
}

// Get returns a new instance of the requested builtin
// Returns nil if the builtin doesn't exist
func Get(name string) Builtin {
	factory, exists := builtinFactories[name]
	if !exists {
		return nil
	}
	return factory()
}

// Names returns a list of all builtin names
func Names() []string {
	result := make([]string, 0, len(builtinFactories))
	for name := range builtinFactories {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}
