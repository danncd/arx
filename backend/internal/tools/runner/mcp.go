package runner

import (
	mcp "arx/internal/integrations/mcp"
	permission "arx/internal/permissions"
	tool "arx/internal/tools"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"strings"
)

type mcpInput struct {
	Operation string          `json:"operation"`
	Server    string          `json:"server"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (r *Runner) prepareMCP(ctx context.Context, raw json.RawMessage) (tool.Prepared, error) {
	if r.mcp == nil {
		return tool.Prepared{}, errors.New("MCP connections are unavailable")
	}
	var in mcpInput
	if err := tool.Decode(raw, &in); err != nil {
		return tool.Prepared{}, err
	}
	if in.Operation == "list" && in.Server == "" {
		return tool.Prepared{Metadata: true, Action: permission.Action{Tool: "mcp", Operation: "list"}, Run: func(context.Context) (tool.Result, error) {
			entries := []map[string]any{}
			for _, s := range r.mcp.State() {
				if s.Enabled {
					entries = append(entries, map[string]any{"id": s.ID, "name": s.Name, "status": s.Status})
				}
			}
			body, _ := json.Marshal(map[string]any{"kind": "mcp_server_catalog", "servers": entries})
			return tool.Output(string(body)), nil
		}}, nil
	}
	if in.Operation != "list" && in.Operation != "describe" && in.Operation != "call" {
		return tool.Prepared{}, errors.New("MCP operation must be list, describe, or call. A server tool name goes in name, not operation. Discover tools with {\"operation\":\"list\",\"server\":\"SERVER_ID\"}, inspect the schema with {\"operation\":\"describe\",\"server\":\"SERVER_ID\",\"name\":\"TOOL_NAME\"}, then execute with {\"operation\":\"call\",\"server\":\"SERVER_ID\",\"name\":\"TOOL_NAME\",\"arguments\":{}}.")
	}
	action := permission.Action{Tool: "mcp", Operation: in.Operation, Server: in.Server, Path: in.Name, Arguments: in.Arguments}
	var config mcp.Configuration
	var definition mcp.Tool
	if in.Operation == "call" {
		var object map[string]json.RawMessage
		if len(in.Arguments) == 0 {
			in.Arguments = json.RawMessage(`{}`)
		}
		if len(in.Arguments) > 256<<10 || json.Unmarshal(in.Arguments, &object) != nil || object == nil {
			return tool.Prepared{}, errors.New("MCP arguments must be a JSON object up to 256 KiB")
		}
		var err error
		config, definition, err = r.mcp.Action(ctx, in.Server, in.Name)
		if err != nil {
			return tool.Prepared{}, err
		}
		var schema jsonschema.Schema
		if err := json.Unmarshal(definition.InputSchema, &schema); err != nil {
			return tool.Prepared{}, errors.New("MCP tool has an invalid input schema")
		}
		resolved, err := schema.Resolve(nil)
		if err != nil {
			return tool.Prepared{}, errors.New("MCP tool input schema cannot be resolved locally")
		}
		var args any
		if err := json.Unmarshal(in.Arguments, &args); err != nil {
			return tool.Prepared{}, err
		}
		if err := resolved.Validate(args); err != nil {
			return tool.Prepared{}, errors.New("Arguments do not match the MCP tool schema: " + err.Error())
		}
		action.Arguments = in.Arguments
		action.Writes = !definition.ReadOnly
	}
	p := tool.Prepared{Action: action, RequireApproval: in.Operation == "call" && (config.Policy == "ask" || !definition.ReadOnly), Metadata: in.Operation != "call", Run: func(ctx context.Context) (tool.Result, error) {
		if in.Operation != "call" {
			definitions, err := r.mcp.Inspect(ctx, in.Server)
			if err != nil {
				return tool.Result{}, err
			}
			if in.Operation == "list" {
				entries := []map[string]string{}
				for _, t := range definitions {
					entries = append(entries, map[string]string{"name": t.Name, "description": t.Description})
				}
				body, _ := json.Marshal(map[string]any{"kind": "mcp_tool_catalog", "server": in.Server, "tools": entries})
				return tool.Output(string(body)), nil
			}
			for _, t := range definitions {
				if t.Name == in.Name {
					body, _ := json.Marshal(map[string]any{"kind": "mcp_tool_schema", "server": in.Server, "tool": t})
					return tool.Output(string(body)), nil
				}
			}
			return tool.Result{}, errors.New("MCP tool was not found")
		}
		result, err := r.mcp.Call(ctx, in.Server, in.Name, in.Arguments, conversation(ctx))
		if result == nil {
			return tool.Result{}, err
		}
		output, conversionErr := r.mcpResult(result)
		return output, errors.Join(err, conversionErr)
	}}
	if in.Operation == "call" {
		p.Denied = func(ctx context.Context) error { return r.mcp.Denied(in.Server, in.Name, conversation(ctx)) }
	}
	return p, nil
}
func (r *Runner) mcpResult(body *sdk.CallToolResult) (tool.Result, error) {
	texts := []string{}
	images := tool.Result{}
	if body.StructuredContent != nil {
		b, err := json.Marshal(body.StructuredContent)
		if err != nil {
			return tool.Result{}, err
		}
		texts = append(texts, string(b))
	}
	for _, c := range body.Content {
		switch v := c.(type) {
		case *sdk.TextContent:
			texts = append(texts, v.Text)
		case *sdk.ImageContent:
			if r.images == nil || len(v.Data) > 8<<20 {
				return tool.Result{}, errors.New("MCP image cannot be stored")
			}
			saved, err := r.images.Save(v.Data, "MCP image")
			if err != nil {
				return tool.Result{}, err
			}
			images.Images = append(images.Images, saved)
		default:
			b, err := json.Marshal(c)
			if err != nil {
				return tool.Result{}, err
			}
			texts = append(texts, string(b))
		}
	}
	out := tool.Output(strings.Join(texts, "\n\n"))
	out.Images = images.Images
	out.Failed = body.IsError
	return out, nil
}
