// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"fmt"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

// Migrate converts an old workflow to v1 format
func Migrate(oldWorkflow any) (Workflow, error) {

	switch old := oldWorkflow.(type) {
	case v0.Workflow:
		wf := Workflow{
			SchemaVersion: SchemaVersion,
			Tasks:         make(TaskMap),
		}
		// Convert aliases from v0 to v1, structure has not changed but go type has
		if old.Aliases != nil {
			wf.Aliases = make(AliasMap)
			for aliasName, v0Alias := range old.Aliases {
				wf.Aliases[aliasName] = Alias{
					Type:         v0Alias.Type,
					Base:         v0Alias.Base,
					TokenFromEnv: v0Alias.TokenFromEnv,
				}
			}
		}

		// Convert workflow-level inputs from v0 to task-level inputs in v1
		inputs := make(InputMap)
		if old.Inputs != nil {
			for inputName, v0Input := range old.Inputs {
				inputs[inputName] = InputParameter{
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
		for taskName, v0Task := range old.Tasks {
			// Create a separate copy of inputs for each task
			taskInputs := make(InputMap)
			for inputName, v0Input := range inputs {
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

			task := Task{
				// In v1, each task gets a copy of the workflow-level inputs from v0 to keep the same behavior
				Inputs: taskInputs,
				Steps:  v1Steps,
			}
			wf.Tasks[taskName] = task
		}

		return wf, nil

	default:
		return Workflow{}, fmt.Errorf("unsupported type: %T", old)
	}
}
