// Package api hosts the dashboard's HTTP handlers.
//
// Route layout:
//
//	Public:
//	  POST /api/login              — { username, password } → session cookie
//
//	Authenticated (any logged-in friend, including admin):
//	  POST /api/logout             — clears cookie
//	  GET  /api/me                 — caller identity
//	  GET  /api/me/summary         — my shares / value / pnl
//	  GET  /api/me/events          — my cash events
//	  GET  /api/me/export.csv      — my statement
//	  GET  /api/equity-curve       — pool NAV time series (?from=ms&to=ms, defaults to 30d)
//	  GET  /api/aggregate          — pool-wide aggregated stats (per-friend table; friends mutually visible)
//	  GET  /api/positions/open     — Binance live positionRisk
//	  GET  /api/positions/closed   — reconstructed from fund.db binance_fills
//	  GET  /api/positions/stats    — aggregate win rate + per-symbol breakdown
//
//	Admin only:
//	  GET  /api/admin/friends             — list
//	  POST /api/admin/friends             — create
//	  POST /api/admin/cash-events         — record deposit/withdraw
//	  GET  /api/admin/recent-fills        — last N fills from fund.db (?limit=N)
//	  POST /api/admin/snapshot            — force one snapshot now
package api

import (
	"database/sql"
	"net/http"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/positions"
)

type Server struct {
	DB           *sql.DB
	Binance      *binance.Client
	Positions    *positions.Orchestrator
	JWTSecret    string
	CookieSecure bool   // true in prod (behind HTTPS), false for local dev
	StaticDir    string // optional: directory holding the built SvelteKit SPA. When set, anything not /api/* or /healthz falls through to it.
}

// Routes returns a configured http.Handler with all routes registered.
// Uses Go 1.22+ router syntax (METHOD path).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /api/login", s.handleLogin)

	// Authenticated (any user)
	mux.Handle("POST /api/logout", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleLogout)))
	mux.Handle("GET /api/me", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleMe)))
	mux.Handle("GET /api/me/summary", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleMySummary)))
	mux.Handle("GET /api/me/events", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleMyEvents)))
	mux.Handle("GET /api/me/export.csv", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleMyExportCSV)))
	mux.Handle("GET /api/equity-curve", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleEquityCurve)))
	mux.Handle("GET /api/aggregate", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleAggregate)))
	mux.Handle("GET /api/positions/open", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleOpenPositions)))
	mux.Handle("GET /api/positions/closed", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleClosedPositions)))
	mux.Handle("GET /api/positions/stats", middleware.RequireAuth(s.JWTSecret, http.HandlerFunc(s.handleStats)))

	// Admin only
	mux.Handle("GET /api/admin/friends", middleware.RequireAdmin(s.JWTSecret, http.HandlerFunc(s.handleListFriends)))
	mux.Handle("POST /api/admin/friends", middleware.RequireAdmin(s.JWTSecret, http.HandlerFunc(s.handleCreateFriend)))
	mux.Handle("POST /api/admin/cash-events", middleware.RequireAdmin(s.JWTSecret, http.HandlerFunc(s.handleAdminCashEvent)))
	mux.Handle("GET /api/admin/cash-events", middleware.RequireAdmin(s.JWTSecret, http.HandlerFunc(s.handleAdminCashEvents)))
	mux.Handle("GET /api/admin/recent-fills", middleware.RequireAdmin(s.JWTSecret, http.HandlerFunc(s.handleAdminRecentFills)))
	mux.Handle("POST /api/admin/snapshot", middleware.RequireAdmin(s.JWTSecret, http.HandlerFunc(s.handleAdminSnapshot)))

	// Health (no auth — for orchestration)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Static SPA fallback (production only — local dev uses vite on :3100).
	// Registered LAST as the catch-all "/" pattern; any /api/* and /healthz
	// match more-specific patterns above and never reach here.
	if s.StaticDir != "" {
		mux.Handle("/", StaticHandler(s.StaticDir))
	}

	return mux
}
