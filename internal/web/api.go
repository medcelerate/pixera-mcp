package web

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/medcelerate/pixera-mcp/internal/app"
	"github.com/medcelerate/pixera-mcp/internal/pixera"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GET /api/status — current target + reachability + API revision + discovery.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, struct {
		app.Status
		DiscoveryEnabled bool              `json:"discoveryEnabled"`
		Discovered       *pixera.Heartbeat `json:"discovered,omitempty"`
	}{
		Status:           s.app.Status(ctx),
		DiscoveryEnabled: s.app.DiscoveryEnabled(),
		Discovered:       s.app.LastDiscovered(),
	})
}

// POST /api/discover — listen for a heartbeat; ?apply=true repoints the target.
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	apply := r.URL.Query().Get("apply") == "true"
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	hb, err := s.app.Discover(ctx, apply, 8*time.Second)
	if err != nil {
		writeError(w, http.StatusOK, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, hb)
}

// GET /api/target returns the current target; POST /api/target repoints it.
func (s *Server) handleTarget(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.app.Target())

	case http.MethodPost:
		var t pixera.Target
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		if t.Host == "" {
			writeError(w, http.StatusBadRequest, "host is required")
			return
		}
		if err := s.app.SetTarget(t); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		writeJSON(w, http.StatusOK, s.app.Status(ctx))

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
