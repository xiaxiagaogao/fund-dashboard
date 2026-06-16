package api

import (
	"net/http"
	"strconv"

	"github.com/xiagao/fund-dashboard/backend/positions"
)

// Position handlers are powered by the Binance-first orchestrator
// (backend/positions). Binance is the source of truth — independent of nofx
// being up. nofx, when reachable, contributes intent_type / entry_thesis /
// close_reason as metadata overlays.

// GET /api/positions/open
func (s *Server) handleOpenPositions(w http.ResponseWriter, r *http.Request) {
	if s.Positions == nil {
		writeErr(w, http.StatusServiceUnavailable, "positions orchestrator not configured (BINANCE_API_KEY/SECRET required)")
		return
	}
	open, _, err := s.Positions.Refresh(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "positions refresh: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, open)
}

// GET /api/positions/closed?limit=N
func (s *Server) handleClosedPositions(w http.ResponseWriter, r *http.Request) {
	if s.Positions == nil {
		writeErr(w, http.StatusServiceUnavailable, "positions orchestrator not configured")
		return
	}
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	_, closed, err := s.Positions.Refresh(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "positions refresh: "+err.Error())
		return
	}
	if limit < len(closed) {
		closed = closed[:limit]
	}
	writeJSON(w, http.StatusOK, closed)
}

// GET /api/positions/allocation
//
// Capital snapshot for the friend-facing donuts: margin-vs-cash split, cross
// leverage, and notional weight per open position. All friends may see it.
func (s *Server) handleAllocation(w http.ResponseWriter, r *http.Request) {
	if s.Positions == nil {
		writeErr(w, http.StatusServiceUnavailable, "positions orchestrator not configured")
		return
	}
	alloc, err := s.Positions.RefreshAllocation(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "allocation refresh: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, alloc)
}

// GET /api/positions/stats?window=N
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if s.Positions == nil {
		writeErr(w, http.StatusServiceUnavailable, "positions orchestrator not configured")
		return
	}
	window := 200
	if q := r.URL.Query().Get("window"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 1000 {
			window = n
		}
	}
	_, closed, err := s.Positions.Refresh(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "positions refresh: "+err.Error())
		return
	}
	subset := closed
	if window < len(subset) {
		subset = subset[:window]
	}
	stats := positions.ComputeStats(subset)
	bySymbol := positions.AggregateBySymbol(subset)
	writeJSON(w, http.StatusOK, map[string]any{
		"window":    window,
		"stats":     stats,
		"by_symbol": bySymbol,
	})
}
