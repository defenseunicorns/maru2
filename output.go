// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package maru2

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// CommandOutputs is a map of step IDs to their outputs.
type CommandOutputs map[string]map[string]any

// ParseOutput parses the output file of a step
//
// Matches behavior of GitHub Actions.
//
// https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#multiline-strings
func ParseOutput(r io.ReadSeeker) (map[string]string, error) {
	if f, ok := r.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}

		// error if larger than 50MB, same limits as GitHub Actions
		if fi.Size() > 50*1024*1024 {
			return nil, fmt.Errorf("output file too large")
		}
	}

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(r)
	result := make(map[string]string)
	var currentKey, currentDelimiter string
	var multiLineValue []string
	var collecting bool

	for scanner.Scan() {
		line := scanner.Text()

		if collecting {
			if line == currentDelimiter {
				// End of multiline value
				value := strings.Join(multiLineValue, "\n")
				result[currentKey] = value
				collecting = false
				multiLineValue = []string{}
				currentKey = ""
				currentDelimiter = ""
			} else {
				multiLineValue = append(multiLineValue, line)
			}
			continue
		}

		if idx := strings.Index(line, "="); idx != -1 {
			// Split the line at the first '=' to handle the key-value pair
			key := line[:idx]
			value := line[idx+1:]
			result[key] = value
		} else if idx := strings.Index(line, "<<"); idx != -1 {
			// Split the line at the first '<<' to handle the key-value pair
			key := line[:idx]
			delimiter := strings.TrimSpace(line[idx+2:])

			if delimiter == "" {
				return nil, fmt.Errorf("invalid syntax: missing delimiter after '<<'")
			}
			currentKey = key
			currentDelimiter = delimiter
			collecting = true
		} else if strings.TrimSpace(line) != "" {
			// Non-empty line without '=' while not collecting a multiline value
			return nil, fmt.Errorf("invalid syntax: non-delimited multiline value")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Handle case where file ends but multiline was being collected
	if collecting {
		return nil, fmt.Errorf("invalid syntax: multiline value not terminated")
	}

	return result, nil
}
