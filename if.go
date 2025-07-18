// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"runtime"

	"github.com/expr-lang/expr"
)

// If controls whether a step is run
type If string

// String implements fmt.Stringer
func (i If) String() string {
	return string(i)
}

// ShouldRun executes If logic using expr as the engine
func (i If) ShouldRun(hasFailed bool, with With, from CommandOutputs, dry bool) (bool, error) {
	if i == "" {
		return !hasFailed, nil
	}

	failure := expr.Function(
		"failure",
		func(_ ...any) (any, error) {
			return hasFailed, nil
		},
		new(func() bool),
	)

	var alwaysTriggered bool
	always := expr.Function(
		"always",
		func(_ ...any) (any, error) {
			alwaysTriggered = true
			return true, nil
		},
		new(func() bool),
	)

	env := map[string]any{
		"input": With{},
		"from":  CommandOutputs{},
		"os":    "",
		"arch":  "",
	}

	program, err := expr.Compile(i.String(), expr.Env(env), expr.AsBool(), failure, always)
	if err != nil {
		return false, err
	}

	if dry {
		return false, nil
	}

	out, err := expr.Run(
		program,
		map[string]any{"input": with, "from": from, "os": runtime.GOOS, "arch": runtime.GOARCH},
	)
	if err != nil {
		return false, err
	}

	if alwaysTriggered { // always short circuits any other logic
		return true, nil
	}

	return out.(bool), nil // this is safe due to expr.AsBool()
}
