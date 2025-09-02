// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package v1

import (
	"regexp"
	"testing"
)

func TestTaskNamePattern(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"foo", true},
		{"foo-bar", true},
		{"foo_bar", true},
		{"foo-bar-1", true},
		{"foo_bar_1", true},
		{"foo1", true},
		{"foo-bar1", true},
		{"0", false},
		{"-foo", false},
		{"1foo", false},
		{"foo@bar", false},
		{"foo bar", false},
		{"_foo", true},
		{"a", true},
		{"foo-bar_baz", true},
		{"", false},
		{"foo--bar", true},
		{"foo__bar", true},
		{"foo-bar_", true},
		{"foo_bar-", true},
		{"foo--bar__baz", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok := TaskNamePattern.MatchString(tc.name)
			if ok != tc.expected {
				t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", tc.name, ok, tc.expected)
			}
		})
	}
}

func FuzzTaskNamePattern(f *testing.F) {
	// Add a variety of initial test cases, including both valid and invalid ones
	testCases := []string{
		"foo",
		"foo-bar",
		"foo_bar",
		"foo-bar-1",
		"foo_bar_1",
		"foo1",
		"foo-bar1",
		"0",             // invalid: single digit / starts with a digit
		"-foo",          // invalid: starts with a dash
		"1foo",          // invalid: starts with a digit
		"foo@bar",       // invalid: contains an illegal character
		"foo bar",       // invalid: contains a space
		"_foo",          // valid: starts with an underscore
		"a",             // valid: single character
		"foo-bar_baz",   // valid: combination of dash and underscore
		"",              // invalid: empty string
		"foo--bar",      // valid: double dash
		"foo__bar",      // valid: double underscore
		"foo-bar_",      // valid: ends with underscore
		"foo_bar-",      // valid: ends with dash
		"foo--bar__baz", // valid: multiple dashes and underscores
	}

	for _, s := range testCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		ok := TaskNamePattern.MatchString(s)
		// Ensure the match result aligns with the pattern's expected behavior
		if len(s) > 0 {
			startsWithValidChar := s[0] == '_' || (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z')
			containsOnlyValidChars := regexp.MustCompile("^[a-zA-Z0-9_-]*$").MatchString(s[1:])

			if startsWithValidChar && containsOnlyValidChars {
				if !ok {
					t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", s, ok, true)
				}
			} else {
				if ok {
					t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", s, ok, false)
				}
			}
		} else {
			if ok {
				t.Errorf("TaskNamePattern.MatchString(%q) = %v, want %v", s, ok, false)
			}
		}
	})
}

func TestEnvVariablePattern(t *testing.T) {
	testCases := []struct {
		name     string
		expected bool
	}{
		{"FOO", true},
		{"_FOO", true},
		{"FOO_BAR", true},
		{"FOO1", true},
		{"_FOO_BAR_1", true},
		{"foo_bar", true},
		{"1FOO", false},
		{"FOO-BAR", false},
		{"FOO@BAR", false},
		{"FOO BAR", false},
		{"FOO$BAR", false},
		{"", false},
		{"FOO__BAR", true},
		{"__FOO", true},
		{"FOO123BAR456", true},
		{"_123FOO", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ok := EnvVariablePattern.MatchString(tc.name)
			if ok != tc.expected {
				t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", tc.name, ok, tc.expected)
			}
		})
	}
}

func FuzzEnvVariablePattern(f *testing.F) {
	// Add a variety of initial test cases, including both valid and invalid ones
	testCases := []string{
		"FOO",
		"_FOO",
		"FOO_BAR",
		"FOO1",
		"_FOO_BAR_1",
		"foo_bar",
		"1FOO",         // invalid: starts with a digit
		"FOO-BAR",      // invalid: contains a dash
		"FOO@BAR",      // invalid: contains an illegal character
		"FOO BAR",      // invalid: contains a space
		"FOO$BAR",      // invalid: contains a dollar sign
		"",             // invalid: empty string
		"FOO__BAR",     // valid: double underscore
		"__FOO",        // valid: starts with double underscore
		"FOO123BAR456", // valid: combination of letters and digits
		"_123FOO",      // valid: starts with underscore followed by digits
	}

	for _, s := range testCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		ok := EnvVariablePattern.MatchString(s)
		// Ensure the match result aligns with the pattern's expected behavior
		if len(s) > 0 {
			startsWithValidChar := (s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z') || s[0] == '_'
			containsOnlyValidChars := regexp.MustCompile("^[a-zA-Z0-9_]*$").MatchString(s[1:])

			if startsWithValidChar && containsOnlyValidChars {
				if !ok {
					t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", s, ok, true)
				}
			} else {
				if ok {
					t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", s, ok, false)
				}
			}
		} else {
			if ok {
				t.Errorf("EnvVariablePattern.MatchString(%q) = %v, want %v", s, ok, false)
			}
		}
	})
}
