package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	EventDeposit  = "deposit"
	EventWithdraw = "withdraw"

	SourceManual          = "manual"
	SourceBinanceTransfer = "binance_transfer"
)

type CashEvent struct {
	ID           int64
	FriendID     int64
	Type         string  // deposit | withdraw
	AmountUSDT   float64 // always positive
	OccurredAt   int64   // unix ms
	NAVAtEvent   float64
	SharesDelta  float64 // signed (+ deposit, - withdraw)
	Source       string
	BinanceTxID  sql.NullString
	Note         sql.NullString
	CreatedAt    int64
}

// CashEventInput is what the API/admin layer hands in. The store fills in
// shares_delta + nav_at_event by reading current pool state inside a TX.
type CashEventInput struct {
	FriendID    int64
	Type        string  // deposit | withdraw
	AmountUSDT  float64 // positive
	OccurredAt  int64   // unix ms
	NAVAtEvent  float64 // already computed by caller (uses live Binance equity)
	SharesDelta float64 // already signed
	Source      string
	BinanceTxID string  // empty for manual entries
	Note        string
}

// InsertCashEvent appends one cash event row. It does NOT compute NAV/shares —
// the caller is responsible for that (so the live equity read happens at the
// admin layer where we have the Binance client). This keeps store.* free of
// I/O dependencies and easy to test.
//
// Uniqueness guard: if a binance_tx_id is provided and already exists, returns
// ErrDuplicateBinanceTx without touching the table.
var ErrDuplicateBinanceTx = errors.New("store: cash event with this binance_tx_id already exists")

func InsertCashEvent(ctx context.Context, db *sql.DB, in CashEventInput) (int64, error) {
	if in.Type != EventDeposit && in.Type != EventWithdraw {
		return 0, fmt.Errorf("invalid type %q", in.Type)
	}
	if in.AmountUSDT <= 0 {
		return 0, fmt.Errorf("amount must be positive, got %v", in.AmountUSDT)
	}
	if in.NAVAtEvent <= 0 {
		return 0, fmt.Errorf("nav_at_event must be positive, got %v", in.NAVAtEvent)
	}

	var binanceTx interface{}
	if in.BinanceTxID != "" {
		binanceTx = in.BinanceTxID
	}
	var note interface{}
	if in.Note != "" {
		note = in.Note
	}

	now := time.Now().UnixMilli()
	res, err := db.ExecContext(ctx,
		`INSERT INTO cash_events(friend_id, type, amount_usdt, occurred_at, nav_at_event, shares_delta, source, binance_tx_id, note, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.FriendID, in.Type, in.AmountUSDT, in.OccurredAt, in.NAVAtEvent, in.SharesDelta, in.Source, binanceTx, note, now,
	)
	if err != nil {
		// modernc.org/sqlite returns "constraint failed: UNIQUE constraint failed: ..." on dup
		if isUniqueViolation(err) && in.BinanceTxID != "" {
			return 0, ErrDuplicateBinanceTx
		}
		return 0, fmt.Errorf("insert cash_event: %w", err)
	}
	return res.LastInsertId()
}

// TotalShares returns the pool's total outstanding shares = SUM(shares_delta).
// Used by the NAV/share computation to determine current NAV before minting.
func TotalShares(ctx context.Context, db *sql.DB) (float64, error) {
	var total sql.NullFloat64
	err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(shares_delta), 0) FROM cash_events`).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Float64, nil
}

// FriendShares returns one friend's current outstanding shares.
func FriendShares(ctx context.Context, db *sql.DB, friendID int64) (float64, error) {
	var total sql.NullFloat64
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(shares_delta), 0) FROM cash_events WHERE friend_id = ?`, friendID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Float64, nil
}

// FriendNetDeposits returns SUM(amount_usdt where deposit) - SUM(amount_usdt where withdraw)
// for one friend. This is the cost basis for PnL computation.
func FriendNetDeposits(ctx context.Context, db *sql.DB, friendID int64) (float64, error) {
	var total sql.NullFloat64
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN type='deposit' THEN amount_usdt ELSE -amount_usdt END), 0)
		FROM cash_events WHERE friend_id = ?`, friendID,
	).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Float64, nil
}

// ListCashEventsByFriend returns one friend's events ordered by occurred_at ASC.
// Used by the friend page (statement view) and CSV export.
func ListCashEventsByFriend(ctx context.Context, db *sql.DB, friendID int64) ([]CashEvent, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, friend_id, type, amount_usdt, occurred_at, nav_at_event, shares_delta, source, binance_tx_id, note, created_at
		 FROM cash_events WHERE friend_id = ? ORDER BY occurred_at ASC, id ASC`, friendID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CashEvent
	for rows.Next() {
		var e CashEvent
		if err := rows.Scan(&e.ID, &e.FriendID, &e.Type, &e.AmountUSDT, &e.OccurredAt,
			&e.NAVAtEvent, &e.SharesDelta, &e.Source, &e.BinanceTxID, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// HasBinanceTx checks whether a binance_tx_id has already been recorded.
// Used by the daily transfer-sync job to skip already-reconciled transfers.
func HasBinanceTx(ctx context.Context, db *sql.DB, txID string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM cash_events WHERE binance_tx_id = ?`, txID,
	).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite surfaces the SQLite error string directly.
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed") || contains(msg, "constraint failed: UNIQUE")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
