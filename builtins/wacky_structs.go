// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package builtins

import (
	"context"
	"fmt"
)

// wackyStructs is a sample builtin to showcase wacky struct handling
type wackyStructs struct {
	Int    int
	Bool   bool
	String string
	Map    map[string]any
	Slice  []any
	Nested struct {
		Field    string
		Slice    []any
		IntSlice []int
		Map      map[string]any
		BoolMap  map[bool]bool
	}
}

// Execute the builtin
func (b wackyStructs) Execute(_ context.Context) (map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}
