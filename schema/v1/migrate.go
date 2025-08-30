// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"fmt"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

// Migrate converts a v0 workflow to v1 format
func Migrate(old any) (Workflow, error) {
	v0Workflow, ok := old.(v0.Workflow)
	if !ok {
		return Workflow{}, fmt.Errorf("expected v0.Workflow, got %T", old)
	}

	// Create the new v1 workflow
	v1Workflow := Workflow{
		SchemaVersion: SchemaVersion,
		Tasks:         make(TaskMap),
	}

	// Convert aliases from v0 to v1
	if v0Workflow.Aliases != nil {
		v1Workflow.Aliases = make(AliasMap)
		for aliasName, v0Alias := range v0Workflow.Aliases {
			v1Workflow.Aliases[aliasName] = Alias{
				Type:         v0Alias.Type,
				Base:         v0Alias.Base,
				TokenFromEnv: v0Alias.TokenFromEnv,
			}
		}
	}

	// Convert workflow-level inputs from v0 to task-level inputs in v1
	v1Inputs := make(InputMap)
	if v0Workflow.Inputs != nil {
		for inputName, v0Input := range v0Workflow.Inputs {
			v1Inputs[inputName] = InputParameter{
				Description:       v0Input.Description,
				DeprecatedMessage: v0Input.DeprecatedMessage,
				Required:          v0Input.Required,
				Default:           v0Input.Default,
				DefaultFromEnv:    v0Input.DefaultFromEnv,
				Validate:          v0Input.Validate,
			}
		}
	}

	// Migrate each task from v0 to v1 format
	for taskName, v0Task := range v0Workflow.Tasks {
		// Create a separate copy of inputs for each task
		taskInputs := make(InputMap)
		for inputName, v0Input := range v1Inputs {
			taskInputs[inputName] = InputParameter{
				Description:       v0Input.Description,
				DeprecatedMessage: v0Input.DeprecatedMessage,
				Required:          v0Input.Required,
				Default:           v0Input.Default,
				DefaultFromEnv:    v0Input.DefaultFromEnv,
				Validate:          v0Input.Validate,
			}
		}

		// Convert v0 steps ([]Step) to v1 steps
		v1Steps := make([]Step, len(v0Task))
		for i, v0Step := range v0Task {
			// Convert environment variables
			var v1Env Env
			if v0Step.Env != nil {
				v1Env = make(Env)
				for envName, envValue := range v0Step.Env {
					v1Env[envName] = envValue
				}
			}

			// Convert with parameters
			var v1With With
			if v0Step.With != nil {
				v1With = make(With)
				for withName, withValue := range v0Step.With {
					v1With[withName] = withValue
				}
			}

			v1Steps[i] = Step{
				Run:     v0Step.Run,
				Env:     v1Env,
				Uses:    v0Step.Uses,
				With:    v1With,
				ID:      v0Step.ID,
				Name:    v0Step.Name,
				If:      v0Step.If,
				Dir:     v0Step.Dir,
				Shell:   v0Step.Shell,
				Timeout: v0Step.Timeout,
				Mute:    v0Step.Mute,
			}
		}

		v1Task := Task{
			// In v1, each task gets a copy of the workflow-level inputs from v0
			Inputs: taskInputs,
			Steps:  v1Steps,
		}
		v1Workflow.Tasks[taskName] = v1Task
	}

	return v1Workflow, nil
}
