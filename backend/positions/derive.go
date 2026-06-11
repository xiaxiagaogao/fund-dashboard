// Package positions reconstructs position lifecycles from a stream of Binance
// fills. Binance gives us per-fill data (price, qty, realizedPnL, time); the
// dashboard needs grouped entry→exit cycles with hold duration and weighted
// average prices.
//
// Algorithm (per (symbol, positionSide) group):
//   1. Sort fills by time ascending.
//   2. Maintain a running net quantity.
//   3. A fill that grows |running_qty| in the position's direction is "opening".
//      A fill that shrinks |running_qty| toward zero is "closing".
//   4. When running_qty crosses zero (full close), emit one closed position
//      with weighted-avg entry/exit prices, sum of realizedPnL, and the time
//      range from first opening fill to the closing fill.
//   5. A closing fill that overshoots zero (a direction flip in one-way mode,
//      e.g. long 5 then SELL 8) is split: |running_qty| closes the current
//      cycle, the remainder opens the next cycle in the opposite direction.
//      Quote value and commission are prorated by quantity; realizedPnL goes
//      to the closing side (Binance realizes PnL only on closes).
//   6. If, at the end of the walk, running_qty != 0, the residual is an OPEN
//      position fragment (we typically don't need this — positionRisk is the
//      authoritative source for OPEN — but we expose it for diagnostics).
//
// Pyramid / scale-in / partial-close behavior:
//   - Scale-in averages the entry price by quote-quantity weight.
//   - Partial closes accumulate realizedPnL and the exit price is the
//     quote-weighted average of the closing fills.
//   - A round trip is the full lifecycle from "first opening when qty was 0"
//     to "qty reaches 0 again".
package positions

import (
	"math"
	"sort"

	"github.com/xiagao/fund-dashboard/backend/binance"
)

// Lifecycle is a derived round-trip (or open fragment) for one symbol/side.
type Lifecycle struct {
	Symbol         string
	PositionSide   string  // BOTH | LONG | SHORT (Binance hedge mode)
	DirectionalSide string // LONG | SHORT (canonicalized — for BOTH we infer from first fill direction)
	EntryQuantity  float64 // total entry qty (sum of opening legs)
	ExitQuantity   float64 // total exit qty (sum of closing legs, 0 if still open)
	EntryPrice     float64 // quote-weighted average
	ExitPrice      float64 // quote-weighted average (0 if still open)
	EntryTime      int64
	ExitTime       int64 // 0 if still open
	RealizedPnL    float64
	Commission     float64
	Open           bool

	// Diagnostics — useful to render on hover, not strictly required.
	NumOpeningFills int
	NumClosingFills int
}

// Derive groups fills by (symbol, positionSide), walks each group, and emits a
// list of completed Lifecycles plus any still-open residuals.
//
// Returns (closed_round_trips, open_residuals). For BOTH-side mode trading,
// position direction is inferred from the sign of running qty at each opening
// fill.
func Derive(trades []binance.UserTrade) (closed, open []Lifecycle) {
	if len(trades) == 0 {
		return
	}
	grouped := map[string][]binance.UserTrade{}
	for _, t := range trades {
		key := t.Symbol + "|" + t.PositionSide
		grouped[key] = append(grouped[key], t)
	}

	for _, fills := range grouped {
		sort.Slice(fills, func(i, j int) bool { return fills[i].Time < fills[j].Time })

		var (
			running       float64 // signed net qty
			cur           Lifecycle
			entryQuoteSum float64 // for weighted-avg entry price
			exitQuoteSum  float64 // for weighted-avg exit price
		)

		// begin starts a fresh lifecycle keyed off fill f.
		begin := func(f binance.UserTrade, isLongLeg bool) {
			cur = Lifecycle{Symbol: f.Symbol, PositionSide: f.PositionSide, EntryTime: f.Time, Open: true}
			if f.PositionSide == "LONG" || f.PositionSide == "SHORT" {
				cur.DirectionalSide = f.PositionSide
			} else if isLongLeg {
				cur.DirectionalSide = "LONG"
			} else {
				cur.DirectionalSide = "SHORT"
			}
			entryQuoteSum, exitQuoteSum = 0, 0
		}
		// emit finalizes weighted prices and appends cur to closed/open.
		emit := func() {
			if cur.EntryQuantity <= 0 {
				cur = Lifecycle{}
				return
			}
			cur.EntryPrice = entryQuoteSum / cur.EntryQuantity
			if cur.ExitQuantity > 0 {
				cur.ExitPrice = exitQuoteSum / cur.ExitQuantity
			}
			if cur.Open {
				open = append(open, cur)
			} else {
				closed = append(closed, cur)
			}
			cur = Lifecycle{}
		}

		for _, f := range fills {
			isLongLeg := f.Side == "BUY" // BUY grows LONG / shrinks SHORT
			signed := f.Qty
			if !isLongLeg {
				signed = -f.Qty
			}

			switch {
			case running == 0 || sign(running) == sign(signed):
				// Opening / scale-in.
				if cur.EntryQuantity == 0 {
					begin(f, isLongLeg)
				}
				cur.EntryQuantity += f.Qty
				entryQuoteSum += f.QuoteQty
				cur.Commission += f.Commission
				cur.NumOpeningFills++
				running += signed

			case math.Abs(signed) <= math.Abs(running)+1e-9:
				// Closing (partial or full) — does not overshoot zero.
				cur.ExitQuantity += f.Qty
				exitQuoteSum += f.QuoteQty
				cur.ExitTime = f.Time
				cur.RealizedPnL += f.RealizedPnL
				cur.Commission += f.Commission
				cur.NumClosingFills++
				running += signed
				if math.Abs(running) < 1e-9 {
					running = 0
					cur.Open = false
					emit()
				}

			default:
				// Direction flip (one-way mode): a single fill that closes the
				// whole position AND opens a new one the other way. Split it —
				// the closing portion extinguishes the current cycle (its
				// realizedPnL belongs entirely to the close), the remainder
				// seeds the next cycle in the opposite direction. Quote value
				// and commission are prorated by quantity.
				closeQty := math.Abs(running)
				frac := closeQty / f.Qty
				cur.ExitQuantity += closeQty
				exitQuoteSum += f.QuoteQty * frac
				cur.ExitTime = f.Time
				cur.RealizedPnL += f.RealizedPnL
				cur.Commission += f.Commission * frac
				cur.NumClosingFills++
				cur.Open = false
				running += signed // overshoot remainder, signed toward the new direction
				emit()

				begin(f, isLongLeg)
				cur.EntryQuantity = f.Qty - closeQty
				entryQuoteSum = f.QuoteQty * (1 - frac)
				cur.Commission = f.Commission * (1 - frac)
				cur.NumOpeningFills = 1
			}
		}
		// Residual open at end of walk.
		emit()
	}

	// Sort closed by ExitTime desc (most recent first).
	sort.Slice(closed, func(i, j int) bool { return closed[i].ExitTime > closed[j].ExitTime })
	return
}

func sign(x float64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}
