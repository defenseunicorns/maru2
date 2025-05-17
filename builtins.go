// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2/builtins"
	"github.com/go-viper/mapstructure/v2"
)

// ExecuteBuiltin determines which builtin to run based upon the uses string, converts the With map to the expected struct, then calls the builtin's Execute method
func ExecuteBuiltin(ctx context.Context, step Step, with With, previous CommandOutputs, dry bool) (map[string]any, error) {
	name := strings.TrimPrefix(step.Uses, "builtin:")
	logger := log.FromContext(ctx)

	builtin, ok := builtins.Builtins[name]
	if !ok || builtin == nil {
		return nil, fmt.Errorf("%s not found", step.Uses)
	}

	var rendered With
	if with != nil {
		var err error
		rendered, err = TemplateWithMap(ctx, with, previous, step.With, dry)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", step.Uses, err)
		}
	}

	if dry {
		logger.Info("dry run", "builtin", name)
		err := printBuiltin(ctx, rendered)
		if err != nil {
			return nil, err
		}
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

	logger.Print(">", "builtin", name, "with", builtin)

	result, err := builtin.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", step.Uses, err)
	}

	return result, nil
}
