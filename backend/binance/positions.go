package binance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// PositionRiskRow mirrors one entry from /fapi/v3/positionRisk.
//
// Numeric fields come back as JSON strings on the wire — we keep the raw strings
// here and parse via PositionRisks() below.
type PositionRiskRow struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	MarkPrice        string `json:"markPrice"`
	UnRealizedProfit string `json:"unRealizedProfit"`
	LiquidationPrice string `json:"liquidationPrice"`
	Leverage         string `json:"leverage"`
	MarginType       string `json:"marginType"`
	PositionSide     string `json:"positionSide"`
	BreakEvenPrice   string `json:"breakEvenPrice"`
	UpdateTime       int64  `json:"updateTime"`
}

// PositionRisk is the parsed shape callers actually want.
type PositionRisk struct {
	Symbol           string
	PositionAmt      float64 // signed (LONG positive, SHORT negative)
	EntryPrice       float64
	MarkPrice        float64
	UnRealizedProfit float64
	LiquidationPrice float64
	Leverage         int
	MarginType       string
	PositionSide     string // BOTH | LONG | SHORT
	UpdateTime       int64
}

// PositionRisks returns the open positions on the account, parsed from
// /fapi/v3/positionRisk. Zero-quantity rows are dropped so callers only see
// truly open positions.
//
// Used to power the "current positions" panel — independent of nofx.
func (c *Client) PositionRisks(ctx context.Context) ([]PositionRisk, error) {
	var rows []PositionRiskRow
	if err := c.signedGET(ctx, "/fapi/v3/positionRisk", nil, &rows); err != nil {
		return nil, err
	}
	out := make([]PositionRisk, 0, len(rows))
	for _, r := range rows {
		amt, _ := strconv.ParseFloat(r.PositionAmt, 64)
		if amt == 0 {
			continue
		}
		entry, _ := strconv.ParseFloat(r.EntryPrice, 64)
		mark, _ := strconv.ParseFloat(r.MarkPrice, 64)
		unP, _ := strconv.ParseFloat(r.UnRealizedProfit, 64)
		liq, _ := strconv.ParseFloat(r.LiquidationPrice, 64)
		lev, _ := strconv.Atoi(r.Leverage)
		out = append(out, PositionRisk{
			Symbol: r.Symbol, PositionAmt: amt, EntryPrice: entry,
			MarkPrice: mark, UnRealizedProfit: unP, LiquidationPrice: liq,
			Leverage: lev, MarginType: r.MarginType, PositionSide: r.PositionSide,
			UpdateTime: r.UpdateTime,
		})
	}
	return out, nil
}

// UserTradeRow is one fill from /fapi/v1/userTrades. Wire shape (numbers as strings).
type UserTradeRow struct {
	ID              int64  `json:"id"`
	OrderID         int64  `json:"orderId"`
	Symbol          string `json:"symbol"`
	Side            string `json:"side"`         // BUY | SELL
	PositionSide    string `json:"positionSide"` // BOTH | LONG | SHORT
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	QuoteQty        string `json:"quoteQty"`
	RealizedPnl     string `json:"realizedPnl"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	Maker           bool   `json:"maker"`
	Buyer           bool   `json:"buyer"`
}

// UserTrade is the parsed shape.
type UserTrade struct {
	ID           int64
	OrderID      int64
	Symbol       string
	Side         string
	PositionSide string
	Price        float64
	Qty          float64
	QuoteQty     float64
	RealizedPnL  float64
	Commission   float64
	Time         int64
	Maker        bool
	Buyer        bool
}

// UserTradesSince returns up to `limit` fills for `symbol` after `sinceMs`,
// page by page. Binance returns max 1000 per call.
//
// If sinceMs is 0 it returns the most recent N (where N <= limit).
// To pull all symbols' trades, call once per symbol — Binance doesn't have a
// cross-symbol endpoint.
func (c *Client) UserTradesSince(ctx context.Context, symbol string, sinceMs int64, limit int) ([]UserTrade, error) {
	if limit <= 0 {
		limit = 1000
	}
	if limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("limit", strconv.Itoa(limit))
	if sinceMs > 0 {
		params.Set("startTime", strconv.FormatInt(sinceMs, 10))
	}
	var rows []UserTradeRow
	if err := c.signedGET(ctx, "/fapi/v1/userTrades", params, &rows); err != nil {
		return nil, fmt.Errorf("userTrades %s: %w", symbol, err)
	}
	out := make([]UserTrade, 0, len(rows))
	for _, r := range rows {
		price, _ := strconv.ParseFloat(r.Price, 64)
		qty, _ := strconv.ParseFloat(r.Qty, 64)
		quote, _ := strconv.ParseFloat(r.QuoteQty, 64)
		realized, _ := strconv.ParseFloat(r.RealizedPnl, 64)
		commission, _ := strconv.ParseFloat(r.Commission, 64)
		out = append(out, UserTrade{
			ID: r.ID, OrderID: r.OrderID, Symbol: r.Symbol, Side: r.Side,
			PositionSide: r.PositionSide, Price: price, Qty: qty, QuoteQty: quote,
			RealizedPnL: realized, Commission: commission, Time: r.Time,
			Maker: r.Maker, Buyer: r.Buyer,
		})
	}
	return out, nil
}

// IncomeRow is one row of /fapi/v1/income. The dashboard only filters for
// REALIZED_PNL events — they're the cleanest "a position closed" signal.
type IncomeRow struct {
	Symbol     string `json:"symbol"`
	IncomeType string `json:"incomeType"`
	Income     string `json:"income"`
	Asset      string `json:"asset"`
	Info       string `json:"info"`
	Time       int64  `json:"time"`
	TranID     int64  `json:"tranId"`
	TradeID    string `json:"tradeId"`
}

// RealizedIncomeSince queries /fapi/v1/income filtered to REALIZED_PNL events
// since sinceMs. Used by the closed-trades reconstruction as a cross-check for
// which symbols had activity in the window (avoids polling every USDT symbol).
func (c *Client) RealizedIncomeSince(ctx context.Context, sinceMs int64, limit int) ([]IncomeRow, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	params := url.Values{}
	params.Set("incomeType", "REALIZED_PNL")
	params.Set("limit", strconv.Itoa(limit))
	if sinceMs > 0 {
		params.Set("startTime", strconv.FormatInt(sinceMs, 10))
	}
	var rows []IncomeRow
	if err := c.signedGET(ctx, "/fapi/v1/income", params, &rows); err != nil {
		return nil, fmt.Errorf("income: %w", err)
	}
	return rows, nil
}
