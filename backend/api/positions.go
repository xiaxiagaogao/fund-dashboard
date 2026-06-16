package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/xiagao/fund-dashboard/backend/positions"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// reportLoc is the wall-clock zone used to bucket trades into calendar days on
// the 复盘 heatmap. Operator is in China; fall back to a fixed +8 if tzdata is
// somehow missing.
var reportLoc = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}()

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

// GET /api/positions/daily-pnl?days=N
//
// Per-day net realized PnL from fund.db binance_fills — powers the 复盘
// calendar heatmap. Reads fund.db directly (no Binance), so it works even
// when live keys are absent. Admin-only (operator's review tool).
func (s *Server) handleDailyPnL(w http.ResponseWriter, r *http.Request) {
	days := 120
	if q := r.URL.Query().Get("days"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 730 {
			days = n
		}
	}
	sinceMs := time.Now().AddDate(0, 0, -days).UnixMilli()
	fills, err := store.ListFillsSince(r.Context(), s.DB, sinceMs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "fills query failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, positions.DailyPnL(fills, reportLoc))
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
