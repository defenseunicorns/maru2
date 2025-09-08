// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package keycloak provides builtins for interacting with Keycloak admin APIs
package keycloak

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"github.com/charmbracelet/log"
)

const (
	// DefaultURL is the default Keycloak admin URL
	DefaultURL = "https://keycloak.admin.uds.dev"
	// DefaultRealm is the default Keycloak realm name
	DefaultRealm = "uds"
)

// CreateGroup is a builtin for creating Keycloak groups
type CreateGroup struct {
	KeycloakURL   string `json:"keycloak-url,omitempty" jsonschema:"description=Base URL for Keycloak,default=https://keycloak.admin.uds.dev"`
	Realm         string `json:"realm,omitempty"        jsonschema:"description=Keycloak realm name,default=uds"`
	AdminUsername string `json:"admin-username"         jsonschema:"description=Admin username"`
	AdminPassword string `json:"admin-password"         jsonschema:"description=Admin password"`
	Group         string `json:"group"                  jsonschema:"description=Name of group to create"`
}

// Execute the builtin
func (b *CreateGroup) Execute(ctx context.Context) (map[string]any, error) {
	b.setDefaults()
	logger := log.FromContext(ctx)
	client := gocloak.NewClient(b.KeycloakURL)

	// Access token via password grant
	token, err := client.LoginAdmin(ctx, b.AdminUsername, b.AdminPassword, b.Realm)
	if err != nil {
		return nil, fmt.Errorf("failed logging into admin user %q in realm %q: %w", b.AdminUsername, b.Realm, err)
	}

	// Check if group already exists
	existing, err := client.GetGroup(ctx, token.AccessToken, b.Realm, b.Group)
	if err != nil {
		return nil, fmt.Errorf("failed checking group %q exists in realm %q: %w", b.Group, b.Realm, err)
	}

	if existing.ID != nil && *existing.ID == b.Group {
		logger.Infof("Group %q already exists with id %q", b.Group, *existing.ID)
		return map[string]any{"id": *existing.ID}, nil
	}

	// Create group and get new ID
	group := gocloak.Group{
		Name: gocloak.StringP(b.Group),
	}

	id, err := client.CreateGroup(ctx, token.AccessToken, b.Realm, group)
	if err != nil {
		return nil, fmt.Errorf("failed creating group %q in realm %q: %w", b.Group, b.Realm, err)
	}

	return map[string]any{"id": id}, nil
}

func (b *CreateGroup) setDefaults() {
	if b.KeycloakURL == "" {
		b.KeycloakURL = DefaultURL
	}

	if b.Realm == "" {
		b.Realm = DefaultRealm
	}
}
