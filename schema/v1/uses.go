// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

// SupportedSchemes returns a list of supported schemes
func SupportedSchemes() []string {
	return []string{"file", "http", "https", "pkg", "oci"}
}
