// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package mcptools

import (
	"context"
	"net/url"

	"github.com/charmbracelet/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/defenseunicorns/maru2"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

// ValidateSchemaInput represents the input parameters for the validate-schema MCP tool.
type ValidateSchemaInput struct {
	ProjectRoot string `json:"cwd,omitempty" jsonschema:"The calling client's project root (usually a file:/// absolute path to a local directory), needed as location can be a relative file location"`
	Location    string `json:"location"      jsonschema:"Either a relative path, or a URI detailing the remote location for the workflow"`
}

// ValidateSchemaOutput represents the output result from the validate-schema MCP tool.
type ValidateSchemaOutput struct {
	IsValid bool `json:"is-valid"`
}

// ValidateSchema validates a maru2 workflow schema at the specified location.
func ValidateSchema(ctx context.Context, _ *mcp.CallToolRequest, input ValidateSchemaInput) (*mcp.CallToolResult, *ValidateSchemaOutput, error) {
	logger := log.FromContext(ctx)

	var root *url.URL

	if input.ProjectRoot != "" {
		var err error
		root, err = url.Parse(input.ProjectRoot)
		if err != nil {
			logger.Error(err)
			return nil, nil, err
		}
		// TODO: ensure root is an ABSOLUTE file URL
	}

	uri, err := uses.ResolveRelative(root, input.Location, nil)
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	svc, err := uses.NewFetcherService(uses.WithFetchPolicy(uses.FetchPolicyAlways))
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	wf, err := maru2.Fetch(ctx, svc, uri)
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	if err := v1.Validate(wf); err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	logger.Info("valid workflow", "location", uri)

	return nil, &ValidateSchemaOutput{true}, nil
}
