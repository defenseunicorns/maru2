// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package mcptools

import (
	"context"
	"encoding/json"

	"github.com/defenseunicorns/maru2"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GenJSONSchemaInput struct {
	Version string `json:"version,omitempty" jsonschema:"the workflow schema version to generate, leave empty to generate the meta-schema"`
}

func GenJSONSchema(_ context.Context, _ *mcp.CallToolRequest, input GenJSONSchemaInput) (*mcp.CallToolResult, any, error) {
	schema := maru2.WorkflowSchema(input.Version)

	s, err := json.Marshal(schema)
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(s)},
		},
	}, nil, nil
}
