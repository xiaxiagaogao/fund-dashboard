package binance

import (
	"context"
	"log"
	"net/url"
	"strconv"
	"time"
)

// WalkRealizedIncome pages through /fapi/v1/income REALIZED_PNL backwards from
// `untilMs` for the given lookback window. Binance limits each call to 1000
// rows; this helper paginates by walking backwards in 7-day windows.
//
// Returns the set of unique symbols that had REALIZED_PNL activity in the
// window. Used by the backfill subcommand to know which symbols to call
// userTrades on (so we don't ask Binance about every USDT-perp symbol).
func (c *Client) WalkRealizedIncomeSymbols(ctx context.Context, lookback time.Duration) (map[string]struct{}, error) {
	until := time.Now()
	earliest := until.Add(-lookback)
	step := 7 * 24 * time.Hour

	symbols := map[string]struct{}{}
	cur := until
	for cur.After(earliest) {
		from := cur.Add(-step)
		if from.Before(earliest) {
			from = earliest
		}
		params := url.Values{}
		params.Set("incomeType", "REALIZED_PNL")
		params.Set("startTime", strconv.FormatInt(from.UnixMilli(), 10))
		params.Set("endTime", strconv.FormatInt(cur.UnixMilli(), 10))
		params.Set("limit", "1000")
		var rows []IncomeRow
		if err := c.signedGET(ctx, "/fapi/v1/income", params, &rows); err != nil {
			return nil, err
		}
		for _, r := range rows {
			if r.Symbol != "" {
				symbols[r.Symbol] = struct{}{}
			}
		}
		cur = from
	}
	return symbols, nil
}

// WalkUserTradesSince paginates /fapi/v1/userTrades for one symbol from
// sinceMs to now and returns every fill in time-ascending order.
//
// Binance gotcha: when startTime is passed without endTime, the endpoint
// silently caps the window at startTime + 7d (so a 60-day startTime returns
// only the first week of activity). We work around this by walking forward
// in explicit 7-day windows. Within each window, if the first call hits the
// 1000-row page cap, we paginate with `fromId` to drain that window before
// advancing.
func (c *Client) WalkUserTradesSince(ctx context.Context, symbol string, sinceMs int64) ([]UserTrade, error) {
	const (
		pageSize  = 1000
		windowMs  = int64(7 * 24 * 60 * 60 * 1000)
		throttle  = 120 * time.Millisecond
	)
	now := time.Now().UnixMilli()
	if sinceMs <= 0 {
		sinceMs = now - 30*24*60*60*1000
	}

	var all []UserTrade
	winStart := sinceMs
	for winStart < now {
		winEnd := winStart + windowMs
		if winEnd > now {
			winEnd = now
		}

		// Within this window, paginate by fromId. First call uses
		// startTime+endTime; subsequent calls use fromId only and we filter
		// in code to stop once we cross winEnd.
		fromID := int64(0)
		for {
			params := url.Values{}
			params.Set("symbol", symbol)
			params.Set("limit", strconv.Itoa(pageSize))
			if fromID > 0 {
				params.Set("fromId", strconv.FormatInt(fromID, 10))
			} else {
				params.Set("startTime", strconv.FormatInt(winStart, 10))
				params.Set("endTime", strconv.FormatInt(winEnd, 10))
			}
			var rows []UserTradeRow
			if err := c.signedGET(ctx, "/fapi/v1/userTrades", params, &rows); err != nil {
				return all, err
			}
			if len(rows) == 0 {
				break
			}
			pastWindow := false
			for _, r := range rows {
				if fromID > 0 && r.Time > winEnd {
					pastWindow = true
					break
				}
				t, err := parseUserTrade(r)
				if err != nil {
					log.Printf("binance: skipping bad userTrades row: %v", err)
					continue
				}
				all = append(all, t)
			}
			if pastWindow || len(rows) < pageSize {
				break
			}
			fromID = rows[len(rows)-1].ID + 1
			select {
			case <-ctx.Done():
				return all, ctx.Err()
			case <-time.After(throttle):
			}
		}

		winStart = winEnd + 1
		// Polite throttle between windows too.
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		case <-time.After(throttle):
		}
	}
	return all, nil
}
