// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package mcptools provides MCP (Model Context Protocol) tool implementations for maru2.
package mcptools

import "github.com/modelcontextprotocol/go-sdk/mcp"

// AddAll registers all of the maru2 specific tools to the given MCP server
func AddAll(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "validate-schema",
		Description: "Used to validate the YAML/JSON schema of a maru2 workflow",
	}, ValidateSchema)
}
