package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Pixera transport modes.
const (
	modePlay  = 1
	modePause = 2
	modeStop  = 3
)

// registerTimelineTools registers curated timeline transport tools built on the
// Pixera.Compound namespace (name/index addressed, no handles required).
func registerTimelineTools(s *mcp.Server, d *deps) {
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_play",
		Description: "Play a timeline by name (Pixera.Compound.setTransportModeOnTimeline, mode Play).",
		Annotations: annWrite("Play timeline")}, d.timelinePlay)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_pause",
		Description: "Pause a timeline by name.",
		Annotations: annWrite("Pause timeline")}, d.timelinePause)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_stop",
		Description: "Stop a timeline by name.",
		Annotations: annWrite("Stop timeline")}, d.timelineStop)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_transport_by_index",
		Description: "Set a timeline's transport mode by its zero-based index in the Compositing tab (mode: 1=play, 2=pause, 3=stop).",
		Annotations: annWrite("Timeline transport by index")}, d.timelineTransportByIndex)

	mcp.AddTool(s, &mcp.Tool{Name: "pixera_start_first_timeline",
		Description: "Play the first timeline (Pixera.Compound.startFirstTimeline).",
		Annotations: annWrite("Start first timeline")}, d.startFirst)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_pause_first_timeline",
		Description: "Pause the first timeline.",
		Annotations: annWrite("Pause first timeline")}, d.pauseFirst)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_stop_first_timeline",
		Description: "Stop the first timeline.",
		Annotations: annWrite("Stop first timeline")}, d.stopFirst)

	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_get_transport_mode",
		Description: "Get a timeline's transport mode (1=play, 2=pause, 3=stop).",
		Annotations: annRead("Get transport mode")}, d.getTransportMode)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_get_time",
		Description: "Get a timeline's current time as HH:MM:SS:FF.",
		Annotations: annRead("Get timeline time")}, d.getTime)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_goto_seconds",
		Description: "Seek a timeline to a time in seconds, optionally also setting the transport mode.",
		Annotations: annWrite("Seek timeline")}, d.gotoSeconds)

	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_apply_cue_number",
		Description: "Trigger a cue by number on a timeline (Pixera.Compound.applyCueNumberOnTimeline).",
		Annotations: annWrite("Apply cue number")}, d.applyCueNumber)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_apply_cue_name",
		Description: "Trigger a cue by name on a timeline (Pixera.Compound.applyCueOnTimeline).",
		Annotations: annWrite("Apply cue")}, d.applyCueName)
	mcp.AddTool(s, &mcp.Tool{Name: "pixera_timeline_set_opacity",
		Description: "Set a timeline's opacity (0.0–1.0).",
		Annotations: annWrite("Set timeline opacity")}, d.setOpacity)
}

type timelineIn struct {
	Timeline string `json:"timeline" jsonschema:"the timeline name"`
}

func (d *deps) timelinePlay(ctx context.Context, _ *mcp.CallToolRequest, in timelineIn) (*mcp.CallToolResult, any, error) {
	return d.setMode(ctx, in.Timeline, modePlay, "Play")
}
func (d *deps) timelinePause(ctx context.Context, _ *mcp.CallToolRequest, in timelineIn) (*mcp.CallToolResult, any, error) {
	return d.setMode(ctx, in.Timeline, modePause, "Pause")
}
func (d *deps) timelineStop(ctx context.Context, _ *mcp.CallToolRequest, in timelineIn) (*mcp.CallToolResult, any, error) {
	return d.setMode(ctx, in.Timeline, modeStop, "Stop")
}

func (d *deps) setMode(ctx context.Context, timeline string, mode int, label string) (*mcp.CallToolResult, any, error) {
	if timeline == "" {
		return nil, nil, fmt.Errorf("timeline is required")
	}
	return d.do(ctx, fmt.Sprintf("%s timeline %q:", label, timeline),
		"Pixera.Compound.setTransportModeOnTimeline",
		map[string]any{"timelineName": timeline, "mode": mode})
}

type transportByIndexIn struct {
	Index int `json:"index" jsonschema:"zero-based timeline index in the Compositing tab"`
	Mode  int `json:"mode" jsonschema:"transport mode: 1 play, 2 pause, 3 stop"`
}

func (d *deps) timelineTransportByIndex(ctx context.Context, _ *mcp.CallToolRequest, in transportByIndexIn) (*mcp.CallToolResult, any, error) {
	return d.do(ctx, fmt.Sprintf("Transport mode %d on timeline index %d:", in.Mode, in.Index),
		"Pixera.Compound.setTransportModeOnTimelineAtIndex",
		map[string]any{"index": in.Index, "mode": in.Mode})
}

func (d *deps) startFirst(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return d.do(ctx, "Start first timeline:", "Pixera.Compound.startFirstTimeline", map[string]any{})
}
func (d *deps) pauseFirst(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return d.do(ctx, "Pause first timeline:", "Pixera.Compound.pauseFirstTimeline", map[string]any{})
}
func (d *deps) stopFirst(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
	return d.do(ctx, "Stop first timeline:", "Pixera.Compound.stopFirstTimeline", map[string]any{})
}

func (d *deps) getTransportMode(ctx context.Context, _ *mcp.CallToolRequest, in timelineIn) (*mcp.CallToolResult, any, error) {
	if in.Timeline == "" {
		return nil, nil, fmt.Errorf("timeline is required")
	}
	return d.do(ctx, fmt.Sprintf("Transport mode of %q:", in.Timeline),
		"Pixera.Compound.getTransportModeOnTimeline", map[string]any{"timelineName": in.Timeline})
}

func (d *deps) getTime(ctx context.Context, _ *mcp.CallToolRequest, in timelineIn) (*mcp.CallToolResult, any, error) {
	if in.Timeline == "" {
		return nil, nil, fmt.Errorf("timeline is required")
	}
	return d.do(ctx, fmt.Sprintf("Current time of %q:", in.Timeline),
		"Pixera.Compound.getCurrentHMSFOfTimeline", map[string]any{"timelineName": in.Timeline})
}

type gotoSecondsIn struct {
	Timeline string  `json:"timeline" jsonschema:"the timeline name"`
	Seconds  float64 `json:"seconds" jsonschema:"target time in seconds"`
	Mode     int     `json:"mode,omitempty" jsonschema:"optional transport mode to set at the same time: 1 play, 2 pause, 3 stop"`
}

func (d *deps) gotoSeconds(ctx context.Context, _ *mcp.CallToolRequest, in gotoSecondsIn) (*mcp.CallToolResult, any, error) {
	if in.Timeline == "" {
		return nil, nil, fmt.Errorf("timeline is required")
	}
	if in.Mode != 0 {
		return d.do(ctx, fmt.Sprintf("Seek %q to %.3fs (mode %d):", in.Timeline, in.Seconds, in.Mode),
			"Pixera.Compound.setCurrentTimeAndTransportModeOfTimelineInSeconds",
			map[string]any{"timelineName": in.Timeline, "time": in.Seconds, "mode": in.Mode})
	}
	return d.do(ctx, fmt.Sprintf("Seek %q to %.3fs:", in.Timeline, in.Seconds),
		"Pixera.Compound.setCurrentTimeOfTimelineInSeconds",
		map[string]any{"timelineName": in.Timeline, "time": in.Seconds})
}

type cueNumberIn struct {
	Timeline  string `json:"timeline" jsonschema:"the timeline name"`
	CueNumber int    `json:"cueNumber" jsonschema:"the cue number to trigger"`
}

func (d *deps) applyCueNumber(ctx context.Context, _ *mcp.CallToolRequest, in cueNumberIn) (*mcp.CallToolResult, any, error) {
	if in.Timeline == "" {
		return nil, nil, fmt.Errorf("timeline is required")
	}
	return d.do(ctx, fmt.Sprintf("Apply cue %d on %q:", in.CueNumber, in.Timeline),
		"Pixera.Compound.applyCueNumberOnTimeline",
		map[string]any{"timelineName": in.Timeline, "cueNumber": in.CueNumber})
}

type cueNameIn struct {
	Timeline string `json:"timeline" jsonschema:"the timeline name"`
	CueName  string `json:"cueName" jsonschema:"the cue name to trigger"`
}

func (d *deps) applyCueName(ctx context.Context, _ *mcp.CallToolRequest, in cueNameIn) (*mcp.CallToolResult, any, error) {
	if in.Timeline == "" || in.CueName == "" {
		return nil, nil, fmt.Errorf("timeline and cueName are required")
	}
	return d.do(ctx, fmt.Sprintf("Apply cue %q on %q:", in.CueName, in.Timeline),
		"Pixera.Compound.applyCueOnTimeline",
		map[string]any{"timelineName": in.Timeline, "cueName": in.CueName})
}

type opacityIn struct {
	Timeline string  `json:"timeline" jsonschema:"the timeline name"`
	Opacity  float64 `json:"opacity" jsonschema:"opacity from 0.0 to 1.0"`
}

func (d *deps) setOpacity(ctx context.Context, _ *mcp.CallToolRequest, in opacityIn) (*mcp.CallToolResult, any, error) {
	if in.Timeline == "" {
		return nil, nil, fmt.Errorf("timeline is required")
	}
	return d.do(ctx, fmt.Sprintf("Set opacity of %q to %.3f:", in.Timeline, in.Opacity),
		"Pixera.Compound.setOpacityOnTimeline",
		map[string]any{"timelineName": in.Timeline, "opacity": in.Opacity})
}
