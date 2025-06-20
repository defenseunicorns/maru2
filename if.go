// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"fmt"
	"html/template"
	"maps"
	"slices"
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
func (i If) ShouldRun(hasFailed bool, with With, from CommandOutputs) (bool, error) {
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

	type env struct {
		Inputs With           `expr:"inputs"`
		From   CommandOutputs `expr:"from"`
	}

	program, err := expr.Compile(i.String(), expr.Env(env{}), expr.AsBool(), failure, always)
	if err != nil {
		return false, err
	}

	out, err := expr.Run(program, env{with, from})
	if err != nil {
		return false, err
	}

	if alwaysTriggered { // always short circuits any other logic
		return true, nil
	}

	return out.(bool), nil // this is safe due to expr.AsBool()
}

// ShouldRunTemplate executes If logic using text/template as the engine
func (i If) ShouldRunTemplate(hasFailed bool, with With, from CommandOutputs) (bool, error) {
	if i == "" {
		return !hasFailed, nil
	}

	inputKeys := make([]string, 0, len(with))
	for k := range maps.Keys(with) {
		inputKeys = append(inputKeys, k)
	}
	slices.Sort(inputKeys)

	var alwaysTriggered bool
	fm := template.FuncMap{
		"failure": func() bool {
			return hasFailed
		},
		"always": func() bool {
			alwaysTriggered = true
			return true
		},
		// same as TemplateString
		"input": func(in string) (any, error) {
			v, ok := with[in]
			if !ok {
				return "", fmt.Errorf("input %q does not exist in %s", in, inputKeys)
			}
			return v, nil
		},
		// same as TemplateString
		"from": func(stepName, id string) (any, error) {
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
