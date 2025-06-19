// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"

	"github.com/expr-lang/expr"
)

type If string

const (
	IfAlways  If = "always"
	IfFailure If = "failure"
)

func (i If) String() string {
	return string(i)
}

func (i If) ShouldRun(ctx context.Context, hasFailed bool) (bool, error) {
	if i == "" {
		return !hasFailed, nil
	}

	failure := expr.Function("failure", func(_ ...any) (any, error) {
		return hasFailed, nil
	})

	program, err := expr.Compile(i.String(), expr.AsBool(), failure)
	if err != nil {
		return false, err
	}

	out, err := expr.Run(program, nil)
	if err != nil {
		return false, err
	}

	return out.(bool), nil // this is safe due to expr.AsBool()
}
