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

// Builtin is a simple interface, only implementable on structs due to how the with re-parsing logic works
type Builtin interface {
	Execute(ctx context.Context) (map[string]any, error)
}

var _registrations = map[string]func() Builtin{
	"echo":          func() Builtin { return &echo{} },
	"fetch":         func() Builtin { return &fetch{} },
	"wacky-structs": func() Builtin { return &wackyStructs{} },
}

// Get retrieves a new instance of a registered builtin task
//
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

// Register registers a new builtin
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

// Names returns a list of all builtin names
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
