package store

import (
	"context"
	"database/sql"
	"time"
)

// IndexClose is one daily close of a benchmark symbol.
type IndexClose struct {
	Symbol string
	DayMs  int64
	Close  float64
}

// UpsertIndexClose writes (or refreshes) one daily close. Idempotent on
// (symbol, day_ms) so re-running the daily sync is cheap and safe.
func UpsertIndexClose(ctx context.Context, db *sql.DB, symbol string, dayMs int64, close float64) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO index_prices (symbol, day_ms, close, fetched_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(symbol, day_ms) DO UPDATE SET close = excluded.close, fetched_at = excluded.fetched_at`,
		symbol, dayMs, close, time.Now().UnixMilli(),
	)
	return err
}

// ListIndexCloses returns a symbol's daily closes with day_ms in [fromMs, toMs],
// ascending. Used by the index-comparison endpoint.
func ListIndexCloses(ctx context.Context, db *sql.DB, symbol string, fromMs, toMs int64) ([]IndexClose, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT symbol, day_ms, close FROM index_prices
		 WHERE symbol = ? AND day_ms BETWEEN ? AND ? ORDER BY day_ms ASC`,
		symbol, fromMs, toMs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndexClose
	for rows.Next() {
		var c IndexClose
		if err := rows.Scan(&c.Symbol, &c.DayMs, &c.Close); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
