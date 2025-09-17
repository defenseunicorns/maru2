package mcptools

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2"
	v1 "github.com/defenseunicorns/maru2/schema/v1"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ValidateSchemaInput struct {
	Location string `json:"location"`
}

type ValidateSchemaOutput struct {
	Error error `json:"error"`
}

func ValidateSchema(ctx context.Context, req *mcp.CallToolRequest, input ValidateSchemaInput) (*mcp.CallToolResult, ValidateSchemaOutput, error) {
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
