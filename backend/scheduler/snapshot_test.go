package scheduler

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/nav"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// These tests exercise the divergence guardrail math against a real temp DB.
// The Binance HTTP call inside RunOnce is not mocked — we test the guardrail
// path by setting up the share ledger + most-recent snapshot, then computing
// a candidate NAV the way RunOnce would and asserting the deviation crosses
// (or doesn't cross) maxHourlyNAVJump.

func TestGuardrail_RejectsCliffJump(t *testing.T) {
	ctx := context.Background()
	db := mustOpenTempDB(t)
	defer db.Close()

	mustInsertSnapshot(t, ctx, db, 1_700_000_000_000, 100, 100, 1.0)

	// Candidate: equity dropped to 10 but shares stayed 100 → NAV 0.10
	// (= 90% drop). Real-world: this only happens if the ledger is missing a
	// withdrawal that took 90 USDT out, not a market event.
	candidate := nav.CurrentNAV(10, 100)
	if candidate != 0.10 {
		t.Fatalf("candidate NAV math broke: got %v want 0.10", candidate)
	}
	last, err := store.LatestNAV(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	dev := math.Abs(candidate-last.NAV) / last.NAV
	if dev <= maxHourlyNAVJump {
		t.Errorf("90%% drop should exceed guardrail (%.0f%%), got %.2f", maxHourlyNAVJump*100, dev*100)
	}
}

func TestGuardrail_AllowsNormalMove(t *testing.T) {
	ctx := context.Background()
	db := mustOpenTempDB(t)
	defer db.Close()

	mustInsertSnapshot(t, ctx, db, 1_700_000_000_000, 100, 100, 1.0)

	// 2% gain — typical hourly market noise, must NOT trip the guardrail.
	candidate := nav.CurrentNAV(102, 100)
	last, _ := store.LatestNAV(ctx, db)
	dev := math.Abs(candidate-last.NAV) / last.NAV
	if dev > maxHourlyNAVJump {
		t.Errorf("2%% normal move tripped guardrail (dev=%.4f)", dev)
	}
}

func TestGuardrail_EdgeAt30Pct(t *testing.T) {
	ctx := context.Background()
	db := mustOpenTempDB(t)
	defer db.Close()

	mustInsertSnapshot(t, ctx, db, 1_700_000_000_000, 100, 100, 1.0)

	// Exactly 30% jump (NAV 1.0 → 1.30) — the guardrail is strictly > 30%
	// so this is the boundary. Allowed.
	candidate := nav.CurrentNAV(130, 100)
	last, _ := store.LatestNAV(ctx, db)
	dev := math.Abs(candidate-last.NAV) / last.NAV
	if dev > maxHourlyNAVJump+1e-9 {
		t.Errorf("30%% edge case shouldn't trip: dev=%.6f vs threshold %.2f", dev, maxHourlyNAVJump)
	}
}

func mustOpenTempDB(t *testing.T) *sql.DB {
	t.Helper()
	path := t.TempDir() + "/fund.db"
	db, err := store.Open(path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	return db
}

func mustInsertSnapshot(t *testing.T, ctx context.Context, db *sql.DB, ts int64, equity, shares, n float64) {
	t.Helper()
	if _, err := store.InsertNAVSnapshot(ctx, db, store.NAVSnapshot{
		TakenAt: ts, TotalEquityUSDT: equity, TotalShares: shares, NAV: n,
		Source: store.SnapshotScheduled,
	}); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
}
