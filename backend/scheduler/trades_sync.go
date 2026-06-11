package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// TradesSyncJob keeps fund.db's binance_fills table in sync with Binance.
//
// Algorithm (idempotent):
//  1. Find the set of symbols to poll. Two sources OR'd together:
//     - currently OPEN positions (from /fapi/v3/positionRisk)
//     - every symbol that already has at least one fill in fund.db
//  2. For each symbol, query userTrades since LastFillTimeBySymbol(symbol)+1ms.
//  3. INSERT OR IGNORE each fill into binance_fills.
//
// Designed to run frequently (every snapshot tick = every hour). Each call is
// cheap when nothing new happened.
type TradesSyncJob struct {
	DB *sql.DB
	BN *binance.Client
}

// RunOnce performs one sync pass and returns how many new fills landed.
func (j *TradesSyncJob) RunOnce(ctx context.Context) (int, error) {
	if j.BN == nil {
		return 0, fmt.Errorf("trades sync: binance client not configured")
	}

	symbols, err := j.symbolsToPoll(ctx)
	if err != nil {
		return 0, fmt.Errorf("trades sync: discover symbols: %w", err)
	}

	totalNew := 0
	for sym := range symbols {
		last, _ := store.LastFillTimeBySymbol(ctx, j.DB, sym)
		// 1ms forward so the same fill isn't re-fetched on every poll.
		since := last + 1
		if since <= 1 {
			// First time seeing this symbol — fall back to last 7d to avoid
			// pulling all of history (which is what dashctl backfill-history
			// is for).
			since = time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
		}
		fills, err := j.BN.WalkUserTradesSince(ctx, sym, since)
		if err != nil {
			log.Printf("trades sync: %s WalkUserTradesSince err: %v", sym, err)
			continue
		}
		for _, t := range fills {
			_, inserted, err := store.InsertFillIgnore(ctx, j.DB, store.BinanceFill{
				BinanceTradeID: t.ID, BinanceOrderID: t.OrderID,
				Symbol: t.Symbol, Side: t.Side, PositionSide: t.PositionSide,
				Price: t.Price, Qty: t.Qty, QuoteQty: t.QuoteQty,
				RealizedPnL: t.RealizedPnL, Commission: t.Commission,
				Maker: t.Maker, Buyer: t.Buyer,
				FillTime: t.Time,
			})
			if err != nil {
				log.Printf("trades sync: insert err for %s trade %d: %v", sym, t.ID, err)
				continue
			}
			if inserted {
				totalNew++
			}
		}
	}
	if totalNew > 0 {
		store.WriteAudit(ctx, j.DB, "system", "trades_sync.run", map[string]any{
			"new_fills": totalNew, "symbols": len(symbols),
		})
	}
	return totalNew, nil
}

// Start launches a sync loop that runs at interval. First pass fires after the
// first interval (so dashboard startup isn't blocked on Binance latency).
func (j *TradesSyncJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n, err := j.RunOnce(ctx)
				if err != nil {
					log.Printf("trades sync: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("trades sync: +%d new fills", n)
				}
			}
		}
	}()
}

// symbolsToPoll unions three discovery sources so we never miss a symbol that
// was traded since the last sync:
//
//  1. **Live positionRisk** — every currently-open (and recently-open) position.
//     Catches symbols that are still on the book.
//
//  2. **fund.db symbols with fills in the last 90d** — cheap, no API call.
//     Catches symbols whose positions closed recently but might have NEW
//     fills since LastFillTimeBySymbol. Bounded by recency so the poll set
//     (one userTrades call per symbol per tick) doesn't grow forever; a
//     symbol untraded for 90d that comes back gets re-discovered via
//     positionRisk or income.
//
//  3. **/fapi/v1/income REALIZED_PNL last 2h** — catches the edge case where a
//     brand-new symbol was opened AND closed within one sync tick, then
//     vacated positionRisk. income still records the realized event so we can
//     backfill userTrades for it on the next tick.
//
// Together these cover any conceivable trade pattern (intraday flips,
// brand-new Binance listings, multi-day holds) without needing a hard-coded
// symbol allow-list.
func (j *TradesSyncJob) symbolsToPoll(ctx context.Context) (map[string]struct{}, error) {
	syms := map[string]struct{}{}

	risks, err := j.BN.PositionRisks(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range risks {
		syms[r.Symbol] = struct{}{}
	}

	seen, err := store.DistinctSymbolsSince(ctx, j.DB, time.Now().Add(-90*24*time.Hour).UnixMilli())
	if err != nil {
		return nil, err
	}
	for _, s := range seen {
		syms[s] = struct{}{}
	}

	// Third source: REALIZED_PNL income events from the last 2 hours.
	// Window = 2x scheduler tick to be tolerant of small clock drift / a
	// missed tick. Non-fatal — if Binance throws on income, we degrade
	// gracefully to the first two sources.
	if incomeSyms, err := j.BN.WalkRealizedIncomeSymbols(ctx, 2*time.Hour); err == nil {
		for s := range incomeSyms {
			syms[s] = struct{}{}
		}
	} else {
		log.Printf("trades sync: income symbol discovery degraded: %v", err)
	}

	return syms, nil
}
