// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package mcptools

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/defenseunicorns/maru2"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
)

// ValidateWorkflowInput represents the input parameters for the validate-workflow MCP tool.
type ValidateWorkflowInput struct {
	From string `json:"from" jsonschema:"Either an absolute path, a relative path from CWD, or a URI detailing the remote location for the workflow"`
}

// ValidateWorkflowOutput represents the output result from the validate-workflow MCP tool.
type ValidateWorkflowOutput struct {
	IsValid bool `json:"is-valid" jsonschema:"whether the resolved and fetched worklow conforms to maru2's JSON schema and other misc structural checks"`
}

// ValidateWorkflow validates a maru2 workflow schema at the specified location.
func ValidateWorkflow(ctx context.Context, _ *mcp.CallToolRequest, input ValidateWorkflowInput) (*mcp.CallToolResult, *ValidateWorkflowOutput, error) {
	logger := log.FromContext(ctx)

	uri, err := uses.ResolveRelative(nil, input.From, nil)
	if err != nil {
		logger.Error(err)
		return nil, nil, err
	}

	svc, err := uses.NewFetcherService()
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

	return nil, &ValidateWorkflowOutput{true}, nil
}
