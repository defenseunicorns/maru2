// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

var _register sync.RWMutex

// Builtin defines the interface for built-in tasks (builtin:echo, builtin:fetch)
//
// Implementations must be structs to support configuration binding via mapstructure.
// The Execute method receives context and returns outputs that can be accessed by subsequent steps
type Builtin interface {
	Execute(ctx context.Context) (map[string]any, error)
}

var _registrations = map[string]func() Builtin{
	"echo":          func() Builtin { return &echo{} },
	"fetch":         func() Builtin { return &fetch{} },
	"wacky-structs": func() Builtin { return &wackyStructs{} },
}

// Get retrieves a fresh instance of a registered builtin task
//
// Each call returns a new instance to avoid shared state between executions.
// Returns nil if the builtin doesn't exist
func Get(name string) Builtin {
	_register.RLock()
	factory, exists := _registrations[name]
	_register.RUnlock()

	if !exists {
		return nil
	}
	return factory()
}

// Register adds a new builtin task to the global registry
//
// Used by internal packages and extensions to provide additional builtin functionality.
// Registration functions must return fresh instances to avoid shared state
func Register(name string, registrationFunc func() Builtin) error {
	_register.Lock()
	defer _register.Unlock()

	_, exists := _registrations[name]
	if exists {
		return fmt.Errorf("%q is already registered", name)
	}

	if name == "" {
		return fmt.Errorf("builtin name cannot be empty")
	}

	if registrationFunc == nil {
		return fmt.Errorf("registration function cannot be nil")
	}

	_registrations[name] = registrationFunc
	return nil
}

// Names returns a sorted list of all registered builtin task names
//
// Used for completion, help text, and validation of builtin: references
func Names() []string {
	_register.RLock()
	defer _register.RUnlock()

	result := make([]string, 0, len(_registrations))
	for name := range _registrations {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}
