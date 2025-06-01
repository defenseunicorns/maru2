// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package config

import (
	"fmt"

	"github.com/spf13/pflag"
)

// FetchPolicy defines the fetching behavior for the fetcher service
type FetchPolicy string

var _ pflag.Value = (*FetchPolicy)(nil)

const (
	// FetchPolicyAlways will always use the cache if available, never fetching from source
	FetchPolicyAlways FetchPolicy = "always"
	// FetchPolicyIfNotPresent will use the cache if available, otherwise fetch from source
	FetchPolicyIfNotPresent FetchPolicy = "if-not-present"
	// FetchPolicyNever will never use the cache, always fetching from source
	FetchPolicyNever FetchPolicy = "never"
	// DefaultFetchPolicy is the default fetch policy used when none is specified
	DefaultFetchPolicy FetchPolicy = FetchPolicyIfNotPresent
)

// AvailablePolicies returns a list of available fetch policies
func AvailablePolicies() []string {
	return []string{
		string(FetchPolicyAlways),
		string(FetchPolicyIfNotPresent),
		string(FetchPolicyNever),
	}
}

// String implements the pflag.Value and fmt.Stringer interfaces
func (f *FetchPolicy) String() string {
	return string(*f)
}

// Set implements the pflag.Value interface
func (f *FetchPolicy) Set(value string) error {
	switch value {
	case string(FetchPolicyAlways):
		*f = FetchPolicyAlways
	case string(FetchPolicyIfNotPresent):
		*f = FetchPolicyIfNotPresent
	case string(FetchPolicyNever):
		*f = FetchPolicyNever
	default:
		return fmt.Errorf("invalid fetch policy: %s", value)
	}
	return nil
}

// Type implements the pflag.Value interface
func (f *FetchPolicy) Type() string {
	return "string"
}
