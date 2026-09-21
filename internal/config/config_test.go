package config

import (
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Pixera.Port != 1400 || cfg.Pixera.Host != "127.0.0.1" {
		t.Fatalf("pixera defaults wrong: %+v", cfg.Pixera)
	}
	if cfg.MCP.Transport != TransportHTTP || cfg.MCP.HTTP.Addr != "0.0.0.0:8095" {
		t.Fatalf("mcp defaults wrong: %+v", cfg.MCP)
	}
	if cfg.Target().Address() != "127.0.0.1:1400" {
		t.Fatalf("target = %q", cfg.Target().Address())
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	cfg := Default()
	cfg.Pixera.Host = "10.0.0.30"
	cfg.Pixera.Port = 1412
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Pixera.Host != "10.0.0.30" || loaded.Pixera.Port != 1412 {
		t.Fatalf("round-trip lost values: %+v", loaded.Pixera)
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	cfg.MCP.Transport = "bogus"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected transport error")
	}
}
