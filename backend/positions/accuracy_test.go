package positions

import (
	"testing"

	"github.com/xiagao/fund-dashboard/backend/binance"
)

// Regression tests for the closed-position accounting bugs found on 2026-08-30,
// where the closed panel reported +34.26 USDT while the account had actually
// realized -8.09. See each test for the specific defect it pins down.

// Binance stamps several fills of one order with the same millisecond (44 such
// groups in the live data). Sorting by time alone with an unstable sort let the
// walk see them in arbitrary order, changing the round-trip count. Trade IDs are
// monotonic in execution order, so they must break the tie.
func TestDerive_SameMillisecondTiesAreDeterministic(t *testing.T) {
	// A close and an immediate re-open share t=2000.
	fills := []binance.UserTrade{
		{ID: 1, Symbol: "NVDAUSDT", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 100, QuoteQty: 100, Time: 1_000},
		{ID: 2, Symbol: "NVDAUSDT", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 110, QuoteQty: 110, RealizedPnL: 10, Time: 2_000},
		{ID: 3, Symbol: "NVDAUSDT", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 111, QuoteQty: 111, Time: 2_000},
		{ID: 4, Symbol: "NVDAUSDT", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 120, QuoteQty: 120, RealizedPnL: 9, Time: 3_000},
	}
	shuffled := []binance.UserTrade{fills[0], fills[2], fills[1], fills[3]} // ties swapped

	a, aOpen := Derive(fills)
	b, bOpen := Derive(shuffled)

	if len(a) != len(b) || len(aOpen) != len(bOpen) {
		t.Fatalf("tie order changed the result: %d closed/%d open vs %d closed/%d open",
			len(a), len(aOpen), len(b), len(bOpen))
	}
	// Execution order (by ID) is close-then-reopen → two distinct round trips.
	if len(a) != 2 {
		t.Errorf("want 2 round trips in trade-ID order, got %d", len(a))
	}
	for i := range a {
		if a[i].RealizedPnL != b[i].RealizedPnL || a[i].EntryQuantity != b[i].EntryQuantity {
			t.Errorf("cycle %d differs between orderings: %+v vs %+v", i, a[i], b[i])
		}
	}
}

// A position that is still open can already have booked realized PnL from
// partial closes. That money is real and must be surfaced on the open position,
// otherwise it is invisible everywhere (-55.97 USDT of it in the live account).
func TestBuildOpenViews_CarriesRealizedPnLAndCommission(t *testing.T) {
	risks := []binance.PositionRisk{
		{Symbol: "NVDAUSDT", PositionAmt: 0.6, EntryPrice: 207.82, MarkPrice: 210,
			UnRealizedProfit: 1.3, PositionSide: "LONG", UpdateTime: 9_000},
	}
	residuals := []Lifecycle{
		{Symbol: "NVDAUSDT", DirectionalSide: "LONG", EntryTime: 1_000,
			EntryQuantity: 1.78, ExitQuantity: 1.18, RealizedPnL: 9.5086, Commission: 0.42, Open: true},
	}

	views := buildOpenViews(risks, residuals)
	if len(views) != 1 {
		t.Fatalf("want 1 open view, got %d", len(views))
	}
	v := views[0]
	if v.EntryTime != 1_000 {
		t.Errorf("entry time should come from the fill history: got %d", v.EntryTime)
	}
	nearly(t, v.RealizedPnL, 9.5086, "open position's booked realized PnL")
	nearly(t, v.Commission, 0.42, "open position's commission")
	nearly(t, v.UnrealizedPnL, 1.3, "unrealized still from positionRisk")
}

// Fees are 3.64 USDT against -8.09 of realized PnL in the live account, so a
// PnL figure that ignores them is materially wrong. Closed views must carry it.
func TestBuildClosedViews_IncludesCommission(t *testing.T) {
	cycles := []Lifecycle{
		{Symbol: "SPYUSDT", DirectionalSide: "LONG", EntryQuantity: 10, ExitQuantity: 10,
			EntryPrice: 100, ExitPrice: 101, EntryTime: 1_000, ExitTime: 2_000,
			RealizedPnL: 10, Commission: 0.804},
	}
	views := buildClosedViews(cycles)
	if len(views) != 1 {
		t.Fatalf("want 1 closed view, got %d", len(views))
	}
	nearly(t, views[0].RealizedPnL, 10, "gross realized")
	nearly(t, views[0].Commission, 0.804, "commission must be surfaced")
}

// A trade that grosses +0.05 but pays 0.08 in fees lost money. Classifying it
// as a win (and summing gross) inflated both the win rate and the total.
func TestComputeStats_UsesNetOfFees(t *testing.T) {
	closed := []View{
		{Symbol: "A", RealizedPnL: 0.05, Commission: 0.08, EntryTime: 1_000, ExitTime: 2_000},
		{Symbol: "B", RealizedPnL: 5.00, Commission: 0.50, EntryTime: 1_000, ExitTime: 2_000},
	}
	s := ComputeStats(closed)
	if s.Wins != 1 || s.Losses != 1 {
		t.Errorf("fee-eaten trade must count as a loss: wins=%d losses=%d", s.Wins, s.Losses)
	}
	nearly(t, s.TotalPnL, 0.05-0.08+5.00-0.50, "total must be net of fees")
}

// Per-symbol aggregation must use the same net basis as the stats panel,
// otherwise the two views of the same trades disagree.
func TestAggregateBySymbol_UsesNetOfFees(t *testing.T) {
	rows := AggregateBySymbol([]View{
		{Symbol: "A", RealizedPnL: 0.05, Commission: 0.08},
	})
	if len(rows) != 1 {
		t.Fatalf("want 1 symbol row, got %d", len(rows))
	}
	if rows[0].Wins != 0 {
		t.Errorf("fee-eaten trade must not count as a win, got wins=%d", rows[0].Wins)
	}
	nearly(t, rows[0].TotalPnL, -0.03, "symbol total must be net")
}

// The reconstruction walks a position from birth. Starting the walk in the
// middle of an open position (the old 90-day window) cut off entry legs and
// fabricated completed round trips out of still-open positions, so the default
// must be "no truncation".
func TestOrchestrator_ReconstructsFromFullHistoryByDefault(t *testing.T) {
	o := &Orchestrator{}
	if got := o.lookback(); got != 0 {
		t.Errorf("default lookback must be 0 (full history), got %v", got)
	}
}
