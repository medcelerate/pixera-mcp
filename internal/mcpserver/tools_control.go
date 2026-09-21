package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/medcelerate/pixera-mcp/internal/pixera"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerControlTools registers connection/status and the generic call tool.
func registerControlTools(s *mcp.Server, d *deps) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "pixera_status",
		Description: "Report which Pixera server the bridge targets, whether it is reachable, and its API revision.",
		Annotations: annRead("Pixera status"),
	}, d.status)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pixera_set_target",
		Description: "Repoint the bridge at a different Pixera server by host and port. Persists to the config file. The Pixera Native API TCP port defaults to 1400.",
		Annotations: annWrite("Set Pixera target"),
	}, d.setTarget)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pixera_get_api_revision",
		Description: "Return the Pixera API revision number (Pixera.Utility.getApiRevision).",
		Annotations: annRead("API revision"),
	}, d.apiRevision)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pixera_call",
		Description: "Call any Pixera Native API method by its fully-qualified name (e.g. Pixera.Screens.getScreenNames, Pixera.Timelines...) with a JSON params object. Use for methods without a dedicated tool. Class methods require a handle in params.",
		Annotations: annDestructive("Raw Pixera API call"),
	}, d.call)
}

// --- pixera_status ---

func (d *deps) status(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	st := d.app.Status(ctx)
	summary := fmt.Sprintf("Pixera %s — reachable=%v", st.Address, st.Reachable)
	if st.APIRevision != "" {
		summary += " apiRevision=" + st.APIRevision
	}
	if st.Error != "" {
		summary += " (" + st.Error + ")"
	}
	res := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}
	return res, st, nil
}

// --- pixera_set_target ---

type setTargetIn struct {
	Host string `json:"host" jsonschema:"hostname or IP of the Pixera server"`
	Port int    `json:"port,omitempty" jsonschema:"Native API TCP port (default 1400)"`
}

func (d *deps) setTarget(ctx context.Context, _ *mcp.CallToolRequest, in setTargetIn) (*mcp.CallToolResult, any, error) {
	if in.Host == "" {
		return nil, nil, fmt.Errorf("host is required")
	}
	if err := d.app.SetTarget(pixera.Target{Host: in.Host, Port: in.Port}); err != nil {
		return nil, nil, err
	}
	st := d.app.Status(ctx)
	summary := fmt.Sprintf("Target set to %s — reachable=%v", st.Address, st.Reachable)
	if st.Error != "" {
		summary += " (" + st.Error + ")"
	}
	res := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}
	return res, st, nil
}

// --- pixera_get_api_revision ---

func (d *deps) apiRevision(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	raw, err := d.app.Client().Call(ctx, "Pixera.Utility.getApiRevision", map[string]any{})
	if err != nil {
		return nil, nil, err
	}
	return jsonResult("API revision:", raw)
}

// --- pixera_call ---

type callIn struct {
	Method string `json:"method" jsonschema:"fully-qualified method name, e.g. Pixera.Screens.getScreenNames"`
	Params string `json:"params,omitempty" jsonschema:"JSON object of named parameters (e.g. {\"timelineName\":\"Main\",\"mode\":1})"`
}

func (d *deps) call(ctx context.Context, _ *mcp.CallToolRequest, in callIn) (*mcp.CallToolResult, any, error) {
	if in.Method == "" {
		return nil, nil, fmt.Errorf("method is required")
	}
	var params any = map[string]any{}
	if s := in.Params; s != "" {
		if err := json.Unmarshal([]byte(s), &params); err != nil {
			return nil, nil, fmt.Errorf("params is not valid JSON: %w", err)
		}
	}
	raw, err := d.app.Client().Call(ctx, in.Method, params)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(in.Method+":", raw)
}
