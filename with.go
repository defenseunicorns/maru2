// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"maps"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"text/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cast"
)

// With is a map of string keys and WithEntry values used to pass parameters to called tasks and within steps
//
// Each key will be mapped to an equivalent environment variable
// when the command is run. eg. `with: {foo: bar}` will be passed
// as `foo=bar` to the command.
type With = map[string]any

// TemplateWith templates a With map with the given input and previous outputs
func TemplateWith(ctx context.Context, input, local With, previousOutputs CommandOutputs, dry bool) (With, error) {
	logger := log.FromContext(ctx)

	if len(local) == 0 {
		return input, nil
	}

	logger.Debug("templating", "input", input, "local", local)

	r := make(With, len(local))

	for k, v := range local {
		val, ok := v.(string)
		// if the val is not a string we can skip templating
		if !ok {
			r[k] = v
			continue
		}
		result, err := TemplateString(ctx, input, previousOutputs, val, dry)
		if err != nil {
			return nil, err
		}
		r[k] = result
	}

	logger.Debug("templated", "result", r)

	return r, nil
}

// TemplateString templates a string with the given input and previous outputs
func TemplateString(ctx context.Context, input With, previousOutputs CommandOutputs, str string, dry bool) (string, error) {
	var tmpl *template.Template

	inputKeys := make([]string, 0, len(input))
	for k := range maps.Keys(input) {
		inputKeys = append(inputKeys, k)
	}
	slices.Sort(inputKeys)

	logger := log.FromContext(ctx)

	if dry {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBF00")) // amber

		fm := template.FuncMap{
			"input": func(in string) (any, error) {
				v, ok := input[in]
				if !ok {
					logger.Warnf("input %q was not provided, available: %s", in, inputKeys)
					return style.Render(fmt.Sprintf("❯ input %s ❮", in)), nil
				}
				return v, nil
			},
			"from": func(stepName, id string) (any, error) {
				stepOutputs, ok := previousOutputs[stepName]
				if !ok {
					logger.Warnf("no outputs from step %q", stepName)
					return style.Render(fmt.Sprintf("❯ from %s %s ❮", stepName, id)), nil
				}

				v, ok := stepOutputs[id]
				if ok {
					return v, nil
				}
				logger.Warnf("no output %q from %q", id, stepName)
				return style.Render(fmt.Sprintf("❯ from %s %s ❮", stepName, id)), nil
			},
		}
		tmpl = template.New("dry-run expression evaluator").Funcs(fm)
	} else {
		fm := template.FuncMap{
			"input": func(in string) (any, error) {
				v, ok := input[in]
				if !ok {
					return "", fmt.Errorf("input %q does not exist in %s", in, inputKeys)
				}
				return v, nil
			},
			"from": func(stepName, id string) (any, error) {
				stepOutputs, ok := previousOutputs[stepName]
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
		tmpl = template.New("expression evaluator").Funcs(fm)
	}

	var err error
	tmpl, err = tmpl.Option("missingkey=error").Delims("${{", "}}").Parse(str)
	if err != nil {
		return "", err
	}

	var result strings.Builder

	if err := tmpl.Execute(&result, struct {
		OS       string
		ARCH     string
		PLATFORM string
	}{
		OS:       runtime.GOOS,
		ARCH:     runtime.GOARCH,
		PLATFORM: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}); err != nil {
		return "", err
	}

	return result.String(), nil
}

// TemplateWithMap recursively processes a With map and templates all string values
func TemplateWithMap(ctx context.Context, input With, previousOutputs CommandOutputs, withMap With, dry bool) (With, error) {
	if withMap == nil {
		return nil, nil
	}

	result := make(With, len(withMap))
	for k, v := range withMap {
		switch val := v.(type) {
		case string:
			templated, err := TemplateString(ctx, input, previousOutputs, val, dry)
			if err != nil {
				return nil, err
			}
			result[k] = templated
		case map[string]any:
			nestedMap, err := TemplateWithMap(ctx, input, previousOutputs, val, dry)
			if err != nil {
				return nil, err
			}
			result[k] = nestedMap
		case []any:
			templatedSlice, err := templateSlice(ctx, input, previousOutputs, val, dry)
			if err != nil {
				return nil, err
			}
			result[k] = templatedSlice
		default:
			result[k] = v
		}
	}
	return result, nil
}

// templateSlice recursively processes a slice and templates all string values
func templateSlice(ctx context.Context, input With, previousOutputs CommandOutputs, slice []any, dry bool) ([]any, error) {
	result := make([]any, len(slice))
	for i, v := range slice {
		switch val := v.(type) {
		case string:
			templated, err := TemplateString(ctx, input, previousOutputs, val, dry)
			if err != nil {
				return nil, err
			}
			result[i] = templated
		case map[string]any:
			nestedMap, err := TemplateWithMap(ctx, input, previousOutputs, val, dry)
			if err != nil {
				return nil, err
			}
			result[i] = nestedMap
		case []any:
			templatedSlice, err := templateSlice(ctx, input, previousOutputs, val, dry)
			if err != nil {
				return nil, err
			}
			result[i] = templatedSlice
		default:
			result[i] = v
		}
	}
	return result, nil
}

// MergeWithAndParams merges a With map into an InputMap, handling defaults, logging warnings on deprections, etc...
func MergeWithAndParams(ctx context.Context, with With, params InputMap) (With, error) {
	logger := log.FromContext(ctx)
	merged := maps.Clone(with)

	for name, param := range params {
		// the default behavior is that an input is required, this is reflected in the json schema "default" value field
		required := param.Required == nil || (param.Required != nil && *param.Required)

		if _, ok := merged[name]; !ok {
			if required && merged[name] == nil && param.Default == nil && param.DefaultFromEnv == "" {
				return nil, fmt.Errorf("missing required input: %q", name)
			}
			// param.Default and param.DefaultFromEnv are mutually exclusive
			// enforced by JSON schema
			if merged[name] == nil && param.Default != nil {
				merged[name] = param.Default
			}
			if merged[name] == nil && param.DefaultFromEnv != "" {
				val, ok := os.LookupEnv(param.DefaultFromEnv)
				if !ok {
					return nil, fmt.Errorf("environment variable %q not set and no input provided for %q", param.DefaultFromEnv, name)
				}
				merged[name] = val
			}
		}
		// If the input is deprecated AND provided, log a warning
		if param.DeprecatedMessage != "" && with[name] != nil {
			logger.Warnf("input %q is deprecated: %s", name, param.DeprecatedMessage)
		}

		// If the input is provided, and the default is set, ensure the types match, cast otherwise
		if param.Default != nil && with[name] != nil {
			switch param.Default.(type) {
			case bool:
				casted, err := cast.ToE[bool](with[name])
				if err != nil {
					return nil, err
				}
				merged[name] = casted
			case string:
				casted, err := cast.ToE[string](with[name])
				if err != nil {
					return nil, err
				}
				merged[name] = casted
			case int:
				casted, err := cast.ToE[int](with[name])
				if err != nil {
					return nil, err
				}
				merged[name] = casted
			case uint64:
				casted, err := cast.ToE[uint64](with[name])
				if err != nil {
					return nil, err
				}
				merged[name] = casted
			default:
				return nil, fmt.Errorf("unable to cast input %q from %T to %T", name, with[name], param.Default)
			}
		}

		if param.Validate != "" {
			stringified, err := cast.ToE[string](merged[name])
			if err != nil {
				return nil, err
			}

			expr, err := regexp.Compile(param.Validate)
			if err != nil {
				return nil, err
			}

			ok := expr.MatchString(stringified)
			if !ok {
				return nil, fmt.Errorf("failed to validate: input=%s, value=%s, regexp=%s", name, merged[name], param.Validate)
			}
		}
	}

	return merged, nil
}
