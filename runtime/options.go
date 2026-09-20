// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package runtime specifies global runtime configurations and options
package runtime

import (
	"io"
)

// Options are configurations whereupon the default empty/nil value is an acceptable case
// and are not a requirement for a valid runtime execution
//
// Leave this type as a concrete instance vs a pointer type as this object will be mutated along the runtime pipeline,
// and copy by value is the desired behavior versus copy by reference
type Options struct {
	// The WorkingDir to start the execution from, passed to the exec.Command in run blocks, leave blank for current process' PWD
	WorkingDir string
	// Environment variables passed to the exec.Command in run blocks, usually initialized w/ os.Environ()
	Env []string
	// Whether this execution is a dry run or not
	Dry bool
	// Whether this execution is already inside a collapsible section in CI, disables nested collapsible sections if true
	Collapsed bool
	// See `go doc exec.Cmd.Stdout`
	Stdout io.Writer
	// See `go doc exec.Cmd.Stderr`
	Stderr io.Writer
	// See `go doc exec.Cmd.Stdin`
	Stdin io.Reader
}
