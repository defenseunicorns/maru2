package mcptools

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/charmbracelet/log"
	"github.com/defenseunicorns/maru2"
	"github.com/defenseunicorns/maru2/uses"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DescribeWorkflowInput struct {
	From string `json:"from"      jsonschema:"Either an absolute path, a relative path from CWD, or a URI detailing the remote location for the workflow"`
}

type DescribeOutput struct {
	WorkflowDescription string            `json:"workflow-description"`
	Tasks               map[string]string `json:"tasks"`
}

func DescribeWorkflow(ctx context.Context, _ *mcp.CallToolRequest, input ValidateWorkflowInput) (*mcp.CallToolResult, *DescribeOutput, error) {
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

	out := &DescribeOutput{}

	desc := fmt.Sprintf("%s is schema version %s, has %d tasks, and %d defined aliases", uri, wf.SchemaVersion, len(wf.Tasks), len(wf.Aliases))
	out.WorkflowDescription = desc

	out.Tasks = make(map[string]string, len(wf.Tasks))

	tmpl, err := template.New("task description").Parse(strings.TrimSpace(`
has {{ .Inputs | len }} inputs{{- if .Inputs }}, required: {{- range $name, $input := .Inputs }}{{- if or (not $input.Required) $input.Required }} {{ $name }}{{- end }}{{- end }}{{- end }}

{{- range $i, $v := .Steps }}
- step {{ $i }}{{ if ne $v.Run "" }} is a {{ if $v.Shell }}{{ $v.Shell }}{{ else }}sh{{ end }} script{{ else if ne $v.Uses "" }} uses {{ $v.Uses }}{{ end }}
{{- end }}
`))
	if err != nil {
		return nil, nil, err
	}

	for name, task := range wf.Tasks {
		var buf strings.Builder

		if err := tmpl.Execute(&buf, task); err != nil {
			return nil, nil, err
		}

		out.Tasks[name] = buf.String()
	}

	return nil, out, nil
}
