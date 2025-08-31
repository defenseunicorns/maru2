// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"fmt"
	"maps"

	v0 "github.com/defenseunicorns/maru2/schema/v0"
)

// Migrate converts an old workflow to v1 format
func Migrate(oldWorkflow any) (Workflow, error) {

	switch old := oldWorkflow.(type) {
	case v0.Workflow:
		wf := Workflow{
			SchemaVersion: SchemaVersion,
			Tasks:         make(TaskMap, len(old.Tasks)),
		}
		// Convert aliases from v0 to v1, structure has not changed but go type has
		if old.Aliases != nil {
			wf.Aliases = make(AliasMap, len(old.Aliases))
			for aliasName, v0Alias := range old.Aliases {
				wf.Aliases[aliasName] = Alias{
					Type:         v0Alias.Type,
					Base:         v0Alias.Base,
					TokenFromEnv: v0Alias.TokenFromEnv,
				}
			}
		}

		// Convert workflow-level inputs from v0 to task-level inputs in v1
		inputs := make(InputMap, len(old.Inputs))
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
		if old.Inputs == nil {
			inputs = nil
		}

		// Migrate each task from v0 to v1 format
		for taskName, v0Task := range old.Tasks {
			// Create a separate copy of inputs for each task
			taskInputs := make(InputMap, len(inputs))
			maps.Copy(taskInputs, inputs)
			if inputs == nil {
				taskInputs = nil
			}

			// Convert v0 steps ([]Step) to v1 steps
			steps := make([]Step, len(v0Task))
			for i, v0Step := range v0Task {
				steps[i] = Step{
					Run:     v0Step.Run,
					Env:     maps.Clone(v0Step.Env),
					Uses:    v0Step.Uses,
					With:    maps.Clone(v0Step.With),
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
				Inputs: taskInputs,
				Steps:  steps,
			}
			wf.Tasks[taskName] = task
		}

		return wf, nil

	default:
		return Workflow{}, fmt.Errorf("unsupported type: %T", old)
	}
}
