// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

// Package main provides an MCP server/client that exposes maru2 functionality via the Model Context Protocol.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	maru2cmd "github.com/defenseunicorns/maru2/cmd"
	mcptools "github.com/defenseunicorns/maru2/mcp-tools"
)

func main() {
	var format string

	root := &cobra.Command{
		Use: "maru2-mcp",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			logger := log.FromContext(cmd.Context())
			switch format {
			case "", "text":
				logger.SetFormatter(log.TextFormatter)
			case "json":
				logger.SetFormatter(log.JSONFormatter)
			default:
				return fmt.Errorf("invalid output format: %s", format)
			}
			logger = logger.WithPrefix(cmd.Name())
			cmd.SetContext(log.WithContext(cmd.Context(), logger))
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVarP(&format, "output", "o", "text", "output format (text|json)")

	root.AddCommand(newClientCmd(), newServerCmd(), newCLICmd())

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		Level:           log.DebugLevel,
	})

	logger.SetStyles(maru2cmd.DefaultStyles())

	ctx := log.WithContext(context.Background(), logger)

	if err := root.ExecuteContext(ctx); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func newClientCmd() *cobra.Command {
	var s string

	command := &cobra.Command{
		Use: "client",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			logger := log.FromContext(ctx)

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

				// purposefully ignoring the error here
				o, _ := cmd.Flags().GetString("output")

				command := exec.Command(self, "cli", "-o", o)
				command.Stderr = os.Stderr // used for debugging using the logger

				transport = &mcp.CommandTransport{Command: command}
			}

			session, err := client.Connect(ctx, transport, nil)
			if err != nil {
				logger.Fatal(err)
			}
			defer session.Close()

			// TODO: currently hardcoded call for loopback testing purposes, should be abstracted into a cobra command(s)
			params := &mcp.CallToolParams{
				Name:      "describe-workflow",
				Arguments: map[string]any{"from": "file:testdata/simple.yaml"},
			}
			res, err := session.CallTool(ctx, params)
			if err != nil {
				return fmt.Errorf("CallTool failed: %v", err)
			}
			if res.IsError {
				for _, c := range res.Content {
					logger.Error(c.(*mcp.TextContent).Text)
				}
				return fmt.Errorf("tool failed")
			}
			for _, c := range res.Content {
				// assume its a text content for now
				tc := c.(*mcp.TextContent)
				logger.Info(params.Name, "args", params.Arguments)
				// goes to STDOUT
				// for best printing, do:
				// make -j all install
				// ./bin/maru2-mcp client | jq
				fmt.Println(tc.Text)
			}
			return nil
		},
	}

	command.Flags().StringVarP(&s, "server", "s", "", "The scheme, address and port of the maru2-mcp server (example: http://localhost:4371)")

	return command
}

func newServerCmd() *cobra.Command {
	var addr string

	command := &cobra.Command{
		Use: "server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			logger := log.FromContext(ctx)
			impl := &mcp.Implementation{Name: "maru2-mcp-server", Version: "v1.0.0"}
			server := mcp.NewServer(impl, nil)
			mcptools.AddAll(server)

			server.AddReceivingMiddleware(
				// this middleware only adds the logger onto the request's context
				func(next mcp.MethodHandler) mcp.MethodHandler {
					return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
						ctx = log.WithContext(ctx, logger)
						return next(ctx, method, req)
					}
				},
				loggingMiddleware,
			)

			handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
				return server
			}, nil)

			srv := &http.Server{
				Addr:    addr,
				Handler: handler,
			}

			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)

			go func() {
				logger.Info("listening", "addr", srv.Addr)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Errorf("ListenAndServe: %v", err)
					os.Exit(1)
				}
			}()

			<-stop
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				return fmt.Errorf("Shutdown error: %v", err)
			}
			return nil
		},
	}

	command.Flags().StringVarP(&addr, "address", "a", "0.0.0.0:4371", "The address and port of the maru2-mcp server")

	return command
}

func newCLICmd() *cobra.Command {
	command := &cobra.Command{
		Use: "cli",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			impl := &mcp.Implementation{Name: "maru2-mcp-cli", Version: "v1.0.0"}
			server := mcp.NewServer(impl, nil)
			mcptools.AddAll(server)
			server.AddReceivingMiddleware(loggingMiddleware)
			return server.Run(ctx, &mcp.StdioTransport{})
		},
	}
	return command
}

func loggingMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(
		ctx context.Context,
		method string,
		req mcp.Request,
	) (mcp.Result, error) {
		logger := log.FromContext(ctx)

		msg := []any{
			"method", method,
			"session-id", req.GetSession().ID(),
			"has-params", req.GetParams() != nil,
		}

		var toolName string
		if ctr, ok := req.(*mcp.CallToolRequest); ok {
			toolName = ctr.Params.Name
			msg = append(msg, "tool-name", toolName)
		}

		logger.Info("start", msg...)

		start := time.Now()

		result, err := next(ctx, method, req)

		duration := time.Since(start)

		if err != nil {
			logger.Error("failed",
				"method", method,
				"session-id", req.GetSession().ID(),
				"duration-ms", duration.Milliseconds(),
				"err", err,
			)
		} else {
			msg := []any{
				"method", method,
				"session-id", req.GetSession().ID(),
				"duration-ms", duration.Milliseconds(),
			}
			if ctr, ok := result.(*mcp.CallToolResult); ok {
				msg = append(msg, "tool-name", toolName)
				msg = append(msg, "isError", ctr.IsError)
			}
			logger.Info("completed", msg...)
		}

		return result, err
	}
}
