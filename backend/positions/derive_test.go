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
