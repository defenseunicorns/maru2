// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package uses

import (
	"net/url"
	"strings"

	"github.com/spf13/pflag"
)

var _ pflag.Value = &URI{}

// URI is a thin wrapper around *url.URL
// created to implement pflag.Value
type URI struct {
	*url.URL
}

// Parse is essentially a url.Parse
func Parse(value string) (*URI, error) {
	uri := &URI{}
	return uri, uri.Set(value)
}

// String implements pflag.Value and fmt.Stringer
func (u *URI) String() string {
	return u.URL.String()
}

// Type implements pflag.Value
func (u *URI) Type() string {
	return "uri"
}

// Set implements pflag.Value
func (u *URI) Set(value string) error {
	// fix fish needing "'pkg:...'" for tab completion
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)

	parsedURL, err := url.Parse(value)
	if err != nil {
		return err
	}
	u.URL = parsedURL
	return nil
}
