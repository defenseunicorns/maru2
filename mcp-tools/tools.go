// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package mcptools provides MCP (Model Context Protocol) tool implementations for maru2.
package mcptools

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/defenseunicorns/maru2"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

// ValidateSchemaInput represents the input parameters for the validate-schema MCP tool.
type ValidateSchemaInput struct {
	Location string `json:"location"`
}

// ValidateSchemaOutput represents the output result from the validate-schema MCP tool.
type ValidateSchemaOutput struct {
	Error error `json:"error"`
}

// ValidateSchema validates a maru2 workflow schema at the specified location.
func ValidateSchema(ctx context.Context, _ *mcp.CallToolRequest, input ValidateSchemaInput) (*mcp.CallToolResult, ValidateSchemaOutput, error) {
	logger := log.FromContext(ctx)

	uri, err := uses.ResolveRelative(nil, input.Location, nil)
	if err != nil {
		logger.Error(err)
		return nil, ValidateSchemaOutput{}, err
	}

	svc, err := uses.NewFetcherService(uses.WithFetchPolicy(uses.FetchPolicyAlways))
	if err != nil {
		logger.Error(err)
		return nil, ValidateSchemaOutput{}, err
	}

	wf, err := maru2.Fetch(ctx, svc, uri)
	if err != nil {
		logger.Error(err)
		return nil, ValidateSchemaOutput{}, err
	}

	if err := v1.Validate(wf); err != nil {
		logger.Error(err)
		return nil, ValidateSchemaOutput{Error: err}, nil
	}

	logger.Info("valid workflow", "location", uri)

	return nil, ValidateSchemaOutput{}, nil
}
