package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	SnapshotScheduled = "scheduled"
	SnapshotCashEvent = "cash_event"
)

type NAVSnapshot struct {
	ID              int64
	TakenAt         int64 // unix ms
	TotalEquityUSDT float64
	TotalShares     float64
	NAV             float64
	Source          string
}

// InsertNAVSnapshot persists one snapshot row. taken_at must be unique;
// duplicates (e.g. two scheduled jobs racing on the same hour bucket)
// return ErrDuplicateSnapshot rather than blowing up.
var ErrDuplicateSnapshot = errors.New("store: snapshot for this taken_at already exists")

func InsertNAVSnapshot(ctx context.Context, db *sql.DB, s NAVSnapshot) (int64, error) {
	res, err := db.ExecContext(ctx,
		`INSERT INTO nav_snapshots(taken_at, total_equity_usdt, total_shares, nav, source) VALUES(?, ?, ?, ?, ?)`,
		s.TakenAt, s.TotalEquityUSDT, s.TotalShares, s.NAV, s.Source,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrDuplicateSnapshot
		}
		return 0, fmt.Errorf("insert nav_snapshot: %w", err)
	}
	return res.LastInsertId()
}

// LatestNAV returns the most recent snapshot, or NAV=1.0 placeholder if the table
// is empty (no deposits yet, no NAV exists).
func LatestNAV(ctx context.Context, db *sql.DB) (NAVSnapshot, error) {
	var s NAVSnapshot
	err := db.QueryRowContext(ctx,
		`SELECT id, taken_at, total_equity_usdt, total_shares, nav, source FROM nav_snapshots ORDER BY taken_at DESC LIMIT 1`,
	).Scan(&s.ID, &s.TakenAt, &s.TotalEquityUSDT, &s.TotalShares, &s.NAV, &s.Source)
	if errors.Is(err, sql.ErrNoRows) {
		return NAVSnapshot{TakenAt: time.Now().UnixMilli(), NAV: 1.0, Source: SnapshotScheduled}, nil
	}
	if err != nil {
		return NAVSnapshot{}, err
	}
	return s, nil
}

// ListNAVSnapshotsRange returns snapshots in [from, to] (unix ms), ordered ASC.
// Used for the equity curve chart.
func ListNAVSnapshotsRange(ctx context.Context, db *sql.DB, from, to int64) ([]NAVSnapshot, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, taken_at, total_equity_usdt, total_shares, nav, source
		 FROM nav_snapshots WHERE taken_at BETWEEN ? AND ? ORDER BY taken_at ASC`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NAVSnapshot
	for rows.Next() {
		var s NAVSnapshot
		if err := rows.Scan(&s.ID, &s.TakenAt, &s.TotalEquityUSDT, &s.TotalShares, &s.NAV, &s.Source); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
