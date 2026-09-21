// Package app holds the shared runtime state — the Pixera client and current
// configuration — used by both the MCP server and the admin web console.
package app

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/medcelerate/pixera-mcp/internal/config"
	"github.com/medcelerate/pixera-mcp/internal/pixera"
)

// App is the central handle to the Pixera client and config.
type App struct {
	client  *pixera.Client
	logf    func(string, ...any)
	mu      sync.Mutex
	cfg     *config.Config
	cfgPath string
}

// New builds an App from a config. cfgPath may be empty (changes apply in
// memory only).
func New(cfg *config.Config, cfgPath string, logf func(string, ...any)) *App {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	timeout := time.Duration(cfg.Pixera.TimeoutSeconds) * time.Second
	return &App{
		client:  pixera.NewClient(cfg.Target(), timeout),
		logf:    logf,
		cfg:     cfg,
		cfgPath: cfgPath,
	}
}

// Client returns the Pixera API client.
func (a *App) Client() *pixera.Client { return a.client }

// Config returns the current configuration.
func (a *App) Config() *config.Config {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// Target returns the Pixera server the client currently points at.
func (a *App) Target() pixera.Target { return a.client.Target() }

// SetTarget repoints the Pixera client and persists the change.
func (a *App) SetTarget(t pixera.Target) error {
	a.client.SetTarget(t)
	a.mu.Lock()
	defer a.mu.Unlock()
	nt := a.client.Target()
	a.cfg.Pixera.Host = nt.Host
	a.cfg.Pixera.Port = nt.Port
	a.logf("pixera target set to %s", nt.Address())
	if a.cfgPath == "" {
		return nil
	}
	return a.cfg.Save(a.cfgPath)
}

// Status is a snapshot of the connection to the Pixera server.
type Status struct {
	Target      pixera.Target `json:"target"`
	Address     string        `json:"address"`
	Reachable   bool          `json:"reachable"`
	APIRevision string        `json:"apiRevision,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// Status pings the Pixera server (getApiRevision) and reports reachability.
func (a *App) Status(ctx context.Context) Status {
	t := a.Target()
	st := Status{Target: t, Address: t.Address()}
	raw, err := a.client.Ping(ctx)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Reachable = true
	st.APIRevision = strings.TrimSpace(string(json.RawMessage(raw)))
	return st
}
