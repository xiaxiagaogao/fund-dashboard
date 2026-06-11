package positions

import (
	"math"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/binance"
)

func nearly(t *testing.T, got, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s: got %v want %v", label, got, want)
	}
}

// One symbol, one round trip: BUY 1 @100, SELL 1 @110, PnL +10.
func TestDerive_SimpleLongRoundTrip(t *testing.T) {
	trades := []binance.UserTrade{
		{Symbol: "TSLAUSDT", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 100, QuoteQty: 100, Time: 1000},
		{Symbol: "TSLAUSDT", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 110, QuoteQty: 110, RealizedPnL: 10, Time: 2000},
	}
	closed, open := Derive(trades)
	if len(closed) != 1 || len(open) != 0 {
		t.Fatalf("want 1 closed 0 open, got %d/%d", len(closed), len(open))
	}
	c := closed[0]
	if c.Symbol != "TSLAUSDT" || c.DirectionalSide != "LONG" {
		t.Errorf("symbol/side wrong: %+v", c)
	}
	nearly(t, c.EntryPrice, 100, "entry price")
	nearly(t, c.ExitPrice, 110, "exit price")
	nearly(t, c.RealizedPnL, 10, "pnl")
	nearly(t, c.EntryQuantity, 1, "entry qty")
	nearly(t, c.ExitQuantity, 1, "exit qty")
	if c.EntryTime != 1000 || c.ExitTime != 2000 {
		t.Errorf("times wrong: %+v", c)
	}
}

// Scale-in then full close: BUY 1 @100, BUY 1 @120, SELL 2 @130 → entry avg 110, pnl 40.
func TestDerive_PyramidScaleIn(t *testing.T) {
	trades := []binance.UserTrade{
		{Symbol: "NVDA", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 100, QuoteQty: 100, Time: 1000},
		{Symbol: "NVDA", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 120, QuoteQty: 120, Time: 2000},
		{Symbol: "NVDA", Side: "SELL", PositionSide: "LONG", Qty: 2, Price: 130, QuoteQty: 260, RealizedPnL: 40, Time: 3000},
	}
	closed, open := Derive(trades)
	if len(closed) != 1 || len(open) != 0 {
		t.Fatalf("want 1 closed 0 open, got %d/%d", len(closed), len(open))
	}
	c := closed[0]
	nearly(t, c.EntryPrice, 110, "entry avg")
	nearly(t, c.ExitPrice, 130, "exit price")
	nearly(t, c.RealizedPnL, 40, "pnl")
	nearly(t, c.EntryQuantity, 2, "total entry qty")
	if c.NumOpeningFills != 2 || c.NumClosingFills != 1 {
		t.Errorf("fill counts wrong: opens=%d closes=%d", c.NumOpeningFills, c.NumClosingFills)
	}
}

// Partial close then second close: BUY 2 @100, SELL 1 @110 (PnL+10), SELL 1 @90 (PnL-10).
// Should produce ONE round trip with weighted-avg exit (110*1 + 90*1)/2 = 100, PnL=0.
func TestDerive_PartialThenFullClose(t *testing.T) {
	trades := []binance.UserTrade{
		{Symbol: "X", Side: "BUY", PositionSide: "LONG", Qty: 2, Price: 100, QuoteQty: 200, Time: 1000},
		{Symbol: "X", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 110, QuoteQty: 110, RealizedPnL: 10, Time: 2000},
		{Symbol: "X", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 90, QuoteQty: 90, RealizedPnL: -10, Time: 3000},
	}
	closed, open := Derive(trades)
	if len(closed) != 1 || len(open) != 0 {
		t.Fatalf("want 1 closed 0 open, got %d/%d", len(closed), len(open))
	}
	c := closed[0]
	nearly(t, c.EntryPrice, 100, "entry")
	nearly(t, c.ExitPrice, 100, "exit weighted avg")
	nearly(t, c.RealizedPnL, 0, "pnl")
	nearly(t, c.ExitQuantity, 2, "exit qty")
	if c.ExitTime != 3000 {
		t.Errorf("exit_time should be last closing fill, got %d", c.ExitTime)
	}
}

// Short round trip: SELL 1 @110, BUY 1 @100, PnL +10.
func TestDerive_SimpleShortRoundTrip(t *testing.T) {
	trades := []binance.UserTrade{
		{Symbol: "X", Side: "SELL", PositionSide: "SHORT", Qty: 1, Price: 110, QuoteQty: 110, Time: 1000},
		{Symbol: "X", Side: "BUY", PositionSide: "SHORT", Qty: 1, Price: 100, QuoteQty: 100, RealizedPnL: 10, Time: 2000},
	}
	closed, open := Derive(trades)
	if len(closed) != 1 || len(open) != 0 {
		t.Fatalf("want 1 closed, got %d/%d", len(closed), len(open))
	}
	if closed[0].DirectionalSide != "SHORT" {
		t.Errorf("side should be SHORT, got %v", closed[0].DirectionalSide)
	}
	nearly(t, closed[0].RealizedPnL, 10, "pnl")
}

// Open residual: BUY 1, no close yet.
func TestDerive_OpenResidual(t *testing.T) {
	trades := []binance.UserTrade{
		{Symbol: "Y", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 100, QuoteQty: 100, Time: 1000},
	}
	closed, open := Derive(trades)
	if len(closed) != 0 || len(open) != 1 {
		t.Fatalf("want 0/1, got %d/%d", len(closed), len(open))
	}
	if !open[0].Open {
		t.Errorf("residual should have Open=true")
	}
}

// Multiple round trips on the same symbol in sequence.
func TestDerive_TwoRoundTrips(t *testing.T) {
	trades := []binance.UserTrade{
		{Symbol: "Z", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 100, QuoteQty: 100, Time: 1000},
		{Symbol: "Z", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 110, QuoteQty: 110, RealizedPnL: 10, Time: 2000},
		{Symbol: "Z", Side: "BUY", PositionSide: "LONG", Qty: 1, Price: 105, QuoteQty: 105, Time: 3000},
		{Symbol: "Z", Side: "SELL", PositionSide: "LONG", Qty: 1, Price: 100, QuoteQty: 100, RealizedPnL: -5, Time: 4000},
	}
	closed, open := Derive(trades)
	if len(closed) != 2 || len(open) != 0 {
		t.Fatalf("want 2/0, got %d/%d", len(closed), len(open))
	}
	// Sorted desc by exit_time → second trip first.
	nearly(t, closed[0].RealizedPnL, -5, "second pnl (most recent)")
	nearly(t, closed[1].RealizedPnL, 10, "first pnl")
}

// Direction flip in one-way mode (positionSide=BOTH): long 5, then a single
// SELL 8 that closes the 5-lot long AND opens a 3-lot short, then BUY 3 to
// close the short. Must yield TWO round trips with prorated quote/commission —
// not one mangled LONG cycle (the pre-fix behavior).
func TestDerive_DirectionFlipSplitsCycles(t *testing.T) {
	trades := []binance.UserTrade{
		{ID: 1, Symbol: "BTCUSDT", Side: "BUY", PositionSide: "BOTH", Price: 100, Qty: 5, QuoteQty: 500, Commission: 0.5, Time: 1000},
		// SELL 8 @110: 5 closes the long (realized +50), 3 opens a short.
		{ID: 2, Symbol: "BTCUSDT", Side: "SELL", PositionSide: "BOTH", Price: 110, Qty: 8, QuoteQty: 880, RealizedPnL: 50, Commission: 0.8, Time: 2000},
		// BUY 3 @105 closes the short (realized +15).
		{ID: 3, Symbol: "BTCUSDT", Side: "BUY", PositionSide: "BOTH", Price: 105, Qty: 3, QuoteQty: 315, RealizedPnL: 15, Commission: 0.3, Time: 3000},
	}
	closed, open := Derive(trades)
	if len(open) != 0 {
		t.Fatalf("open: got %d want 0 (%+v)", len(open), open)
	}
	if len(closed) != 2 {
		t.Fatalf("closed: got %d want 2 (%+v)", len(closed), closed)
	}
	// closed is sorted by exit time desc → [0] = short, [1] = long.
	short, long := closed[0], closed[1]

	if long.DirectionalSide != "LONG" {
		t.Errorf("first cycle side: got %s want LONG", long.DirectionalSide)
	}
	nearly(t, long.EntryQuantity, 5, "long entry qty")
	nearly(t, long.ExitQuantity, 5, "long exit qty")
	nearly(t, long.EntryPrice, 100, "long entry px")
	nearly(t, long.ExitPrice, 110, "long exit px")
	nearly(t, long.RealizedPnL, 50, "long pnl")
	// Commission: 0.5 (open) + 5/8 of 0.8 (close leg) = 1.0
	nearly(t, long.Commission, 1.0, "long commission")
	if long.EntryTime != 1000 || long.ExitTime != 2000 {
		t.Errorf("long times: entry %d exit %d", long.EntryTime, long.ExitTime)
	}

	if short.DirectionalSide != "SHORT" {
		t.Errorf("second cycle side: got %s want SHORT", short.DirectionalSide)
	}
	nearly(t, short.EntryQuantity, 3, "short entry qty")
	nearly(t, short.ExitQuantity, 3, "short exit qty")
	nearly(t, short.EntryPrice, 110, "short entry px")
	nearly(t, short.ExitPrice, 105, "short exit px")
	nearly(t, short.RealizedPnL, 15, "short pnl")
	// Commission: 3/8 of 0.8 (open leg) + 0.3 (close) = 0.6
	nearly(t, short.Commission, 0.6, "short commission")
	if short.EntryTime != 2000 || short.ExitTime != 3000 {
		t.Errorf("short times: entry %d exit %d", short.EntryTime, short.ExitTime)
	}
}

// A flip whose residual stays open: long 2, SELL 5 → short 3 still on book.
func TestDerive_FlipResidualStaysOpen(t *testing.T) {
	trades := []binance.UserTrade{
		{ID: 1, Symbol: "ETHUSDT", Side: "BUY", PositionSide: "BOTH", Price: 100, Qty: 2, QuoteQty: 200, Time: 1000},
		{ID: 2, Symbol: "ETHUSDT", Side: "SELL", PositionSide: "BOTH", Price: 120, Qty: 5, QuoteQty: 600, RealizedPnL: 40, Time: 2000},
	}
	closed, open := Derive(trades)
	if len(closed) != 1 || len(open) != 1 {
		t.Fatalf("got closed=%d open=%d, want 1/1", len(closed), len(open))
	}
	nearly(t, closed[0].RealizedPnL, 40, "closed pnl")
	nearly(t, closed[0].ExitQuantity, 2, "closed exit qty")
	if open[0].DirectionalSide != "SHORT" {
		t.Errorf("residual side: got %s want SHORT", open[0].DirectionalSide)
	}
	nearly(t, open[0].EntryQuantity, 3, "residual qty")
	nearly(t, open[0].EntryPrice, 120, "residual entry px")
}
