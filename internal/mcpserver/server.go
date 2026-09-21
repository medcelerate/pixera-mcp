// Package mcpserver builds the Model Context Protocol server that exposes the
// Pixera Native API as tools: a curated set of typed control tools plus a
// generic pixera_call for any other method.
package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/medcelerate/pixera-mcp/internal/app"
	"github.com/medcelerate/pixera-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// deps carries shared dependencies for tool handlers.
type deps struct {
	app *app.App
}

// do calls a Pixera method with named params and returns a tool result.
func (d *deps) do(ctx context.Context, summary, method string, params map[string]any) (*mcp.CallToolResult, any, error) {
	raw, err := d.app.Client().Call(ctx, method, params)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(summary, raw)
}

// New builds an MCP server with Pixera control tools.
func New(a *app.App) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "pixera-mcp",
		Version: version.Version,
	}, nil)

	d := &deps{app: a}
	registerControlTools(s, d)
	registerTimelineTools(s, d)
	registerSceneTools(s, d)
	return s
}

// jsonResult builds a tool result: a text summary plus the raw JSON result, and
// the decoded JSON as structured content.
func jsonResult(summary string, raw json.RawMessage) (*mcp.CallToolResult, any, error) {
	text := summary
	if len(raw) > 0 && string(raw) != "null" {
		text = summary + " " + string(raw)
	}
	res := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
	var v any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	return res, v, nil
}

func boolPtr(b bool) *bool { return &b }

func annRead(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true}
}
func annWrite(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, DestructiveHint: boolPtr(false)}
}
func annDestructive(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, DestructiveHint: boolPtr(true)}
}
