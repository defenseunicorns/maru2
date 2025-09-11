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

	"github.com/defenseunicorns/maru2/schema"
)

// ShouldRun evaluates if expressions using the expr engine
//
// Provides built-in functions: failure(), always(), cancelled(), input("name"), from("step-id", "key")
//
// Returns false for failed steps when no expression is provided
func ShouldRun(ctx context.Context, expression string, err error, with schema.With, previousOutputs CommandOutputs, dry bool) (bool, error) {
	hasFailed := err != nil

	if expression == "" {
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

	inputFunc := expr.Function(
		"input",
		func(params ...any) (any, error) {
			in := params[0].(string)
			v, ok := with[in]
			if !ok {
				return nil, nil
			}
			return v, nil
		},
		new(func(string) any),
	)

	fromFunc := expr.Function(
		"from",
		func(params ...any) (any, error) {
			stepName := params[0].(string)
			id := params[1].(string)
			stepOutputs, ok := previousOutputs[stepName]
			if !ok {
				return nil, nil
			}

			v, ok := stepOutputs[id]
			if ok {
				return v, nil
			}
			return nil, nil
		},
		new(func(string, string) any),
	)

	// mirrors TemplateString presets
	type env struct {
		OS       string `expr:"os"`
		Arch     string `expr:"arch"`
		Platform string `expr:"platform"`
	}

	program, err := expr.Compile(expression, expr.Env(env{}), expr.AsBool(), failure, cancelled, always, inputFunc, fromFunc)
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

	ok, val := out.(bool)
	if !ok {
		return false, fmt.Errorf("expression did not evaluate to a boolean")
	}
	return val, nil
}
