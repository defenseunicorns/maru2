package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
	maru2cmd "github.com/defenseunicorns/maru2/cmd"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())

	ctx := log.WithContext(context.Background(), logger)

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	transport := &mcp.CommandTransport{Command: exec.Command("maru2-mcp-server")}

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
		logger.Print(c.(*mcp.TextContent).Text)
	}
}
