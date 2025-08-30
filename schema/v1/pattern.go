// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v0

import "regexp"

// TaskNamePattern is a regular expression for valid task names, it is also used for step IDs
var TaskNamePattern = regexp.MustCompile("^[_a-zA-Z][a-zA-Z0-9_-]*$")

// InputNamePattern is a regular expression for valid input names
var InputNamePattern = TaskNamePattern // regexp.MustCompile("^\\$[A-Z_]+[A-Z0-9_]*$")

// EnvVariablePattern is a regular expression for valid environment variable names
var EnvVariablePattern = regexp.MustCompile("^[a-zA-Z_]+[a-zA-Z0-9_]*$")
