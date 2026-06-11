// Package store owns the dashboard's SQLite persistence.
//
// The driver is pure-Go modernc.org/sqlite — no CGO, smaller container, and the
// nofx data.db can be opened with the same driver in read-only mode (see nofx_ro.go).
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the dashboard fund.db at path and runs migrations.
// WAL is enabled so the hourly snapshot writer doesn't block the API readers.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open fund.db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping fund.db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate fund.db: %w", err)
	}
	return db, nil
}

// migrate runs all CREATE TABLE / CREATE INDEX statements idempotently.
// New schema changes go in this slice in order. Each statement must be safe
// to run on a fresh empty DB AND on an existing DB.
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS friends (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			name          TEXT    NOT NULL,
			username      TEXT    NOT NULL UNIQUE,
			password_hash TEXT    NOT NULL,
			is_admin      INTEGER NOT NULL DEFAULT 0,
			created_at    INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS cash_events (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			friend_id     INTEGER NOT NULL REFERENCES friends(id),
			type          TEXT    NOT NULL CHECK(type IN ('deposit','withdraw')),
			amount_usdt   REAL    NOT NULL CHECK(amount_usdt > 0),
			occurred_at   INTEGER NOT NULL,
			nav_at_event  REAL    NOT NULL CHECK(nav_at_event > 0),
			shares_delta  REAL    NOT NULL,
			source        TEXT    NOT NULL CHECK(source IN ('manual','binance_transfer')),
			binance_tx_id TEXT,
			note          TEXT,
			created_at    INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cash_friend_time ON cash_events(friend_id, occurred_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cash_binance_tx ON cash_events(binance_tx_id) WHERE binance_tx_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS nav_snapshots (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			taken_at          INTEGER NOT NULL UNIQUE,
			total_equity_usdt REAL    NOT NULL,
			total_shares      REAL    NOT NULL,
			nav               REAL    NOT NULL,
			source            TEXT    NOT NULL DEFAULT 'scheduled' CHECK(source IN ('scheduled','cash_event'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nav_taken ON nav_snapshots(taken_at DESC)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			actor        TEXT    NOT NULL,
			action       TEXT    NOT NULL,
			payload_json TEXT    NOT NULL,
			created_at   INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at DESC)`,
		// Binance fills: dashboard's own copy of every Binance USD-M futures
		// trade ever fetched. Backfilled via dashctl, kept current by the
		// scheduler's TradesSync job. Source of truth for closed-position
		// reconstruction in the trade transparency panels — dashboard does not
		// read from nofx for any of this.
		`CREATE TABLE IF NOT EXISTS binance_fills (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			binance_trade_id   INTEGER NOT NULL UNIQUE,
			binance_order_id   INTEGER NOT NULL,
			symbol             TEXT    NOT NULL,
			side               TEXT    NOT NULL CHECK(side IN ('BUY','SELL')),
			position_side      TEXT    NOT NULL,
			price              REAL    NOT NULL,
			qty                REAL    NOT NULL,
			quote_qty          REAL    NOT NULL,
			realized_pnl       REAL    NOT NULL DEFAULT 0,
			commission         REAL    NOT NULL DEFAULT 0,
			commission_asset   TEXT    NOT NULL DEFAULT '',
			maker              INTEGER NOT NULL DEFAULT 0,
			buyer              INTEGER NOT NULL DEFAULT 0,
			fill_time          INTEGER NOT NULL,
			imported_at        INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fills_symbol_time ON binance_fills(symbol, fill_time)`,
		`CREATE INDEX IF NOT EXISTS idx_fills_time ON binance_fills(fill_time DESC)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate stmt %q: %w", firstLine(s), err)
		}
	}
	return nil
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
