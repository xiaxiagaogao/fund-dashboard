// Package scheduler hosts the long-running background jobs.
//
// Lives in its own package so nav/ stays pure-math (no I/O deps) and store/
// stays pure-data-layer (no nav deps) — keeps the test graph free of cycles.
package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/nav"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// maxHourlyNAVJump is the divergence guardrail. If a candidate scheduled
// snapshot would move NAV by more than this fraction from the previous
// snapshot, we SKIP the write and log a loud warning instead.
//
// Why: for a small friend fund (no leverage past 5x, sane position sizing),
// a 30% NAV change in one hour is almost certainly NOT a real market move —
// it's much more likely:
//   - admin forgot to record a recent deposit/withdrawal (ledger drift)
//   - the dashboard is running against a demo/synthetic share ledger
//   - a manual override (manual_nav) set a non-realistic NAV earlier
//
// Skipping protects the equity curve from a fake cliff that would worry
// friends, at the cost of pausing automated tracking until admin investigates.
// Manual snapshots via /api/admin/snapshot bypass this check so the operator
// can still write a snapshot once they've confirmed the divergence is real.
const maxHourlyNAVJump = 0.30

type SnapshotJob struct {
	DB      *sql.DB
	Binance *binance.Client
}

// RunOnce writes one nav_snapshots row reflecting current Binance equity and
// the share ledger. Idempotent at millisecond granularity.
func (j *SnapshotJob) RunOnce(ctx context.Context) error {
	if j.Binance == nil {
		return fmt.Errorf("snapshot: binance client not configured")
	}
	equity, _, err := j.Binance.AccountEquity(ctx)
	if err != nil {
		return fmt.Errorf("snapshot: binance equity: %w", err)
	}
	totalShares, err := store.TotalShares(ctx, j.DB)
	if err != nil {
		return fmt.Errorf("snapshot: total shares: %w", err)
	}
	currentNAV := nav.CurrentNAV(equity, totalShares)
	now := time.Now().UnixMilli()

	// Divergence guardrail: refuse to write a snapshot that would create a
	// suspicious NAV jump. See maxHourlyNAVJump for rationale.
	if last, err := store.LatestNAV(ctx, j.DB); err == nil && last.NAV > 0 {
		dev := math.Abs(currentNAV-last.NAV) / last.NAV
		if dev > maxHourlyNAVJump {
			log.Printf("snapshot: SKIPPED (divergence %.1f%% > %.0f%%) — candidate NAV %.6f vs last %.6f; equity=%.2f shares=%.4f. Likely ledger/equity drift (forgotten deposit?); investigate before next snapshot.",
				dev*100, maxHourlyNAVJump*100, currentNAV, last.NAV, equity, totalShares)
			store.WriteAudit(ctx, j.DB, "system", "snapshot.skipped_divergent", map[string]any{
				"candidate_nav": currentNAV, "last_nav": last.NAV, "deviation": dev,
				"equity": equity, "shares": totalShares,
			})
			return nil
		}
	}

	if _, err := store.InsertNAVSnapshot(ctx, j.DB, store.NAVSnapshot{
		TakenAt: now, TotalEquityUSDT: equity, TotalShares: totalShares,
		NAV: currentNAV, Source: store.SnapshotScheduled,
	}); err != nil && !errors.Is(err, store.ErrDuplicateSnapshot) {
		return fmt.Errorf("snapshot: insert: %w", err)
	}
	store.WriteAudit(ctx, j.DB, "system", "snapshot.write", map[string]any{
		"taken_at": now, "equity": equity, "shares": totalShares, "nav": currentNAV,
	})
	return nil
}

// Start launches an hourly snapshot loop until ctx is cancelled. Aligns to the
// top of each hour so points line up cleanly on the chart.
func (j *SnapshotJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		// Run one immediately so a fresh container has a baseline row.
		if err := j.RunOnce(ctx); err != nil {
			log.Printf("snapshot: initial run failed: %v", err)
		}
		next := time.Now().Truncate(interval).Add(interval)
		t := time.NewTimer(time.Until(next))
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := j.RunOnce(ctx); err != nil {
					log.Printf("snapshot: run failed: %v", err)
				}
				next = next.Add(interval)
				t.Reset(time.Until(next))
			}
		}
	}()
}
