package nav

import (
	"math"
	"testing"
)

const eps = 1e-6

func nearlyEqual(t *testing.T, got, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Errorf("%s: got %.10f, want %.10f (diff %.2e)", label, got, want, math.Abs(got-want))
	}
}

// TestCurrentNAV_BoundaryAndNormal checks the two regimes:
//   - totalShares == 0 → NAV anchored at 1.0 (first deposit ever)
//   - totalShares > 0  → NAV = equity / shares
func TestCurrentNAV_BoundaryAndNormal(t *testing.T) {
	if got := CurrentNAV(0, 0); got != 1.0 {
		t.Errorf("first deposit boundary: got %.4f, want 1.0", got)
	}
	if got := CurrentNAV(1000, 0); got != 1.0 {
		t.Errorf("nonzero equity but zero shares should still anchor at 1.0: got %.4f", got)
	}
	if got := CurrentNAV(1100, 1000); math.Abs(got-1.10) > eps {
		t.Errorf("standard case: got %.4f, want 1.10", got)
	}
	if got := CurrentNAV(2000, 1454.5454545454545); math.Abs(got-1.375) > eps {
		t.Errorf("post-growth case: got %.6f, want 1.375", got)
	}
}

// TestComputeMint_FirstDeposit covers the bootstrap: when total_shares is 0, NAV = 1.0,
// so shares_minted = amount_usdt 1:1.
func TestComputeMint_FirstDeposit(t *testing.T) {
	shares, nav := ComputeMint(1000, 0, 0)
	nearlyEqual(t, shares, 1000.0, "first-deposit shares")
	nearlyEqual(t, nav, 1.0, "first-deposit NAV")
}

// TestComputeMint_LateJoiner covers the plan's canonical example:
// fund equity is $1100 with 1000 outstanding shares (NAV=1.10), friend deposits $500.
func TestComputeMint_LateJoiner(t *testing.T) {
	shares, nav := ComputeMint(500, 1100, 1000)
	nearlyEqual(t, nav, 1.10, "late-joiner NAV")
	nearlyEqual(t, shares, 500.0/1.10, "late-joiner shares minted")
}

// TestComputeBurn covers withdrawal: friend wants out for cash.
func TestComputeBurn_Normal(t *testing.T) {
	shares, nav, err := ComputeBurn(275, 2000, 1454.5454545454545)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nearlyEqual(t, nav, 1.375, "burn NAV")
	nearlyEqual(t, shares, 275.0/1.375, "burn shares (= 200)")
}

func TestComputeBurn_NoSharesIsError(t *testing.T) {
	if _, _, err := ComputeBurn(100, 0, 0); err == nil {
		t.Error("expected error when total_shares is 0, got nil")
	}
}

// TestCanonicalScenario walks the full plan example end to end:
//
//	T0 self deposit $1000  → self holds 1000 shares, NAV=1.0
//	T1 friend A deposits $500 at equity=$1100 (NAV=1.10) → friend ≈ 454.5 shares
//	T2 trading lifts equity to $2000 → NAV = 2000 / 1454.5454 ≈ 1.375
//
// Expected end state:
//
//	friend A: shares ≈ 454.545, value ≈ $625.00, PnL ≈ +25.0%
//	self:     shares = 1000.000, value ≈ $1375.00, PnL ≈ +37.5%
func TestCanonicalScenario(t *testing.T) {
	// T0: self-deposit
	selfShares, _ := ComputeMint(1000, 0, 0)
	totalShares := selfShares

	// T1: friend A deposits $500 at equity=$1100
	friendShares, navAtT1 := ComputeMint(500, 1100, totalShares)
	nearlyEqual(t, navAtT1, 1.10, "T1 NAV")
	nearlyEqual(t, friendShares, 454.5454545454545, "friend A shares minted")
	totalShares += friendShares
	nearlyEqual(t, totalShares, 1454.5454545454545, "total shares after T1")

	// T2: equity moves to $2000
	navAtT2 := CurrentNAV(2000, totalShares)
	nearlyEqual(t, navAtT2, 1.375, "T2 NAV")

	// Friend A's position
	friendValue := friendShares * navAtT2
	friendPnL := friendValue - 500
	friendPnLPct := friendPnL / 500
	nearlyEqual(t, friendValue, 625.0, "friend A value")
	nearlyEqual(t, friendPnL, 125.0, "friend A pnl")
	nearlyEqual(t, friendPnLPct, 0.25, "friend A pnl pct")

	// Self position
	selfValue := selfShares * navAtT2
	selfPnL := selfValue - 1000
	selfPnLPct := selfPnL / 1000
	nearlyEqual(t, selfValue, 1375.0, "self value")
	nearlyEqual(t, selfPnL, 375.0, "self pnl")
	nearlyEqual(t, selfPnLPct, 0.375, "self pnl pct")
}

// TestComputeMintAfterArrival_RealisticAdminFlow covers the typical operational
// flow: friend transfers $500, then admin opens dashboard and records. By that
// time Binance equity already reflects the $500 in the account.
//
// Pre-state: fund had $1100 equity and 1000 shares (NAV = 1.10).
// Friend's $500 has arrived → live equity now $1600.
// Admin records $500 deposit. Expected: shares minted ≈ 454.5 (same as
// ComputeMint with pre-equity), NAV ≈ 1.10.
func TestComputeMintAfterArrival_RealisticAdminFlow(t *testing.T) {
	shares, nav := ComputeMintAfterArrival(500, 1600, 1000)
	nearlyEqual(t, nav, 1.10, "post-arrival NAV")
	nearlyEqual(t, shares, 500.0/1.10, "post-arrival shares minted")
}

// TestComputeMintAfterArrival_Bootstrap covers the empty-pool case:
// when total_shares is 0, NAV is 1.0 and we mint amount 1:1, treating the
// live equity as "the bootstrap deposit IS the account state".
func TestComputeMintAfterArrival_Bootstrap(t *testing.T) {
	shares, nav := ComputeMintAfterArrival(1000, 1000, 0)
	nearlyEqual(t, nav, 1.0, "bootstrap NAV")
	nearlyEqual(t, shares, 1000.0, "bootstrap shares")
}

// TestComputeBurnAfterDeparture covers admin recording a withdrawal after the
// USDT has already left the account.
//
// Pre-state: $2000 equity, 1454.545 shares (NAV = 1.375).
// $275 USDT has left → live equity now $1725.
// Admin records $275 withdrawal. Expected: burn 200 shares, NAV ≈ 1.375.
func TestComputeBurnAfterDeparture(t *testing.T) {
	burned, nav, err := ComputeBurnAfterDeparture(275, 1725, 1454.5454545454545)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nearlyEqual(t, nav, 1.375, "post-departure NAV")
	nearlyEqual(t, burned, 200.0, "post-departure shares burned")
}

// TestRoundTrip_DepositThenFullWithdraw confirms shares math is symmetric:
// a friend deposits then immediately withdraws at the same NAV → zero residual shares.
func TestRoundTrip_DepositThenFullWithdraw(t *testing.T) {
	shares, navIn := ComputeMint(500, 1100, 1000)
	burned, navOut, err := ComputeBurn(500, 1100+500, 1000+shares) // equity grew by deposit
	if err != nil {
		t.Fatalf("burn failed: %v", err)
	}
	// At identical NAV, burned shares should equal minted shares.
	nearlyEqual(t, navIn, navOut, "round-trip NAV")
	nearlyEqual(t, burned, shares, "round-trip shares")
}
