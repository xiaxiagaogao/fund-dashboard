package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// GET /api/me/summary
//
//	{
//	  "shares":         454.5454,
//	  "net_deposits":   500.00,
//	  "value_usdt":     625.00,
//	  "pnl_usdt":       125.00,
//	  "pnl_pct":        0.25,
//	  "latest_nav":     1.375,
//	  "latest_equity":  2000.00,
//	  "snapshot_at_ms": 1715760000000
//	}
func (s *Server) handleMySummary(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())

	shares, err := store.FriendShares(r.Context(), s.DB, c.FriendID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "shares query failed")
		return
	}
	netDep, err := store.FriendNetDeposits(r.Context(), s.DB, c.FriendID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "deposits query failed")
		return
	}
	latest, err := store.LatestNAV(r.Context(), s.DB)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nav query failed")
		return
	}
	value := shares * latest.NAV
	pnl := value - netDep
	pnlPct := 0.0
	if netDep > 0 {
		pnlPct = pnl / netDep
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"shares":         shares,
		"net_deposits":   netDep,
		"value_usdt":     value,
		"pnl_usdt":       pnl,
		"pnl_pct":        pnlPct,
		"latest_nav":     latest.NAV,
		"latest_equity":  latest.TotalEquityUSDT,
		"snapshot_at_ms": latest.TakenAt,
	})
}

// GET /api/me/events  → list of cash events for the calling friend
func (s *Server) handleMyEvents(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	events, err := store.ListCashEventsByFriend(r.Context(), s.DB, c.FriendID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "events query failed")
		return
	}
	out := make([]map[string]any, 0, len(events))
	running := 0.0
	for _, e := range events {
		running += e.SharesDelta
		out = append(out, map[string]any{
			"id":            e.ID,
			"type":          e.Type,
			"amount_usdt":   e.AmountUSDT,
			"occurred_at":   e.OccurredAt,
			"nav_at_event":  e.NAVAtEvent,
			"shares_delta":  e.SharesDelta,
			"shares_after":  running,
			"source":        e.Source,
			"binance_tx_id": nullStr(e.BinanceTxID),
			"note":          nullStr(e.Note),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /api/equity-curve?from=ms&to=ms
//
// Returns the pool's NAV/equity time series. Defaults to last 30 days.
// All friends see the same curve (per spec: friends mutually visible).
func (s *Server) handleEquityCurve(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UnixMilli()
	from := parseInt64Default(r.URL.Query().Get("from"), now-30*24*60*60*1000)
	to := parseInt64Default(r.URL.Query().Get("to"), now)

	snaps, err := store.ListNAVSnapshotsRange(r.Context(), s.DB, from, to)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "snapshots query failed")
		return
	}
	out := make([]map[string]any, 0, len(snaps))
	for _, s := range snaps {
		out = append(out, map[string]any{
			"taken_at":     s.TakenAt,
			"total_equity": s.TotalEquityUSDT,
			"total_shares": s.TotalShares,
			"nav":          s.NAV,
			"source":       s.Source,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /api/aggregate
//
// Per-friend aggregated table — friends see each other (spec).
//
//	[ { "username": "alice", "name": "...", "shares": 454.5, "value_usdt": 625, "pnl_pct": 0.25 }, ... ]
func (s *Server) handleAggregate(w http.ResponseWriter, r *http.Request) {
	friends, err := store.ListFriends(r.Context(), s.DB)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "friends query failed")
		return
	}
	latest, err := store.LatestNAV(r.Context(), s.DB)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "nav query failed")
		return
	}
	out := make([]map[string]any, 0, len(friends))
	for _, f := range friends {
		shares, _ := store.FriendShares(r.Context(), s.DB, f.ID)
		net, _ := store.FriendNetDeposits(r.Context(), s.DB, f.ID)
		value := shares * latest.NAV
		pnl := value - net
		pnlPct := 0.0
		if net > 0 {
			pnlPct = pnl / net
		}
		out = append(out, map[string]any{
			"username":     f.Username,
			"name":         f.Name,
			"is_admin":     f.IsAdmin,
			"shares":       shares,
			"net_deposits": net,
			"value_usdt":   value,
			"pnl_usdt":     pnl,
			"pnl_pct":      pnlPct,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"friends":        out,
		"latest_nav":     latest.NAV,
		"latest_equity":  latest.TotalEquityUSDT,
		"snapshot_at_ms": latest.TakenAt,
	})
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// nullStr unwraps sql.NullString to nil-or-string for JSON output.
func nullStr(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}
