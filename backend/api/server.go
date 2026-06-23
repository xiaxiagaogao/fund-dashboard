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
//	  GET  /api/positions/open       — Binance live positionRisk
//	  GET  /api/positions/closed     — reconstructed from fund.db binance_fills
//	  GET  /api/positions/allocation — capital split + leverage + notional by symbol
//	  GET  /api/positions/stats      — aggregate win rate + per-symbol breakdown
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
	"sync"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/positions"
	"github.com/xiagao/fund-dashboard/backend/store"
)

type Server struct {
	DB           *sql.DB
	Binance      *binance.Client
	Positions    *positions.Orchestrator
	JWTSecret    string
	CookieSecure bool   // true in prod (behind HTTPS), false for local dev
	StaticDir    string // optional: directory holding the built SvelteKit SPA. When set, anything not /api/* or /healthz falls through to it.

	// cashEventMu serializes the read-shares → compute → insert sequence in
	// handleAdminCashEvent. SQLite's busy_timeout protects the writes, but the
	// share math reads pool state first — two concurrent admin submissions
	// could both mint against the same pre-state. One admin in practice, but
	// the lock makes it correct by construction.
	cashEventMu sync.Mutex

	limiterOnce sync.Once
	loginLimit  *loginLimiter
}

func (s *Server) limiter() *loginLimiter {
	s.limiterOnce.Do(func() { s.loginLimit = newLoginLimiter() })
	return s.loginLimit
}

// auth / admin wrap the pure-JWT middleware with a DB-backed session check:
// the friend must still exist and the token's password fingerprint must match
// the current hash. This is what makes set-password / friend deletion take
// effect immediately instead of "whenever the 7-day cookie expires".
func (s *Server) auth(h http.HandlerFunc) http.Handler {
	return middleware.RequireAuth(s.JWTSecret, s.validateSession(h))
}

func (s *Server) admin(h http.HandlerFunc) http.Handler {
	return middleware.RequireAdmin(s.JWTSecret, s.validateSession(h))
}

func (s *Server) validateSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := middleware.FromContext(r.Context())
		f, err := store.GetFriendByID(r.Context(), s.DB, c.FriendID)
		if err != nil || middleware.PasswordVersion(f.PasswordHash) != c.Pwv {
			middleware.ClearSessionCookie(w, s.CookieSecure)
			http.Error(w, "session expired", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Routes returns a configured http.Handler with all routes registered.
// Uses Go 1.22+ router syntax (METHOD path).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("POST /api/login", s.handleLogin)

	// Authenticated (any user)
	mux.Handle("POST /api/logout", s.auth(s.handleLogout))
	mux.Handle("GET /api/me", s.auth(s.handleMe))
	mux.Handle("GET /api/me/summary", s.auth(s.handleMySummary))
	mux.Handle("GET /api/me/events", s.auth(s.handleMyEvents))
	mux.Handle("GET /api/me/export.csv", s.auth(s.handleMyExportCSV))
	mux.Handle("GET /api/equity-curve", s.auth(s.handleEquityCurve))
	mux.Handle("GET /api/index-prices", s.auth(s.handleIndexPrices))
	mux.Handle("GET /api/aggregate", s.auth(s.handleAggregate))
	mux.Handle("GET /api/positions/open", s.auth(s.handleOpenPositions))
	mux.Handle("GET /api/positions/closed", s.auth(s.handleClosedPositions))
	mux.Handle("GET /api/positions/allocation", s.auth(s.handleAllocation))
	mux.Handle("GET /api/positions/stats", s.auth(s.handleStats))
	mux.Handle("GET /api/positions/daily-pnl", s.admin(s.handleDailyPnL))

	// Admin only
	mux.Handle("GET /api/admin/friends", s.admin(s.handleListFriends))
	mux.Handle("POST /api/admin/friends", s.admin(s.handleCreateFriend))
	mux.Handle("POST /api/admin/cash-events", s.admin(s.handleAdminCashEvent))
	mux.Handle("GET /api/admin/cash-events", s.admin(s.handleAdminCashEvents))
	mux.Handle("GET /api/admin/recent-fills", s.admin(s.handleAdminRecentFills))
	mux.Handle("POST /api/admin/snapshot", s.admin(s.handleAdminSnapshot))

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
