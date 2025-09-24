// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2025-Present Defense Unicorns

package mcptools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var prompts = map[*mcp.Prompt]mcp.PromptHandler{
	&mcp.Prompt{}: func(ctx context.Context, gpr *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return nil, nil
	},
}

type BestPractices struct{}

func (BestPractices) Prompt() *mcp.Prompt

func (BestPractices) Handler() *mcp.PromptHandler
