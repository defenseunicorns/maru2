// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package maru2 provides a simple task runner.
package maru2

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/uses"
)

// Run executes a task in a workflow with the given inputs.
//
// For all `uses` steps, this function will be called recursively.
// Returns the outputs from the final step in the task.
func Run(ctx context.Context, wf Workflow, taskName string, outer With, origin string, dry bool, svc *uses.FetcherService) (map[string]any, error) {
	if taskName == "" {
		taskName = DefaultTaskName
	}

	task, ok := wf.Tasks.Find(taskName)
	if !ok {
		return nil, addTrace(fmt.Errorf("task %q not found", taskName), fmt.Sprintf("at (%s)", origin))
	}

	withDefaults, err := MergeWithAndParams(ctx, outer, wf.Inputs)
	if err != nil {
		return nil, addTrace(err, fmt.Sprintf("at (%s)", origin))
	}

	logger := log.FromContext(ctx)
	outputs := make(CommandOutputs)
	var firstError error

	start := time.Now()
	for i, step := range task {
		if (firstError == nil && step.If == "failure") || (firstError != nil && step.If == "") {
			logger.Debug("skip", "step", fmt.Sprintf("%s[%d]", taskName, i), "if", step.If)
			continue
		}

		var stepResult map[string]any
		isLastStep := i == len(task)-1

		logger.Debug("run", "step", fmt.Sprintf("%s[%d]", taskName, i))

		if step.Uses != "" {
			stepResult, err = handleUsesStep(ctx, step, wf, withDefaults, outputs, origin, dry, svc)
		} else if step.Run != "" {
			stepResult, err = handleRunStep(ctx, step, withDefaults, outputs, dry)
		}

		logger.Debug("ran", "step", fmt.Sprintf("%s[%d]", taskName, i), "outputs", len(stepResult), "duration", time.Since(start))

		if err != nil && firstError == nil {
			firstError = addTrace(err, fmt.Sprintf("at %s[%d] (%s)", taskName, i, origin))
			continue
		}

		if isLastStep && stepResult != nil {
			return stepResult, firstError
		}

		if step.ID != "" && stepResult != nil {
			outputs[step.ID] = make(map[string]any, len(stepResult))
			maps.Copy(outputs[step.ID], stepResult)
		}
	}

	return nil, firstError
}

func handleUsesStep(ctx context.Context, step Step, wf Workflow, withDefaults With,
	outputs CommandOutputs, origin string, dry bool, svc *uses.FetcherService) (map[string]any, error) {

	if strings.HasPrefix(step.Uses, "builtin:") {
		return ExecuteBuiltin(ctx, step, withDefaults, outputs, dry)
	}

	templatedWith, err := TemplateWith(ctx, withDefaults, step.With, outputs, dry)
	if err != nil {
		return nil, err
	}

	if _, ok := wf.Tasks.Find(step.Uses); ok {
		return Run(ctx, wf, step.Uses, templatedWith, origin, dry, svc)
	}
	return ExecuteUses(ctx, step.Uses, templatedWith, origin, dry, svc)
}

func handleRunStep(ctx context.Context, step Step, withDefaults With,
	outputs CommandOutputs, dry bool) (map[string]any, error) {

	templatedRun, err := TemplateString(ctx, withDefaults, outputs, step.Run, dry)
	if err != nil {
		if dry {
			printScript(ctx, "$", templatedRun)
		}
		return nil, err
	}

	printScript(ctx, "$", templatedRun)
	if dry {
		return nil, nil
	}

	outFile, err := os.CreateTemp("", "maru2-output-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		outFile.Close()
		os.Remove(outFile.Name())
	}()

	env := prepareEnvironment(withDefaults, outFile.Name())

	cmd := exec.CommandContext(ctx, "sh", "-e", "-c", templatedRun)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	if step.ID != "" {
		out, err := ParseOutput(outFile)
		if err != nil || len(out) == 0 {
			return nil, err
		}

		result := make(map[string]any, len(out))
		for k, v := range out {
			result[k] = v
		}

		return result, nil
	}
	return nil, nil
}

func prepareEnvironment(withDefaults With, outFileName string) []string {
	env := os.Environ()

	for k, v := range withDefaults {
		var val string
		switch v := v.(type) {
		case string:
			val = v
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			val = fmt.Sprintf("%d", v)
		case bool:
			val = fmt.Sprintf("%t", v)
		}
		env = append(env, fmt.Sprintf("INPUT_%s=%s", toEnvVar(k), val))
	}

	env = append(env, fmt.Sprintf("MARU2_OUTPUT=%s", outFileName))
	return env
}

func toEnvVar(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", "_"))
}

// TraceError is an error with a logical stack trace
type TraceError struct {
	err   error    // The original error
	Trace []string // Logical stack trace
}

var _ error = &TraceError{}

// Error returns the original error message
func (e *TraceError) Error() string {
	return e.err.Error()
}

// Unwrap returns the underlying error
func (e *TraceError) Unwrap() error {
	return e.err
}

// addTrace adds a new frame and returns a new TraceError
func addTrace(err error, frame string) error {
	var tErr *TraceError
	if errors.As(err, &tErr) {
		tErr.Trace = append([]string{frame}, tErr.Trace...)
		return tErr
	}

	return &TraceError{
		err:   err,
		Trace: []string{frame},
	}
}
