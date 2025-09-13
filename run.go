// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package maru2 provides a simple task runner.
package maru2

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/charmbracelet/log"
	"github.com/spf13/cast"

	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

/*
Run is the main event loop in maru2

It is implemented as a recursive function instead of a DAG for simplicity (debatable)

Run follows the following general pattern:

 1. Find the called task in the provided workflow

 2. Merge the provided inputs w/ the default workflow inputs

 3. Create a child context to listen for SIGINT

 4. For each step in the task:

    4a. Compile `if` conditionals and determine if the step should run

    4b. Soft reset the context if a previous step was cancelled, timed out, etc...

    4c. Wrap the current context in a timeout if `timeout` was set

    4d. If `uses` is set, resolve & fetch, then goto Step 1

    4e. If `run` is set, render the script with the provided inputs / environment

    4f. Parse the outputs from the script and store for later step retrieval

    4g. Add tracing if there was an error

 5. Return the final step's output and the first error encountered
*/
func Run(
	parent context.Context,
	svc *uses.FetcherService,
	wf v1.Workflow,
	taskName string,
	outer schema.With,
	origin *url.URL,
	cwd string,
	environVars []string,
	dry bool,
) (map[string]any, error) {
	if taskName == "" {
		taskName = schema.DefaultTaskName
	}

	task, ok := wf.Tasks.Find(taskName)
	if !ok {
		return nil, addTrace(fmt.Errorf("task %q not found", taskName), fmt.Sprintf("at (%s)", origin))
	}

	withDefaults, err := MergeWithAndParams(parent, outer, task.Inputs)
	if err != nil {
		return nil, addTrace(err, fmt.Sprintf("at %s.inputs (%s)", taskName, origin))
	}

	logger := log.FromContext(parent)
	outputs := make(CommandOutputs)
	var firstError error
	var lastStepOutput map[string]any

	start := time.Now()
	logger.Debug("run", "task", taskName, "from", origin, "dry-run", dry)
	defer func() {
		logger.Debug("ran", "task", taskName, "from", origin, "duration", time.Since(start))
	}()

	sigCtx, cancel := signal.NotifyContext(parent, syscall.SIGINT)
	defer cancel()

	var taskCancelledLogOnce sync.Once

	for i, step := range task.Steps {
		sub := logger.With("step", fmt.Sprintf("%s[%d]", taskName, i))
		err := func(ctx context.Context) error {
			shouldRun, err := ShouldRun(ctx, step.If, firstError, withDefaults, outputs, dry)
			if err != nil {
				if firstError != nil {
					// if there was an error calculating if we should run during the error path
					// log the error, but don't return it
					sub.Error("invalid", "if", step.If, "error", err)
					return nil
				}
				return err
			}
			if !shouldRun && !dry {
				sub.Debug("completed", "skipped", true)
				return nil
			}

			if errors.Is(ctx.Err(), context.Canceled) {
				taskCancelledLogOnce.Do(func() {
					sub.Warn("task cancelled")
				})
				// reset to use the parent context, but still respect
				// SIGTERM and timeout cancellation
				ctx = parent
			}

			if errors.Is(parent.Err(), context.DeadlineExceeded) {
				// if the parent context timed out, but we still need to run, eg. if: always()
				// then fully reset the context
				ctx = context.WithoutCancel(parent)
			}

			if step.Timeout != "" {
				timeout, err := time.ParseDuration(step.Timeout)
				if err != nil {
					return err
				}
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			var stepResult map[string]any

			if step.Uses != "" {
				stepResult, err = handleUsesStep(ctx, svc, step, wf, withDefaults, outputs, origin, cwd, environVars, dry)
			} else if step.Run != "" {
				stepResult, err = handleRunStep(ctx, step, withDefaults, outputs, cwd, environVars, dry)
			}

			if err != nil {
				return err
			}

			sub.Debug("completed", "outputs", len(stepResult), "duration", time.Since(start))

			isLastStep := i == len(task.Steps)-1
			if isLastStep {
				lastStepOutput = stepResult
			}

			if step.ID != "" && len(stepResult) > 0 {
				outputs[step.ID] = make(map[string]any, len(stepResult))
				maps.Copy(outputs[step.ID], stepResult)
			}

			return nil
		}(sigCtx)

		if err != nil {
			if firstError == nil {
				firstError = addTrace(err, fmt.Sprintf("at %s[%d] (%s)", taskName, i, origin))
				// log the first error if it was caused by a command execution
				if step.Run != "" {
					logger.Error(err)
				}
			} else {
				sub.Warn("failure during error handling", "err", err)
			}
		}
	}

	return lastStepOutput, firstError
}

func handleRunStep(
	ctx context.Context,
	step v1.Step,
	withDefaults schema.With,
	outputs CommandOutputs,
	cwd string,
	environVars []string,
	dry bool,
) (map[string]any, error) {

	logger := log.FromContext(ctx)

	script, err := TemplateString(ctx, step.Run, withDefaults, outputs, dry)
	if err != nil {
		if dry {
			printScript(logger, step.Shell, script)
		}
		return nil, err
	}

	printScript(logger, step.Shell, script)
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

	templatedEnv, err := TemplateWithMap(ctx, step.Env, withDefaults, outputs, dry)
	if err != nil {
		return nil, err
	}

	env, err := prepareEnvironment(environVars, withDefaults, outFile.Name(), templatedEnv)
	if err != nil {
		return nil, err
	}

	shell := step.Shell
	var args []string

	switch step.Shell {
	case "bash":
		args = []string{"-e", "-o", "pipefail", "-c", script}
	case "pwsh", "powershell":
		logger.Warn("support for this shell is currently untested and will potentially be removed in future versions", "shell", step.Shell)
		args = []string{"-Command", "$ErrorActionPreference = 'Stop';", script, "; if ((Test-Path -LiteralPath variable:\\LASTEXITCODE)) { exit $LASTEXITCODE }"}
	case "", "sh":
		shell = "sh"
		args = []string{"-e", "-c", script}
	default:
		return nil, fmt.Errorf("unsupported shell: %s", step.Shell)
	}

	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Env = env
	cmd.Dir = filepath.Join(cwd, step.Dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if step.Mute {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	if err := cmd.Run(); err != nil {
		return nil, err
	}

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

// prepareEnvironment builds the final environment variable list for command execution
//
// Combines system env vars, input parameters as env vars, step-level env vars,
// and the output file path for step communication
func prepareEnvironment(envVars []string, withDefaults schema.With, outFileName string, stepEnv schema.Env) ([]string, error) {
	env := make([]string, len(envVars), len(envVars)+len(withDefaults)+len(stepEnv)+1)
	copy(env, envVars)

	// keeping this local until i figure out if i want to break it out individually
	needsQuoting := func(s string) bool {
		// Check for spaces or other problematic characters
		for _, r := range s {
			if unicode.IsSpace(r) || r == '=' || r == '"' || r == '\n' {
				return true
			}
		}
		return false
	}

	for k, v := range withDefaults {
		val, err := cast.ToStringE(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input %q to string: %w", k, err)
		}
		if needsQuoting(val) {
			env = append(env, fmt.Sprintf("INPUT_%s=%q", toEnvVar(k), val))
		} else {
			env = append(env, fmt.Sprintf("INPUT_%s=%s", toEnvVar(k), val))
		}
	}

	for k, v := range stepEnv {
		// Prevent setting PWD as it should be controlled by exec.Command's Dir field
		if strings.EqualFold(k, "PWD") {
			return nil, fmt.Errorf("setting PWD environment variable is not allowed")
		}

		val, err := cast.ToStringE(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert env var %q to string: %w", k, err)
		}
		if needsQuoting(val) {
			env = append(env, fmt.Sprintf("%s=%q", k, val))
		} else {
			env = append(env, fmt.Sprintf("%s=%s", k, val))
		}
	}

	if outFileName != "" {
		env = append(env, fmt.Sprintf("MARU2_OUTPUT=%s", outFileName))
	}

	return env, nil
}

// toEnvVar converts input parameter names to environment variable format
//
// Transforms kebab-case to SCREAMING_SNAKE_CASE (e.g., "my-input" -> "MY_INPUT")
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

// addTrace wraps errors with execution context for debugging
//
// Creates or extends a TraceError with frame information to show
// the execution path when errors occur
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
