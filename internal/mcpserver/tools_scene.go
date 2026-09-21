package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerSceneTools registers curated screen, project and layer/parameter
// tools (Pixera.Screens, Pixera.Session, Pixera.Compound).
func registerSceneTools(s *mcp.Server, d *deps) {
	// Screens.
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_list_screens",
		Description: "List the screen names (Pixera.Screens.getScreenNames).",
		Annotations: annRead("List screens")}, d.listScreens)

	// Project / session.
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_save_project",
		Description: "Save the current Pixera project (Pixera.Session.saveProject).",
		Annotations: annWrite("Save project")}, d.saveProject)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_save_project_as",
		Description: "Save the current project to a new path (Pixera.Session.saveProjectAs).",
		Annotations: annWrite("Save project as")}, d.saveProjectAs)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_load_project",
		Description: "Load a Pixera project from a path, replacing the current one (Pixera.Session.loadProject).",
		Annotations: annDestructive("Load project")}, d.loadProject)

	// Layers / parameters / resources.
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_layer_transport",
		Description: "Set a layer's transport mode. Layer path is period-separated, e.g. \"Timeline 1.Layer 1\" (Pixera.Compound.setTransportModeOnLayer).",
		Annotations: annWrite("Layer transport")}, d.layerTransport)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_assign_resource_to_layer",
		Description: "Assign a media resource to a layer. Resource path is slash-separated (e.g. \"Media/Folder/video.mov\"); layer path is period-separated (Pixera.Compound.assignResourceToLayer).",
		Annotations: annWrite("Assign resource")}, d.assignResource)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_set_param",
		Description: "Set a parameter value by period-separated path (e.g. \"Timeline 1.Layer 1.Opacity\") — Pixera.Compound.setParamValue.",
		Annotations: annWrite("Set parameter")}, d.setParam)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_get_param",
		Description: "Get a parameter value by period-separated path (Pixera.Compound.getParamValue).",
		Annotations: annRead("Get parameter")}, d.getParam)
}

func (d *deps) listScreens(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return d.do(ctx, "Screens:", "Pixera.Screens.getScreenNames", map[string]any{})
}

func (d *deps) saveProject(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return d.do(ctx, "Saved project.", "Pixera.Session.saveProject", map[string]any{})
}

type pathIn struct {
	Path string `json:"path" jsonschema:"filesystem path on the Pixera server"`
}

func (d *deps) saveProjectAs(ctx context.Context, _ *mcp.CallToolRequest, in pathIn) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return nil, nil, fmt.Errorf("path is required")
	}
	return d.do(ctx, "Saved project as "+in.Path+":", "Pixera.Session.saveProjectAs", map[string]any{"path": in.Path})
}

func (d *deps) loadProject(ctx context.Context, _ *mcp.CallToolRequest, in pathIn) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return nil, nil, fmt.Errorf("path is required")
	}
	return d.do(ctx, "Loading project "+in.Path+":", "Pixera.Session.loadProject", map[string]any{"path": in.Path})
}

type layerTransportIn struct {
	LayerPath string `json:"layerPath" jsonschema:"period-separated layer path, e.g. Timeline 1.Layer 1"`
	Mode      int    `json:"mode" jsonschema:"transport mode: 1 play, 2 pause, 3 stop"`
	Loop      bool   `json:"loop,omitempty" jsonschema:"loop the layer"`
}

func (d *deps) layerTransport(ctx context.Context, _ *mcp.CallToolRequest, in layerTransportIn) (*mcp.CallToolResult, any, error) {
	if in.LayerPath == "" {
		return nil, nil, fmt.Errorf("layerPath is required")
	}
	return d.do(ctx, fmt.Sprintf("Layer %q mode %d:", in.LayerPath, in.Mode),
		"Pixera.Compound.setTransportModeOnLayer",
		map[string]any{"layerPath": in.LayerPath, "mode": in.Mode, "loop": in.Loop})
}

type assignResourceIn struct {
	ResourcePath string `json:"resourcePath" jsonschema:"slash-separated resource path, e.g. Media/Folder/video.mov"`
	LayerPath    string `json:"layerPath" jsonschema:"period-separated layer path, e.g. Timeline 1.Layer 1"`
}

func (d *deps) assignResource(ctx context.Context, _ *mcp.CallToolRequest, in assignResourceIn) (*mcp.CallToolResult, any, error) {
	if in.ResourcePath == "" || in.LayerPath == "" {
		return nil, nil, fmt.Errorf("resourcePath and layerPath are required")
	}
	return d.do(ctx, fmt.Sprintf("Assigned %q to %q:", in.ResourcePath, in.LayerPath),
		"Pixera.Compound.assignResourceToLayer",
		map[string]any{"resourcePath": in.ResourcePath, "layerPath": in.LayerPath})
}

type setParamIn struct {
	Path  string  `json:"path" jsonschema:"period-separated parameter path, e.g. Timeline 1.Layer 1.Opacity"`
	Value float64 `json:"value" jsonschema:"the value to set"`
}

func (d *deps) setParam(ctx context.Context, _ *mcp.CallToolRequest, in setParamIn) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return nil, nil, fmt.Errorf("path is required")
	}
	return d.do(ctx, fmt.Sprintf("Set %q = %v:", in.Path, in.Value),
		"Pixera.Compound.setParamValue", map[string]any{"path": in.Path, "value": in.Value})
}

type getParamIn struct {
	Path string `json:"path" jsonschema:"period-separated parameter path"`
}

func (d *deps) getParam(ctx context.Context, _ *mcp.CallToolRequest, in getParamIn) (*mcp.CallToolResult, any, error) {
	if in.Path == "" {
		return nil, nil, fmt.Errorf("path is required")
	}
	return d.do(ctx, fmt.Sprintf("%q =", in.Path), "Pixera.Compound.getParamValue", map[string]any{"path": in.Path})
}
