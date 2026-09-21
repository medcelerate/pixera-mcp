package mcpserver

import (
	"testing"

	"github.com/medcelerate/pixera-mcp/internal/app"
	"github.com/medcelerate/pixera-mcp/internal/config"
)

// TestNewRegistersTools ensures every tool's input schema is valid — AddTool
// panics on a bad schema, so a clean New() is the assertion.
func TestNewRegistersTools(t *testing.T) {
	cfg := config.Default()
	a := app.New(&cfg, "", nil)
	if s := New(a); s == nil {
		t.Fatal("New returned nil")
	}
}
