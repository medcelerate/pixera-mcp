// Package app holds the shared runtime state — the Pixera client and current
// configuration — used by both the MCP server and the admin web console.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/medcelerate/pixera-mcp/internal/config"
	"github.com/medcelerate/pixera-mcp/internal/pixera"
)

// App is the central handle to the Pixera client and config.
type App struct {
	client         *pixera.Client
	logf           func(string, ...any)
	mu             sync.Mutex
	cfg            *config.Config
	cfgPath        string
	lastDiscovered *pixera.Heartbeat
	watching       bool
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

// DiscoveryEnabled reports whether heartbeat discovery is configured.
func (a *App) DiscoveryEnabled() bool { return a.Config().Pixera.Discovery.Enabled }

// LastDiscovered returns the most recent heartbeat seen, if any.
func (a *App) LastDiscovered() *pixera.Heartbeat {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastDiscovered
}

// Discover returns the discovered Pixera endpoint. When a background watcher is
// running (discovery enabled) it uses/awaits the watcher's latest heartbeat so
// only one socket binds the port; otherwise it opens a transient listener. When
// apply is true it repoints the bridge at the discovered endpoint.
func (a *App) Discover(ctx context.Context, apply bool, timeout time.Duration) (*pixera.Heartbeat, error) {
	a.mu.Lock()
	watching := a.watching
	hb := a.lastDiscovered
	a.mu.Unlock()

	if watching {
		if hb == nil {
			deadline := time.Now().Add(timeout)
			for hb == nil && time.Now().Before(deadline) && ctx.Err() == nil {
				time.Sleep(200 * time.Millisecond)
				a.mu.Lock()
				hb = a.lastDiscovered
				a.mu.Unlock()
			}
		}
		if hb == nil {
			return nil, fmt.Errorf("no heartbeat received within %s", timeout)
		}
	} else {
		dc := a.Config().Pixera.Discovery
		port := dc.Port
		if port == 0 {
			port = 1500
		}
		got, err := pixera.DiscoverOnce(ctx, port, dc.MulticastGroup, timeout)
		if err != nil {
			return nil, err
		}
		a.mu.Lock()
		a.lastDiscovered = got
		a.mu.Unlock()
		hb = got
	}

	if apply {
		if err := a.SetTarget(hb.Target()); err != nil {
			return hb, err
		}
	}
	return hb, nil
}

// AutoDiscover starts a single persistent heartbeat watcher (when discovery is
// enabled) that records the latest heartbeat and, if discovery.autoApply is
// set, repoints the bridge whenever the advertised target changes.
func (a *App) AutoDiscover(ctx context.Context) {
	dc := a.Config().Pixera.Discovery
	if !dc.Enabled {
		return
	}
	port := dc.Port
	if port == 0 {
		port = 1500
	}
	a.mu.Lock()
	a.watching = true
	a.mu.Unlock()

	go func() {
		for ctx.Err() == nil {
			err := pixera.Watch(ctx, port, dc.MulticastGroup, func(hb *pixera.Heartbeat) {
				a.mu.Lock()
				a.lastDiscovered = hb
				a.mu.Unlock()
				if dc.AutoApply {
					if got := hb.Target(); got.Host != "" && got.Address() != a.Target().Address() {
						if err := a.SetTarget(got); err != nil {
							a.logf("auto-discover: repoint failed: %v", err)
						} else {
							a.logf("auto-discover: repointed to %s", got.Address())
						}
					}
				}
			})
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				a.logf("discovery watch error: %v; retrying", err)
				time.Sleep(3 * time.Second)
			}
		}
	}()
}
