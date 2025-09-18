// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main provides an MCP server/client that exposes maru2 functionality via the Model Context Protocol.
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

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

	// logger.Warn("this program is currently marked ALPHA and is subject to breaking changes w/o warning")

	// later do this w/ cobra commands, but let's keep it simple for now
	switch mode {
	case "client":
		clientFlags := flag.NewFlagSet("mcp-client", flag.ExitOnError)

		var s string
		clientFlags.StringVar(&s, "s", "", "The address and port of the maru2-mcp server (example: http://localhost:4371)")

		if err := clientFlags.Parse(os.Args[2:]); err != nil {
			logger.Fatal(err)
		}

		client := mcp.NewClient(&mcp.Implementation{Name: "maru2-mcp-client", Version: "v1.0.0"}, nil)

		var transport mcp.Transport
		transport = &mcp.StreamableClientTransport{Endpoint: s}

		if s == "" {
			self, err := os.Executable()
			if err != nil {
				logger.Fatal(err)
			}
			self, err = filepath.EvalSymlinks(self)
			if err != nil {
				logger.Fatal(err)
			}

			command := exec.Command(self, "cli")
			command.Stderr = os.Stderr // used for debugging using the logger

			transport = &mcp.CommandTransport{Command: command}
		}

		session, err := client.Connect(ctx, transport, nil)
		if err != nil {
			logger.Fatal(err)
		}
		defer session.Close()

		// call validate
		params := &mcp.CallToolParams{
			Name:      "validate-workflow",
			Arguments: map[string]any{"location": "file:testdata/simple.yaml"},
		}
		res, err := session.CallTool(ctx, params)
		if err != nil {
			logger.Fatalf("CallTool failed: %v", err)
		}
		if res.IsError {
			for _, c := range res.Content {
				logger.Error(c.(*mcp.TextContent).Text)
			}
			logger.Fatal("tool failed")
		}
		for _, c := range res.Content {
			// assume its a text content for now
			tc := c.(*mcp.TextContent)
			logger.Info(params.Name, "args", params.Arguments, "result", tc.Text)
		}
	case "server":
		impl := &mcp.Implementation{Name: "maru2-mcp-server", Version: "v1.0.0"}
		server := mcp.NewServer(impl, nil)
		mcptools.AddAll(server)

		server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
			return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
				ctx = log.WithContext(ctx, logger)
				return next(ctx, method, req)
			}
		})

		handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
			return server
		}, nil)

		srv := &http.Server{
			Addr:    "0.0.0.0:4371",
			Handler: handler,
		}

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt)

		go func() {
			logger.Info("listening", "addr", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalf("ListenAndServe: %v", err)
			}
		}()

		<-stop
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatalf("Shutdown error: %v", err)
		}

	case "cli":
		impl := &mcp.Implementation{Name: "maru2-mcp-cli", Version: "v1.0.0"}
		server := mcp.NewServer(impl, nil)
		mcptools.AddAll(server)
		if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
			logger.Fatal(err)
		}
	default:
		logger.Fatal("must specify 'client', 'server', or 'cli' as first argument")
	}
}
