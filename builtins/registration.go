// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import "context"

// Builtin is a simple interface, only implementable on structs due to how the with re-parsing logic works
type Builtin interface {
	Execute(ctx context.Context) (map[string]any, error)
}

// Builtins maps builtin names to their implementations
var Builtins = map[string]Builtin{
	"echo":          echo{},
	"fetch":         fetch{},
	"wacky-structs": wackyStructs{},
}
