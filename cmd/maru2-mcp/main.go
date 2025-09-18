// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main provides an MCP server/client that exposes maru2 functionality via the Model Context Protocol.
package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	maru2cmd "github.com/defenseunicorns/maru2/cmd"
	mcptools "github.com/defenseunicorns/maru2/mcp-tools"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		Level:           log.DebugLevel,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())

	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	logger = logger.WithPrefix(mode)
	ctx := log.WithContext(context.Background(), logger)

	// later do this w/ cobra commands, but let's keep it simple for now
	switch mode {
	case "client":
		client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

		command := exec.Command("maru2-mcp", "cli")
		command.Stderr = os.Stderr // used for debugging using the logger

		transport := &mcp.CommandTransport{Command: command}

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			logger.Fatal(err)
		}
		defer session.Close()

		// call validate
		params := &mcp.CallToolParams{
			Name:      "validate-schema",
			Arguments: map[string]any{"location": "testdata/simple.yaml"},
		}
		res, err := session.CallTool(ctx, params)
		if err != nil {
			logger.Fatalf("CallTool failed: %v", err)
		}
		if res.IsError {
			logger.Fatal("tool failed")
		}
		for _, c := range res.Content {
			logger.Info(c.(*mcp.TextContent).Text)
		}
	case "server":
		logger.Fatal("not implemented")
	case "cli":
		server := mcp.NewServer(&mcp.Implementation{Name: "maru2", Version: "v1.0.0"}, nil)

		mcp.AddTool(server, &mcp.Tool{Name: "validate-schema", Description: "Used to validate the YAML/JSON schema of a maru2 workflow"}, mcptools.ValidateSchema)
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			logger.Fatal(err)
		}
	default:
		logger.Fatal("must specify 'client', 'server', or 'cli' as first argument")
	}
}
