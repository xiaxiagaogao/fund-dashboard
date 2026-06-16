package positions

import (
	"math"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/binance"
)

func TestBuildAllocation(t *testing.T) {
	summ := binance.AccountSummary{Equity: 1000, MarginUsed: 350, UpdateTime: 42}
	risks := []binance.PositionRisk{
		{Symbol: "BTCUSDT", PositionAmt: 0.02, MarkPrice: 50000, PositionSide: "BOTH"}, // notional 1000, LONG
		{Symbol: "ETHUSDT", PositionAmt: -0.2, MarkPrice: 3000, PositionSide: "BOTH"},  // notional 600, SHORT
		{Symbol: "ZEROUSDT", PositionAmt: 0, MarkPrice: 10, PositionSide: "BOTH"},      // dropped
	}
	a := buildAllocation(summ, risks)

	nearly(t, a.Equity, 1000, "equity")
	nearly(t, a.MarginUsed, 350, "margin used")
	nearly(t, a.FreeCash, 650, "free cash = equity - margin")
	nearly(t, a.Notional, 1600, "total notional")
	nearly(t, a.Leverage, 1.6, "leverage = notional/equity")
	if a.UpdateTime != 42 {
		t.Errorf("update time: got %d want 42", a.UpdateTime)
	}
	if len(a.Positions) != 2 {
		t.Fatalf("positions: got %d want 2 (zero-notional dropped)", len(a.Positions))
	}
	// Sorted by notional desc → BTC first.
	if a.Positions[0].Symbol != "BTCUSDT" || a.Positions[0].Side != "LONG" {
		t.Errorf("first slice: %+v", a.Positions[0])
	}
	nearly(t, a.Positions[0].Pct, 1000.0/1600.0, "btc pct")
	if a.Positions[1].Symbol != "ETHUSDT" || a.Positions[1].Side != "SHORT" {
		t.Errorf("second slice: %+v", a.Positions[1])
	}
	nearly(t, a.Positions[1].Pct, 600.0/1600.0, "eth pct")
	// pct sums to 1.
	if math.Abs(a.Positions[0].Pct+a.Positions[1].Pct-1) > 1e-9 {
		t.Errorf("pcts don't sum to 1")
	}
}

func TestBuildAllocation_Flat(t *testing.T) {
	a := buildAllocation(binance.AccountSummary{Equity: 500, MarginUsed: 0}, nil)
	nearly(t, a.FreeCash, 500, "all cash")
	nearly(t, a.Leverage, 0, "no leverage when flat")
	if len(a.Positions) != 0 {
		t.Errorf("expected no positions, got %d", len(a.Positions))
	}
}
