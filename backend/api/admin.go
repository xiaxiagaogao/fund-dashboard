package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/nav"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// GET /api/admin/cash-events — full pool ledger (every deposit/withdrawal
// across all friends), newest first. Admin only — friends see only their own
// rows via /api/me/events.
func (s *Server) handleAdminCashEvents(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT ce.id, ce.friend_id, f.username, f.name,
		       ce.type, ce.amount_usdt, ce.occurred_at,
		       ce.nav_at_event, ce.shares_delta, ce.source,
		       COALESCE(ce.binance_tx_id, ''), COALESCE(ce.note, ''),
		       ce.created_at
		FROM cash_events ce
		JOIN friends f ON f.id = ce.friend_id
		ORDER BY ce.occurred_at DESC
		LIMIT ?`, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ledger query failed: "+err.Error())
		return
	}
	defer rows.Close()
	out := make([]map[string]any, 0, 16)
	for rows.Next() {
		var (
			id, friendID, occurredAt, createdAt int64
			username, name, typ, source, txID, note string
			amount, navAt, sharesDelta float64
		)
		if err := rows.Scan(&id, &friendID, &username, &name, &typ, &amount, &occurredAt,
			&navAt, &sharesDelta, &source, &txID, &note, &createdAt); err != nil {
			writeErr(w, http.StatusInternalServerError, "ledger scan failed: "+err.Error())
			return
		}
		out = append(out, map[string]any{
			"id": id, "friend_id": friendID, "username": username, "name": name,
			"type": typ, "amount_usdt": amount, "occurred_at": occurredAt,
			"nav_at_event": navAt, "shares_delta": sharesDelta, "source": source,
			"binance_tx_id": txID, "note": note, "created_at": createdAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// GET /api/admin/friends — list all friends (admin sees password_hash absent)
func (s *Server) handleListFriends(w http.ResponseWriter, r *http.Request) {
	fs, err := store.ListFriends(r.Context(), s.DB)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list friends failed")
		return
	}
	out := make([]map[string]any, 0, len(fs))
	for _, f := range fs {
		out = append(out, map[string]any{
			"id": f.ID, "name": f.Name, "username": f.Username,
			"is_admin": f.IsAdmin, "created_at": f.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type createFriendReq struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// POST /api/admin/friends
func (s *Server) handleCreateFriend(w http.ResponseWriter, r *http.Request) {
	var in createFriendReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	if in.Name == "" || in.Username == "" || len(in.Password) < 8 {
		writeErr(w, http.StatusBadRequest, "name, username required; password >= 8 chars")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash failed")
		return
	}
	id, err := store.CreateFriend(r.Context(), s.DB, in.Name, in.Username, string(hash), in.IsAdmin)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "create failed (username taken?)")
		return
	}
	caller := middleware.FromContext(r.Context())
	store.WriteAudit(r.Context(), s.DB, caller.Username, "friend.create", map[string]any{
		"id": id, "username": in.Username, "is_admin": in.IsAdmin,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

type cashEventReq struct {
	Username     string  `json:"username"`
	Type         string  `json:"type"`           // deposit | withdraw
	AmountUSDT   float64 `json:"amount_usdt"`
	OccurredAtMs int64   `json:"occurred_at_ms"` // 0 = now
	Note         string  `json:"note"`

	// Optional manual override — if ManualNAV > 0, the auto Binance-based math
	// is bypassed entirely and the supplied NAV is recorded as-is. Use for
	// backdated entries or cases where the admin already knows the right NAV.
	ManualNAV float64 `json:"manual_nav,omitempty"`

	// Bootstrap escape hatch — for the very first deposit, system normally
	// enforces |amount - current_binance_equity| / equity <= 1% so the pool
	// starts cleanly aligned. Set true to bypass that check (NAV will diverge).
	SkipBootstrapCheck bool `json:"skip_bootstrap_check,omitempty"`
}

// POST /api/admin/cash-events
//
// Records a deposit/withdraw against the share ledger. Default math assumes
// the cash has already settled on Binance at admin-record time
// (record-after-transfer flow):
//
//	for deposit:  pre_equity = live_equity - amount; NAV = pre_equity / shares; shares_minted = amount / NAV
//	for withdraw: pre_equity = live_equity + amount; NAV = pre_equity / shares; shares_burned = amount / NAV
//
// For bootstrap (total_shares == 0): NAV = 1.0, mint = amount. A soft check
// rejects the request if amount diverges from live equity by > 1% (see
// SkipBootstrapCheck to bypass).
//
// If ManualNAV > 0 is supplied, the entire Binance round-trip is skipped and
// the supplied NAV is used as-is. Useful for backdated entries or when admin
// already computed the NAV manually.
func (s *Server) handleAdminCashEvent(w http.ResponseWriter, r *http.Request) {
	var in cashEventReq
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "bad json")
		return
	}
	if in.Type != store.EventDeposit && in.Type != store.EventWithdraw {
		writeErr(w, http.StatusBadRequest, "type must be deposit or withdraw")
		return
	}
	if in.AmountUSDT <= 0 {
		writeErr(w, http.StatusBadRequest, "amount_usdt must be positive")
		return
	}
	if in.OccurredAtMs == 0 {
		in.OccurredAtMs = time.Now().UnixMilli()
	}

	friend, err := store.GetFriendByUsername(r.Context(), s.DB, in.Username)
	if err != nil {
		writeErr(w, http.StatusNotFound, "friend not found")
		return
	}
	totalShares, _ := store.TotalShares(r.Context(), s.DB)

	var (
		sharesDelta float64
		navUsed     float64
		liveEquity  float64 // for response + snapshot; 0 if manual override path
	)

	if in.ManualNAV > 0 {
		// Manual override — trust admin's NAV, no Binance call.
		navUsed = in.ManualNAV
		switch in.Type {
		case store.EventDeposit:
			sharesDelta = in.AmountUSDT / navUsed
		case store.EventWithdraw:
			sharesDelta = -in.AmountUSDT / navUsed
		}
	} else {
		// Auto path: read live Binance equity.
		if s.Binance == nil {
			writeErr(w, http.StatusServiceUnavailable, "binance client not configured (set BINANCE_API_KEY/SECRET or pass manual_nav)")
			return
		}
		equity, _, err := s.Binance.AccountEquity(r.Context())
		if err != nil {
			writeErr(w, http.StatusBadGateway, "binance: "+err.Error())
			return
		}
		liveEquity = equity

		// Bootstrap soft check
		if totalShares == 0 && !in.SkipBootstrapCheck {
			if equity <= 0 {
				writeErr(w, http.StatusBadRequest, "bootstrap requires Binance equity > 0; deposit USDT first or pass skip_bootstrap_check=true")
				return
			}
			if absDiff(in.AmountUSDT, equity)/equity > 0.01 {
				writeErr(w, http.StatusBadRequest,
					fmt.Sprintf("bootstrap amount %.4f deviates from current Binance equity %.4f by >1%%; adjust the account balance to match, or pass skip_bootstrap_check=true to accept NAV divergence",
						in.AmountUSDT, equity))
				return
			}
		}

		switch in.Type {
		case store.EventDeposit:
			sharesDelta, navUsed = nav.ComputeMintAfterArrival(in.AmountUSDT, equity, totalShares)
		case store.EventWithdraw:
			burned, n, berr := nav.ComputeBurnAfterDeparture(in.AmountUSDT, equity, totalShares)
			if berr != nil {
				writeErr(w, http.StatusBadRequest, berr.Error())
				return
			}
			sharesDelta, navUsed = -burned, n
		}
	}

	// Withdrawal: never let a friend burn more than they hold.
	if in.Type == store.EventWithdraw {
		myShares, _ := store.FriendShares(r.Context(), s.DB, friend.ID)
		if -sharesDelta > myShares+1e-9 {
			writeErr(w, http.StatusBadRequest, "friend does not hold enough shares to withdraw that much")
			return
		}
	}

	id, err := store.InsertCashEvent(r.Context(), s.DB, store.CashEventInput{
		FriendID: friend.ID, Type: in.Type, AmountUSDT: in.AmountUSDT,
		OccurredAt: in.OccurredAtMs, NAVAtEvent: navUsed, SharesDelta: sharesDelta,
		Source: store.SourceManual, Note: in.Note,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "insert failed: "+err.Error())
		return
	}

	// Precise snapshot at the event time. If we have liveEquity, use it directly
	// (it already reflects the post-event state); otherwise extrapolate.
	postEquity := liveEquity
	if postEquity == 0 {
		// Manual-NAV path: synthesize from NAV * shares
		postEquity = (totalShares + sharesDelta) * navUsed
	}
	_, _ = store.InsertNAVSnapshot(r.Context(), s.DB, store.NAVSnapshot{
		TakenAt: in.OccurredAtMs, TotalEquityUSDT: postEquity,
		TotalShares: totalShares + sharesDelta, NAV: navUsed, Source: store.SnapshotCashEvent,
	})

	caller := middleware.FromContext(r.Context())
	store.WriteAudit(r.Context(), s.DB, caller.Username, "cash_event.create", map[string]any{
		"id": id, "username": in.Username, "type": in.Type, "amount_usdt": in.AmountUSDT,
		"nav_at_event": navUsed, "shares_delta": sharesDelta, "live_equity": liveEquity,
		"manual_nav": in.ManualNAV > 0,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":            id,
		"nav_at_event":  navUsed,
		"shares_delta":  sharesDelta,
		"equity_at_evt": liveEquity,
		"manual_nav":    in.ManualNAV > 0,
	})
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

// GET /api/admin/recent-fills?limit=N
//
// Reads fund.db's own binance_fills table. No nofx coupling.
func (s *Server) handleAdminRecentFills(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	fills, err := store.ListRecentFills(r.Context(), s.DB, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "fills query failed: "+err.Error())
		return
	}
	out := make([]map[string]any, 0, len(fills))
	for _, f := range fills {
		out = append(out, map[string]any{
			"id":            f.ID,
			"trade_id":      f.BinanceTradeID,
			"order_id":      f.BinanceOrderID,
			"symbol":        f.Symbol,
			"side":          f.Side,
			"position_side": f.PositionSide,
			"price":         f.Price,
			"qty":           f.Qty,
			"quote_qty":     f.QuoteQty,
			"realized_pnl":  f.RealizedPnL,
			"commission":    f.Commission,
			"maker":         f.Maker,
			"fill_time":     f.FillTime,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// POST /api/admin/snapshot — force one snapshot now (for testing / on-demand)
func (s *Server) handleAdminSnapshot(w http.ResponseWriter, r *http.Request) {
	if s.Binance == nil {
		writeErr(w, http.StatusServiceUnavailable, "binance client not configured")
		return
	}
	equity, _, err := s.Binance.AccountEquity(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "binance: "+err.Error())
		return
	}
	totalShares, _ := store.TotalShares(r.Context(), s.DB)
	currentNAV := nav.CurrentNAV(equity, totalShares)
	now := time.Now().UnixMilli()
	id, err := store.InsertNAVSnapshot(r.Context(), s.DB, store.NAVSnapshot{
		TakenAt: now, TotalEquityUSDT: equity, TotalShares: totalShares, NAV: currentNAV, Source: store.SnapshotScheduled,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "snapshot insert failed: "+err.Error())
		return
	}
	caller := middleware.FromContext(r.Context())
	store.WriteAudit(r.Context(), s.DB, caller.Username, "snapshot.write", map[string]any{
		"id": id, "equity": equity, "shares": totalShares, "nav": currentNAV, "manual": true,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "taken_at": now, "total_equity": equity, "total_shares": totalShares, "nav": currentNAV,
	})
}
