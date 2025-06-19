// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"html/template"
	"strings"

	"github.com/expr-lang/expr"
)

// If controls whether a step is run
type If string

// String implements fmt.Stringers
func (i If) String() string {
	return string(i)
}

// ShouldRun executes If logic using expr as the engine
func (i If) ShouldRun(_ context.Context, hasFailed bool, with With, from CommandOutputs) (bool, error) {
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

	program, err := expr.Compile(i.String(), expr.AsBool(), failure, always)
	if err != nil {
		return false, err
	}

	env := map[string]any{
		"inputs": with,
		"from":   from,
	}

	out, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}

	if alwaysTriggered { // always short circuits any other logic
		return true, nil
	}

	return out.(bool), nil // this is safe due to expr.AsBool()
}

// ShouldRunTemplate executes If logic using text/template as the engine
func (i If) ShouldRunTemplate(_ context.Context, hasFailed bool) (bool, error) {
	if i == "" {
		return !hasFailed, nil
	}

	var alwaysTriggered bool
	fm := template.FuncMap{
		"failure": func() bool {
			return hasFailed
		},
		"always": func() bool {
			alwaysTriggered = true
			return true
		},
	}

	tmpl, err := template.New("should run").Funcs(fm).Option("missingkey=error").Delims("${{", "}}").Parse(i.String())
	if err != nil {
		return false, err
	}

	var result strings.Builder

	if err := tmpl.Execute(&result, nil); err != nil {
		return false, err
	}

	if alwaysTriggered { // always short circuits any other logic
		return true, nil
	}

	// now things get ugly
	r := strings.TrimSpace(result.String())

	return r == "true", nil
}
