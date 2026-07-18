// Package store owns the dashboard's SQLite persistence.
//
// The driver is pure-Go modernc.org/sqlite — no CGO, smaller container, and the
// nofx data.db can be opened with the same driver in read-only mode (see nofx_ro.go).
package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// CheckpointWAL runs a TRUNCATE checkpoint, folding the -wal file back into
// the main DB and resetting it to zero bytes. The dashboard's long-lived read
// connections can starve SQLite's automatic checkpointing (observed: 4MB WAL
// on the VPS while the main file sat untouched for days), so the snapshot
// scheduler calls this periodically. Best-effort — busy_timeout applies.
func CheckpointWAL(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

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
		// Daily closes of tokenized index perps (QQQUSDT, SPYUSDT) used to
		// benchmark the fund NAV against the market. Filled by the daily
		// IndexSyncJob from the public klines endpoint. day_ms is the candle's
		// open time (start of the UTC day).
		`CREATE TABLE IF NOT EXISTS index_prices (
			symbol     TEXT    NOT NULL,
			day_ms     INTEGER NOT NULL,
			close      REAL    NOT NULL,
			fetched_at INTEGER NOT NULL,
			PRIMARY KEY (symbol, day_ms)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate stmt %q: %w", firstLine(s), err)
		}
	}
	// Column additions to existing tables. CREATE TABLE IF NOT EXISTS above
	// won't touch a table that already exists, so evolving columns go here,
	// each guarded so re-running on a DB that already has the column is a no-op.
	// active=1 default means every pre-existing friend stays enabled on upgrade.
	if err := addColumnIfMissing(db, "friends", "active", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	return nil
}

// addColumnIfMissing runs ALTER TABLE ADD COLUMN only when the column isn't
// already present, so migrate() stays idempotent across restarts.
func addColumnIfMissing(db *sql.DB, table, column, ddl string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("table_info(%s): %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, ctype      string
			dfltValue        sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(%s): %w", table, err)
		}
		if name == column {
			return rows.Close() // already present
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, ddl)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
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
