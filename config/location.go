// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package config provides system-level configuration for maru2
package config

import (
	"os"
	"path/filepath"
)

// DefaultFileName is the default file name for the config file
const DefaultFileName = "config.yaml"

// DefaultDirectory returns the default directory for maru2 configuration ($HOME/.maru2)
//
// Currently this relies upon the $HOME environment variable being set
// In future iterations this may instead leverage the XDG fallback system
func DefaultDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".maru2"), nil
}
