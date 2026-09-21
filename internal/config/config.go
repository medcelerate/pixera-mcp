// Package config defines the pixera-mcp configuration file format and the
// loading, validation and persistence logic shared by the CLI and the web UI.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/medcelerate/pixera-mcp/internal/pixera"
	"gopkg.in/yaml.v3"
)

// MCP transport values.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
	TransportBoth  = "both"
)

// Config is the full bridge configuration.
type Config struct {
	Pixera PixeraConfig `yaml:"pixera"`
	MCP    MCPConfig    `yaml:"mcp"`
	Web    WebConfig    `yaml:"web"`
	Log    LogConfig    `yaml:"log"`
}

// PixeraConfig is the target Pixera server's API endpoint.
type PixeraConfig struct {
	Host           string          `yaml:"host"`
	Port           int             `yaml:"port"`
	TimeoutSeconds int             `yaml:"timeoutSeconds"`
	Discovery      DiscoveryConfig `yaml:"discovery"`
}

// DiscoveryConfig controls auto-discovery of the Pixera API endpoint via the
// heartbeat port (a UDP JSON broadcast/multicast that advertises the active
// API port and server IP).
type DiscoveryConfig struct {
	// Enabled turns on the pixera_discover tool and the console's discovery.
	Enabled bool `yaml:"enabled"`
	// Port is the heartbeat UDP port configured in Pixera (default 1500).
	Port int `yaml:"port"`
	// MulticastGroup is the multicast IP to join, if Pixera multicasts the
	// heartbeat. Leave empty for unicast/broadcast.
	MulticastGroup string `yaml:"multicastGroup"`
	// AutoApply repoints the bridge at the discovered server automatically
	// (on startup and whenever a heartbeat changes the target).
	AutoApply bool `yaml:"autoApply"`
}

// MCPConfig selects how MCP clients connect.
type MCPConfig struct {
	Transport string     `yaml:"transport"`
	HTTP      HTTPConfig `yaml:"http"`
}

// HTTPConfig is the bind address for the Streamable HTTP MCP endpoint.
type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

// WebConfig controls the embedded admin console.
type WebConfig struct {
	Enabled bool   `yaml:"enabled"`
	Addr    string `yaml:"addr"`
}

// LogConfig controls logging.
type LogConfig struct {
	Level string `yaml:"level"`
}

// Default returns a configuration with sensible defaults filled in.
func Default() Config {
	return Config{
		Pixera: PixeraConfig{Host: "127.0.0.1", Port: 1400, TimeoutSeconds: 15},
		// MCP is exposed on all interfaces by default: the bridge typically runs
		// alongside Pixera and AI clients connect to it over the network.
		MCP: MCPConfig{Transport: TransportHTTP, HTTP: HTTPConfig{Addr: "0.0.0.0:8095"}},
		Web: WebConfig{Enabled: true, Addr: "0.0.0.0:8096"},
		Log: LogConfig{Level: "info"},
	}
}

// Load reads a YAML config from path, applies defaults and env overrides.
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", path, err)
			}
		case os.IsNotExist(err):
		default:
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
	}
	cfg.applyDefaults()
	cfg.applyEnv()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	d := Default()
	if c.Pixera.Host == "" {
		c.Pixera.Host = d.Pixera.Host
	}
	if c.Pixera.Port == 0 {
		c.Pixera.Port = d.Pixera.Port
	}
	if c.Pixera.TimeoutSeconds == 0 {
		c.Pixera.TimeoutSeconds = d.Pixera.TimeoutSeconds
	}
	if c.Pixera.Discovery.Port == 0 {
		c.Pixera.Discovery.Port = 1500
	}
	if c.MCP.Transport == "" {
		c.MCP.Transport = d.MCP.Transport
	}
	if c.MCP.HTTP.Addr == "" {
		c.MCP.HTTP.Addr = d.MCP.HTTP.Addr
	}
	if c.Web.Addr == "" {
		c.Web.Addr = d.Web.Addr
	}
	if c.Log.Level == "" {
		c.Log.Level = d.Log.Level
	}
}

func (c *Config) applyEnv() {
	if v := os.Getenv("PIXERAMCP_HOST"); v != "" {
		c.Pixera.Host = v
	}
	if v := os.Getenv("PIXERAMCP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Pixera.Port = p
		}
	}
	if v := os.Getenv("PIXERAMCP_MCP_TRANSPORT"); v != "" {
		c.MCP.Transport = v
	}
	if v := os.Getenv("PIXERAMCP_MCP_HTTP_ADDR"); v != "" {
		c.MCP.HTTP.Addr = v
	}
	if v := os.Getenv("PIXERAMCP_WEB_ADDR"); v != "" {
		c.Web.Addr = v
	}
	if v := os.Getenv("PIXERAMCP_WEB_ENABLED"); v != "" {
		c.Web.Enabled, _ = strconv.ParseBool(v)
	}
	if v := os.Getenv("PIXERAMCP_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
}

// Validate checks the configuration for internal consistency.
func (c *Config) Validate() error {
	switch c.MCP.Transport {
	case TransportStdio, TransportHTTP, TransportBoth:
	default:
		return fmt.Errorf("invalid mcp.transport %q (want stdio|http|both)", c.MCP.Transport)
	}
	if c.Pixera.Port < 0 || c.Pixera.Port > 65535 {
		return fmt.Errorf("invalid pixera.port %d", c.Pixera.Port)
	}
	return nil
}

// Save writes the config to path as YAML using an atomic temp-file rename.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".pixera-mcp-*.yaml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// Target returns the pixera client target for this config.
func (c *Config) Target() pixera.Target {
	return pixera.Target{Host: c.Pixera.Host, Port: c.Pixera.Port}
}
