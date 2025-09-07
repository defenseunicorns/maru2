// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/go-viper/mapstructure/v2"

	"github.com/defenseunicorns/maru2/builtins"
	"github.com/defenseunicorns/maru2/schema"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
)

// ExecuteBuiltin dispatches to registered builtin tasks (builtin:echo, builtin:fetch)
//
// Strips the "builtin:" prefix, renders templates in the With map,
// then delegates to the appropriate builtin's Execute method
func ExecuteBuiltin(ctx context.Context, step v1.Step, with schema.With, previous CommandOutputs, dry bool) (map[string]any, error) {
	name := strings.TrimPrefix(step.Uses, "builtin:")
	logger := log.FromContext(ctx)

	builtin := builtins.Get(name)
	if builtin == nil {
		return nil, fmt.Errorf("%s not found", step.Uses)
	}

	var rendered schema.With
	if with != nil {
		var err error
		rendered, err = TemplateWithMap(ctx, with, previous, step.With, dry)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", step.Uses, err)
		}
	}

	if dry {
		logger.Info("dry run", "builtin", name)
		printBuiltin(logger, rendered)
		return nil, nil
	}

	if rendered != nil {
		config := &mapstructure.DecoderConfig{
			WeaklyTypedInput: true,
			Result:           &builtin,
		}
		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", step.Uses, err)
		}
		if err := decoder.Decode(rendered); err != nil {
			return nil, fmt.Errorf("%s: %w", step.Uses, err)
		}
	}

	logger.Debug(">", "builtin", name, "with", builtin)

	result, err := builtin.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", step.Uses, err)
	}

	return result, nil
}
