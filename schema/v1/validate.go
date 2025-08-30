// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/xeipuuv/gojsonschema"

	"github.com/defenseunicorns/maru2/schema"
)

// Read reads a workflow from a file
func Read(r io.Reader) (Workflow, error) {
	if rs, ok := r.(io.Seeker); ok {
		_, err := rs.Seek(0, io.SeekStart)
		if err != nil {
			return Workflow{}, err
		}
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return Workflow{}, err
	}

	var versioned schema.Versioned
	if err := yaml.Unmarshal(data, &versioned); err != nil {
		return Workflow{}, err
	}

	switch version := versioned.SchemaVersion; version {
	case SchemaVersion:
		var wf Workflow
		return wf, yaml.Unmarshal(data, &wf)
	default:
		return Workflow{}, fmt.Errorf("unsupported schema version: expected %q, got %q", SchemaVersion, version)
	}
}

var schemaOnce = sync.OnceValues(func() (string, error) {
	s := WorkFlowSchema()
	b, err := json.Marshal(s)
	return string(b), err
})

// Validate validates a workflow
func Validate(wf Workflow) error {
	if len(wf.Tasks) == 0 {
		return errors.New("no tasks available")
	}

	for name, task := range wf.Tasks {
		if ok := TaskNamePattern.MatchString(name); !ok {
			return fmt.Errorf("task name %q does not satisfy %q", name, TaskNamePattern.String())
		}

		ids := make(map[string]int, len(task.Steps))

		for idx, step := range task.Steps {
			// ensure that only one of run or uses fields is set
			switch {
			// both
			case step.Uses != "" && step.Run != "":
				return fmt.Errorf(".tasks.%s[%d] has both run and uses fields set", name, idx)
			// neither
			case step.Uses == "" && step.Run == "":
				return fmt.Errorf(".tasks.%s[%d] must have one of [run, uses] fields set", name, idx)
			}

			if step.ID != "" {
				if ok := TaskNamePattern.MatchString(step.ID); !ok {
					return fmt.Errorf(".tasks.%s[%d].id %q does not satisfy %q", name, idx, step.ID, TaskNamePattern.String())
				}

				if _, ok := ids[step.ID]; ok {
					return fmt.Errorf(".tasks.%s[%d] and .tasks.%s[%d] have the same ID %q", name, ids[step.ID], name, idx, step.ID)
				}
				ids[step.ID] = idx
			}

			if step.Uses != "" {
				u, err := url.Parse(step.Uses)
				if err != nil {
					return fmt.Errorf(".tasks.%s[%d].uses %w", name, idx, err)
				}

				if u.Scheme == "" {
					if step.Uses == name {
						return fmt.Errorf(".tasks.%s[%d].uses cannot reference itself", name, idx)
					}
					_, ok := wf.Tasks.Find(step.Uses)
					if !ok {
						return fmt.Errorf(".tasks.%s[%d].uses %q not found", name, idx, step.Uses)
					}
				} else {
					schemes := append(SupportedSchemes(), "builtin")

					if !slices.Contains(schemes, u.Scheme) {
						return fmt.Errorf(".tasks.%s[%d].uses %q is not one of [%s]", name, idx, u.Scheme, strings.Join(schemes, ", "))
					}
				}
			}

			if step.Dir != "" {
				if filepath.IsAbs(step.Dir) {
					return fmt.Errorf(".tasks.%s[%d].dir %q must not be absolute", name, idx, step.Dir)
				}
			}

			if step.Timeout != "" {
				_, err := time.ParseDuration(step.Timeout)
				if err != nil {
					return fmt.Errorf(".tasks.%s[%d].timeout %q is not a valid time duration", name, idx, step.Timeout)
				}
			}

			for envName := range step.Env {
				if ok := EnvVariablePattern.MatchString(envName); !ok {
					return fmt.Errorf(".tasks.%s[%d].env %q does not satisfy %q", name, idx, envName, EnvVariablePattern.String())
				}
			}
			for inputName, param := range task.Inputs {
				if ok := InputNamePattern.MatchString(inputName); !ok {
					return fmt.Errorf(".tasks.%s.inputs.%s %q does not satisfy %q", name, inputName, inputName, InputNamePattern.String())
				}

				if param.Validate != "" {
					_, err := regexp.Compile(param.Validate)
					if err != nil {
						return fmt.Errorf(".tasks.%s.inputs.%s: %v", name, inputName, err)
					}
				}
			}
		}
	}

	schema, err := schemaOnce()
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(schema)

	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewGoLoader(wf))
	if err != nil {
		return err
	}

	if result.Valid() {
		return nil
	}

	var resErr error
	for _, err := range result.Errors() {
		resErr = errors.Join(resErr, errors.New(err.String()))
	}

	return resErr
}

// ReadAndValidate reads and validates a workflow
func ReadAndValidate(r io.Reader) (Workflow, error) {
	wf, err := Read(r)
	if err != nil {
		return Workflow{}, err
	}
	return wf, Validate(wf)
}
