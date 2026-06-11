package store

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/nav"
)

// openTestDB creates a fresh fund.db in a temp dir, runs migrations,
// and registers cleanup. The DB is on disk (not :memory:) so we exercise the
// real WAL + pragma path.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "fund.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestCanonicalScenario_DBLayer mirrors the nav package's canonical scenario but
// drives it through the actual DB layer end-to-end. If this passes, the schema
// + insert/query helpers correctly implement the share-accounting design.
//
//	T0 self-deposit $1000  → self holds 1000 shares, NAV=1.0
//	T1 friend A deposits $500 at equity=$1100 (NAV=1.10) → ≈454.5 shares
//	T2 trading lifts equity to $2000 → NAV ≈ 1.375
func TestCanonicalScenario_DBLayer(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	selfID, err := CreateFriend(ctx, db, "Self (operator)", "operator", "irrelevant", true)
	if err != nil {
		t.Fatalf("CreateFriend self: %v", err)
	}
	friendID, err := CreateFriend(ctx, db, "Friend A", "alice", "irrelevant", false)
	if err != nil {
		t.Fatalf("CreateFriend friend: %v", err)
	}

	// T0: self deposits $1000 into an empty pool.
	totalShares, err := TotalShares(ctx, db)
	if err != nil {
		t.Fatalf("TotalShares: %v", err)
	}
	selfMint, navT0 := nav.ComputeMint(1000, 0, totalShares)
	if _, err := InsertCashEvent(ctx, db, CashEventInput{
		FriendID: selfID, Type: EventDeposit, AmountUSDT: 1000, OccurredAt: 1000,
		NAVAtEvent: navT0, SharesDelta: selfMint, Source: SourceManual,
	}); err != nil {
		t.Fatalf("self deposit insert: %v", err)
	}

	// T1: friend deposits $500 at equity=$1100.
	totalShares, _ = TotalShares(ctx, db)
	friendMint, navT1 := nav.ComputeMint(500, 1100, totalShares)
	if _, err := InsertCashEvent(ctx, db, CashEventInput{
		FriendID: friendID, Type: EventDeposit, AmountUSDT: 500, OccurredAt: 2000,
		NAVAtEvent: navT1, SharesDelta: friendMint, Source: SourceManual,
	}); err != nil {
		t.Fatalf("friend deposit insert: %v", err)
	}

	// T2: persist a snapshot when equity is $2000.
	totalShares, _ = TotalShares(ctx, db)
	navT2 := nav.CurrentNAV(2000, totalShares)
	if _, err := InsertNAVSnapshot(ctx, db, NAVSnapshot{
		TakenAt: 3000, TotalEquityUSDT: 2000, TotalShares: totalShares, NAV: navT2, Source: SnapshotScheduled,
	}); err != nil {
		t.Fatalf("snapshot insert: %v", err)
	}

	// Now drive the friend page math via the DB queries.
	friendShares, err := FriendShares(ctx, db, friendID)
	if err != nil {
		t.Fatalf("FriendShares: %v", err)
	}
	friendNet, err := FriendNetDeposits(ctx, db, friendID)
	if err != nil {
		t.Fatalf("FriendNetDeposits: %v", err)
	}
	latest, err := LatestNAV(ctx, db)
	if err != nil {
		t.Fatalf("LatestNAV: %v", err)
	}

	// Friend A's expected end state.
	friendValue := friendShares * latest.NAV
	friendPnLPct := (friendValue - friendNet) / friendNet
	want := map[string]float64{
		"friend shares":     500.0 / 1.10,
		"friend net depos.": 500.0,
		"latest NAV":        1.375,
		"friend value":      625.0,
		"friend pnl %":      0.25,
	}
	got := map[string]float64{
		"friend shares":     friendShares,
		"friend net depos.": friendNet,
		"latest NAV":        latest.NAV,
		"friend value":      friendValue,
		"friend pnl %":      friendPnLPct,
	}
	for k, w := range want {
		if math.Abs(got[k]-w) > 1e-6 {
			t.Errorf("%s: got %.6f, want %.6f", k, got[k], w)
		}
	}

	// Self's end state.
	selfShares, _ := FriendShares(ctx, db, selfID)
	selfNet, _ := FriendNetDeposits(ctx, db, selfID)
	selfValue := selfShares * latest.NAV
	if math.Abs(selfShares-1000) > 1e-6 {
		t.Errorf("self shares: got %.6f, want 1000", selfShares)
	}
	if math.Abs(selfValue-1375) > 1e-6 {
		t.Errorf("self value: got %.6f, want 1375", selfValue)
	}
	if math.Abs((selfValue-selfNet)/selfNet-0.375) > 1e-6 {
		t.Errorf("self pnl%%: got %.6f, want 0.375", (selfValue-selfNet)/selfNet)
	}
}

// TestUniqueBinanceTx ensures the unique partial index actually prevents
// double-counting an auto-pulled Binance transfer.
func TestUniqueBinanceTx(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	fid, _ := CreateFriend(ctx, db, "Bob", "bob", "x", false)

	in := CashEventInput{
		FriendID: fid, Type: EventDeposit, AmountUSDT: 100, OccurredAt: 1,
		NAVAtEvent: 1.0, SharesDelta: 100, Source: SourceBinanceTransfer, BinanceTxID: "tx-abc",
	}
	if _, err := InsertCashEvent(ctx, db, in); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	_, err := InsertCashEvent(ctx, db, in)
	if !errors.Is(err, ErrDuplicateBinanceTx) {
		t.Errorf("expected ErrDuplicateBinanceTx, got %v", err)
	}

	// But two manual entries (no binance_tx_id) for the same friend must coexist.
	in.BinanceTxID = ""
	in.Source = SourceManual
	if _, err := InsertCashEvent(ctx, db, in); err != nil {
		t.Fatalf("manual #1: %v", err)
	}
	if _, err := InsertCashEvent(ctx, db, in); err != nil {
		t.Errorf("manual #2 should not collide: %v", err)
	}
}

// TestUniqueSnapshot ensures duplicate hourly buckets are rejected cleanly.
func TestUniqueSnapshot(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	s := NAVSnapshot{TakenAt: 12345, TotalEquityUSDT: 100, TotalShares: 100, NAV: 1.0, Source: SnapshotScheduled}
	if _, err := InsertNAVSnapshot(ctx, db, s); err != nil {
		t.Fatalf("first snapshot: %v", err)
	}
	if _, err := InsertNAVSnapshot(ctx, db, s); !errors.Is(err, ErrDuplicateSnapshot) {
		t.Errorf("expected ErrDuplicateSnapshot, got %v", err)
	}
}
