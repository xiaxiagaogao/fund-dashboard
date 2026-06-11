package positions

import (
	"context"
	"database/sql"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// View is what API handlers return for a single position.
//
// Source of truth is the dashboard's own fund.db (binance_fills table backfilled
// from /fapi/v1/userTrades + kept current by scheduler.TradesSyncJob). Open
// positions also augment with live /fapi/v3/positionRisk for mark price and
// unrealized PnL. The dashboard does NOT read from nofx for any of this.
type View struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"` // LONG | SHORT
	Quantity         float64 `json:"quantity"`
	EntryPrice       float64 `json:"entry_price"`
	EntryTime        int64   `json:"entry_time"`
	ExitPrice        float64 `json:"exit_price,omitempty"`
	ExitTime         int64   `json:"exit_time,omitempty"`
	RealizedPnL      float64 `json:"realized_pnl"`
	UnrealizedPnL    float64 `json:"unrealized_pnl,omitempty"` // only meaningful for OPEN
	MarkPrice        float64 `json:"mark_price,omitempty"`     // only meaningful for OPEN
	LiquidationPrice float64 `json:"liquidation_price,omitempty"`
	Leverage         int     `json:"leverage,omitempty"`
	Status           string  `json:"status"` // OPEN | CLOSED
}

// Orchestrator builds OPEN + CLOSED position views.
//
//   - CLOSED come from store.ListFillsSince → Derive (FIFO reconstruction).
//   - OPEN come live from Binance /fapi/v3/positionRisk (gives mark / unrealized
//     PnL that fund.db can't store accurately). Entry time is recovered from
//     fund.db by looking up the most recent opening fill.
type Orchestrator struct {
	BN       *binance.Client
	FundDB   *sql.DB
	Lookback time.Duration // how far back to look for closed positions; default 90d
	CacheTTL time.Duration // default 60s

	mu     sync.Mutex // guards cached
	cached *cacheEntry

	fetchMu sync.Mutex // single-flight: at most one Binance refresh in flight
}

type cacheEntry struct {
	at     time.Time
	open   []View
	closed []View
}

// maxStaleServe caps how old a cache entry may be and still be served as a
// fallback when a refresh fails — a transient Binance hiccup shouldn't blank
// the positions panels for everyone.
const maxStaleServe = 15 * time.Minute

// Refresh returns (open, closed) views with a short cache so multiple friends
// loading the page don't trigger a Binance positionRisk fetch each time.
func (o *Orchestrator) Refresh(ctx context.Context) (open []View, closed []View, err error) {
	if c := o.freshCache(o.cacheTTL()); c != nil {
		return c.open, c.closed, nil
	}

	// Single-flight: one goroutine talks to Binance, late arrivals queue here
	// and then find the freshly written cache instead of fetching again.
	o.fetchMu.Lock()
	defer o.fetchMu.Unlock()
	if c := o.freshCache(o.cacheTTL()); c != nil {
		return c.open, c.closed, nil
	}

	open, closed, err = o.refreshNow(ctx)
	if err != nil {
		if c := o.freshCache(maxStaleServe); c != nil {
			log.Printf("positions: refresh failed, serving %s-old cache: %v",
				time.Since(c.at).Round(time.Second), err)
			return c.open, c.closed, nil
		}
		return nil, nil, err
	}
	o.mu.Lock()
	o.cached = &cacheEntry{at: time.Now(), open: open, closed: closed}
	o.mu.Unlock()
	return open, closed, nil
}

// freshCache returns the cache entry if it is younger than maxAge, else nil.
func (o *Orchestrator) freshCache(maxAge time.Duration) *cacheEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cached != nil && time.Since(o.cached.at) < maxAge {
		return o.cached
	}
	return nil
}

func (o *Orchestrator) cacheTTL() time.Duration {
	if o.CacheTTL > 0 {
		return o.CacheTTL
	}
	return 60 * time.Second
}

func (o *Orchestrator) lookback() time.Duration {
	if o.Lookback > 0 {
		return o.Lookback
	}
	return 90 * 24 * time.Hour
}

func (o *Orchestrator) refreshNow(ctx context.Context) ([]View, []View, error) {
	// 1) OPEN — live from Binance positionRisk (mark + unrealized must be current).
	risks, err := o.BN.PositionRisks(ctx)
	if err != nil {
		return nil, nil, err
	}

	// 2) Read fills from fund.db. This is the dashboard's own canonical record.
	sinceMs := time.Now().Add(-o.lookback()).UnixMilli()
	fills, err := store.ListFillsSince(ctx, o.FundDB, sinceMs)
	if err != nil {
		return nil, nil, err
	}
	trades := fillsToUserTrades(fills)
	closedCycles, openResidualsFromFills := Derive(trades)

	// 3) Build OPEN views from positionRisk; recover entry_time from fills.
	openByKey := map[string]Lifecycle{}
	for _, l := range openResidualsFromFills {
		openByKey[l.Symbol+"|"+l.DirectionalSide] = l
	}
	openViews := make([]View, 0, len(risks))
	for _, r := range risks {
		side := "LONG"
		if r.PositionAmt < 0 {
			side = "SHORT"
		}
		if r.PositionSide == "SHORT" {
			side = "SHORT"
		} else if r.PositionSide == "LONG" {
			side = "LONG"
		}
		v := View{
			Symbol:           r.Symbol,
			Side:             side,
			Quantity:         absF(r.PositionAmt),
			EntryPrice:       r.EntryPrice,
			MarkPrice:        r.MarkPrice,
			UnrealizedPnL:    r.UnRealizedProfit,
			LiquidationPrice: r.LiquidationPrice,
			Leverage:         r.Leverage,
			Status:           "OPEN",
		}
		if matched, ok := openByKey[r.Symbol+"|"+side]; ok {
			v.EntryTime = matched.EntryTime
		} else {
			v.EntryTime = r.UpdateTime
		}
		openViews = append(openViews, v)
	}

	// 4) Build CLOSED views from reconstructed lifecycles.
	closedViews := make([]View, 0, len(closedCycles))
	for _, l := range closedCycles {
		closedViews = append(closedViews, View{
			Symbol:      l.Symbol,
			Side:        l.DirectionalSide,
			Quantity:    l.EntryQuantity,
			EntryPrice:  l.EntryPrice,
			EntryTime:   l.EntryTime,
			ExitPrice:   l.ExitPrice,
			ExitTime:    l.ExitTime,
			RealizedPnL: l.RealizedPnL,
			Status:      "CLOSED",
		})
	}

	sort.Slice(closedViews, func(i, j int) bool { return closedViews[i].ExitTime > closedViews[j].ExitTime })
	sort.Slice(openViews, func(i, j int) bool { return openViews[i].EntryTime > openViews[j].EntryTime })
	return openViews, closedViews, nil
}

// fillsToUserTrades adapts the persisted-fill rows back into the in-memory
// shape Derive expects.
func fillsToUserTrades(fills []store.BinanceFill) []binance.UserTrade {
	out := make([]binance.UserTrade, 0, len(fills))
	for _, f := range fills {
		out = append(out, binance.UserTrade{
			ID:           f.BinanceTradeID,
			OrderID:      f.BinanceOrderID,
			Symbol:       f.Symbol,
			Side:         f.Side,
			PositionSide: f.PositionSide,
			Price:        f.Price,
			Qty:          f.Qty,
			QuoteQty:     f.QuoteQty,
			RealizedPnL:  f.RealizedPnL,
			Commission:   f.Commission,
			Time:         f.FillTime,
			Maker:        f.Maker,
			Buyer:        f.Buyer,
		})
	}
	return out
}

// --- Stats / SymbolPnL (unchanged from prior implementation) ---

type Stats struct {
	Total           int     `json:"total"`
	Wins            int     `json:"wins"`
	Losses          int     `json:"losses"`
	WinRate         float64 `json:"win_rate"`
	TotalPnL        float64 `json:"total_pnl"`
	AvgWinUSDT      float64 `json:"avg_win_usdt"`
	AvgLossUSDT     float64 `json:"avg_loss_usdt"`
	WinLossRatio    float64 `json:"win_loss_ratio"`
	AvgHoldHours    float64 `json:"avg_hold_hours"`
	MedianHoldHours float64 `json:"median_hold_hours"`
}

func ComputeStats(closed []View) Stats {
	s := Stats{Total: len(closed)}
	if s.Total == 0 {
		return s
	}
	var winSum, lossSum float64
	var holds []float64
	for _, p := range closed {
		s.TotalPnL += p.RealizedPnL
		if p.RealizedPnL > 0 {
			s.Wins++
			winSum += p.RealizedPnL
		} else if p.RealizedPnL < 0 {
			s.Losses++
			lossSum += p.RealizedPnL
		}
		if p.ExitTime > p.EntryTime {
			holds = append(holds, float64(p.ExitTime-p.EntryTime)/3_600_000)
		}
	}
	s.WinRate = float64(s.Wins) / float64(s.Total)
	if s.Wins > 0 {
		s.AvgWinUSDT = winSum / float64(s.Wins)
	}
	if s.Losses > 0 {
		s.AvgLossUSDT = lossSum / float64(s.Losses)
		if s.AvgLossUSDT < 0 {
			s.WinLossRatio = s.AvgWinUSDT / (-s.AvgLossUSDT)
		}
	}
	if len(holds) > 0 {
		var sum float64
		for _, h := range holds {
			sum += h
		}
		s.AvgHoldHours = sum / float64(len(holds))
		sort.Float64s(holds)
		mid := len(holds) / 2
		if len(holds)%2 == 0 {
			s.MedianHoldHours = (holds[mid-1] + holds[mid]) / 2
		} else {
			s.MedianHoldHours = holds[mid]
		}
	}
	return s
}

type SymbolPnL struct {
	Symbol   string  `json:"symbol"`
	Trades   int     `json:"trades"`
	Wins     int     `json:"wins"`
	TotalPnL float64 `json:"total_pnl"`
	WinRate  float64 `json:"win_rate"`
}

func AggregateBySymbol(closed []View) []SymbolPnL {
	by := map[string]*SymbolPnL{}
	for _, p := range closed {
		s, ok := by[p.Symbol]
		if !ok {
			s = &SymbolPnL{Symbol: p.Symbol}
			by[p.Symbol] = s
		}
		s.Trades++
		s.TotalPnL += p.RealizedPnL
		if p.RealizedPnL > 0 {
			s.Wins++
		}
	}
	out := make([]SymbolPnL, 0, len(by))
	for _, s := range by {
		if s.Trades > 0 {
			s.WinRate = float64(s.Wins) / float64(s.Trades)
		}
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return absF(out[i].TotalPnL) > absF(out[j].TotalPnL) })
	return out
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
