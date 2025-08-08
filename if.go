// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"

	"github.com/expr-lang/expr"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

// If controls whether a step is run
type If string

// String implements fmt.Stringer
func (i If) String() string {
	return string(i)
}

// ShouldRun executes If logic using expr as the engine
func (i If) ShouldRun(ctx context.Context, err error, with v0.With, from CommandOutputs, dry bool) (bool, error) {
	hasFailed := err != nil

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

	cancelled := expr.Function(
		"cancelled",
		func(_ ...any) (any, error) {
			return ctx != nil && errors.Is(ctx.Err(), context.Canceled), nil
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

	inputKeys := make([]string, 0, len(with))
	for k := range with {
		inputKeys = append(inputKeys, k)
	}
	slices.Sort(inputKeys)

	// mirrors TemplateString func
	inputFunc := expr.Function(
		"input",
		func(params ...any) (any, error) {
			in := params[0].(string)
			v, ok := with[in]
			if !ok {
				return nil, fmt.Errorf("input %q does not exist in %s", in, inputKeys)
			}
			return v, nil
		},
		new(func(string) (any, error)),
	)

	// mirrors TemplateString func
	fromFunc := expr.Function(
		"from",
		func(params ...any) (any, error) {
			stepName := params[0].(string)
			id := params[1].(string)
			stepOutputs, ok := from[stepName]
			if !ok {
				return "", fmt.Errorf("no outputs from step %q", stepName)
			}

			v, ok := stepOutputs[id]
			if ok {
				return v, nil
			}
			return "", fmt.Errorf("no output %q from step %q", id, stepName)
		},
		new(func(string, string) (any, error)),
	)

	// mirrors TemplateString presets
	type env struct {
		OS       string `expr:"os"`
		Arch     string `expr:"arch"`
		Platform string `expr:"platform"`
	}

	program, err := expr.Compile(i.String(), expr.Env(env{}), expr.AsBool(), failure, cancelled, always, inputFunc, fromFunc)
	if err != nil {
		return false, err
	}

	if dry {
		return false, nil
	}

	out, err := expr.Run(
		program,
		env{OS: runtime.GOOS, Arch: runtime.GOARCH, Platform: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)},
	)
	if err != nil {
		return false, err
	}

	if alwaysTriggered { // always short circuits any other logic
		return true, nil
	}

	return out.(bool), nil // this is safe due to expr.AsBool()
}
