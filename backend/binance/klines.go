package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Kline is the single field the dashboard needs from a candle: its open time
// and close price. Used to benchmark the fund NAV against tokenized index
// perps (QQQUSDT ≈ Nasdaq-100, SPYUSDT ≈ S&P 500).
type Kline struct {
	OpenTime int64
	Close    float64
}

// DailyCloses fetches up to `limit` daily candles for symbol from the public
// /fapi/v1/klines endpoint (unsigned). Returns ascending by time. Binance caps
// limit at 1500.
func (c *Client) DailyCloses(ctx context.Context, symbol string, limit int) ([]Kline, error) {
	if limit <= 0 || limit > 1500 {
		limit = 500
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", "1d")
	params.Set("limit", strconv.Itoa(limit))

	// Each kline is a heterogeneous array: [openTime, open, high, low, close, ...].
	var raw [][]json.RawMessage
	if err := c.publicGET(ctx, "/fapi/v1/klines", params, &raw); err != nil {
		return nil, fmt.Errorf("klines %s: %w", symbol, err)
	}
	out := make([]Kline, 0, len(raw))
	for _, k := range raw {
		if len(k) < 5 {
			continue
		}
		var openTime int64
		if err := json.Unmarshal(k[0], &openTime); err != nil {
			continue
		}
		var closeStr string
		if err := json.Unmarshal(k[4], &closeStr); err != nil {
			continue
		}
		closePx, err := strconv.ParseFloat(closeStr, 64)
		if err != nil || closePx <= 0 {
			continue
		}
		out = append(out, Kline{OpenTime: openTime, Close: closePx})
	}
	return out, nil
}
