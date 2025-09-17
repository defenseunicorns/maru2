package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	maru2cmd "github.com/defenseunicorns/maru2/cmd"
	mcptools "github.com/defenseunicorns/maru2/mcp-tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())

	ctx := log.WithContext(context.Background(), logger)

	server := mcp.NewServer(&mcp.Implementation{Name: "maru2", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "validate-schema", Description: "Used to validate the YAML/JSON schema of a maru2 workflow"}, mcptools.ValidateSchema)

	logger.Info("running maru2-mcp-server", "transport", "stdio")
	// Run the server over stdin/stdout, until the client disconnects
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.Fatal(err)
	}
}
