package store

import (
	"context"
	"database/sql"
	"time"
)

// BinanceFill is one Binance USD-M futures trade execution as stored in fund.db.
//
// This is the dashboard's own canonical record of trade history — pulled from
// /fapi/v1/userTrades during initial backfill and kept current by the hourly
// scheduler's TradesSync job. The dashboard does not read trade data from any
// other source (no nofx coupling).
type BinanceFill struct {
	ID              int64
	BinanceTradeID  int64
	BinanceOrderID  int64
	Symbol          string
	Side            string // BUY | SELL
	PositionSide    string // BOTH | LONG | SHORT
	Price           float64
	Qty             float64
	QuoteQty        float64
	RealizedPnL     float64
	Commission      float64
	CommissionAsset string
	Maker           bool
	Buyer           bool
	FillTime        int64 // unix ms — Binance's trade time
	ImportedAt      int64 // unix ms — when this row was written to fund.db
}

// InsertFillIgnore writes one fill, skipping silently if binance_trade_id is
// already present. Returns (rowID, true) when newly inserted, (0, false) when
// it was a duplicate.
func InsertFillIgnore(ctx context.Context, db *sql.DB, f BinanceFill) (int64, bool, error) {
	if f.ImportedAt == 0 {
		f.ImportedAt = time.Now().UnixMilli()
	}
	res, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO binance_fills (
			binance_trade_id, binance_order_id, symbol, side, position_side,
			price, qty, quote_qty, realized_pnl, commission, commission_asset,
			maker, buyer, fill_time, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		f.BinanceTradeID, f.BinanceOrderID, f.Symbol, f.Side, f.PositionSide,
		f.Price, f.Qty, f.QuoteQty, f.RealizedPnL, f.Commission, f.CommissionAsset,
		boolInt(f.Maker), boolInt(f.Buyer), f.FillTime, f.ImportedAt,
	)
	if err != nil {
		return 0, false, err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return 0, false, nil
	}
	id, _ := res.LastInsertId()
	return id, true, nil
}

// ListFillsSince returns all fills with fill_time >= sinceMs, oldest first.
// Used by the position-reconstruction orchestrator.
func ListFillsSince(ctx context.Context, db *sql.DB, sinceMs int64) ([]BinanceFill, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, binance_trade_id, binance_order_id, symbol, side, position_side,
		       price, qty, quote_qty, realized_pnl, commission, commission_asset,
		       maker, buyer, fill_time, imported_at
		FROM binance_fills
		WHERE fill_time >= ?
		ORDER BY fill_time ASC
	`, sinceMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFills(rows)
}

// ListRecentFills returns up to limit fills, newest first. Used by the admin
// "recent fills" view.
func ListRecentFills(ctx context.Context, db *sql.DB, limit int) ([]BinanceFill, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, binance_trade_id, binance_order_id, symbol, side, position_side,
		       price, qty, quote_qty, realized_pnl, commission, commission_asset,
		       maker, buyer, fill_time, imported_at
		FROM binance_fills
		ORDER BY fill_time DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFills(rows)
}

// LastFillTimeBySymbol returns the newest fill_time for `symbol`, or 0 if no
// rows exist. Used by incremental sync to fetch only fills newer than what we
// already have.
func LastFillTimeBySymbol(ctx context.Context, db *sql.DB, symbol string) (int64, error) {
	var ts sql.NullInt64
	err := db.QueryRowContext(ctx,
		`SELECT MAX(fill_time) FROM binance_fills WHERE symbol = ?`, symbol,
	).Scan(&ts)
	if err != nil {
		return 0, err
	}
	if !ts.Valid {
		return 0, nil
	}
	return ts.Int64, nil
}

// LastFillTime returns the newest fill_time across all symbols.
func LastFillTime(ctx context.Context, db *sql.DB) (int64, error) {
	var ts sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT MAX(fill_time) FROM binance_fills`).Scan(&ts)
	if err != nil {
		return 0, err
	}
	if !ts.Valid {
		return 0, nil
	}
	return ts.Int64, nil
}

// DistinctSymbolsTouched returns every symbol that has at least one fill in
// the DB. Used by incremental sync to know which symbols to poll.
func DistinctSymbolsTouched(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT symbol FROM binance_fills ORDER BY symbol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanFills(rows *sql.Rows) ([]BinanceFill, error) {
	var out []BinanceFill
	for rows.Next() {
		var f BinanceFill
		var maker, buyer int
		if err := rows.Scan(
			&f.ID, &f.BinanceTradeID, &f.BinanceOrderID, &f.Symbol, &f.Side, &f.PositionSide,
			&f.Price, &f.Qty, &f.QuoteQty, &f.RealizedPnL, &f.Commission, &f.CommissionAsset,
			&maker, &buyer, &f.FillTime, &f.ImportedAt,
		); err != nil {
			return nil, err
		}
		f.Maker = maker != 0
		f.Buyer = buyer != 0
		out = append(out, f)
	}
	return out, rows.Err()
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
