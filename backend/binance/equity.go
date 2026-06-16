package binance

import (
	"context"
	"fmt"
	"strconv"
)

// AccountResponse is the subset of /fapi/v2/account fields the dashboard cares
// about. Binance returns numeric values as JSON strings.
type AccountResponse struct {
	TotalWalletBalance         string `json:"totalWalletBalance"`
	TotalUnrealizedProfit      string `json:"totalUnrealizedProfit"`
	TotalMarginBalance         string `json:"totalMarginBalance"`
	TotalPositionInitialMargin string `json:"totalPositionInitialMargin"`
	AvailableBalance           string `json:"availableBalance"`
	UpdateTime                 int64  `json:"updateTime"`
}

// AccountSummary is the parsed capital snapshot powering the friend-facing
// "资金配置" donut: how much of the pool is posted as position margin vs idle.
type AccountSummary struct {
	Equity        float64 // totalWalletBalance + totalUnrealizedProfit
	MarginUsed    float64 // totalPositionInitialMargin
	UpdateTime    int64
}

// AccountSummaryNow fetches /fapi/v2/account and parses the capital fields.
func (c *Client) AccountSummaryNow(ctx context.Context) (AccountSummary, error) {
	var resp AccountResponse
	if err := c.signedGET(ctx, "/fapi/v2/account", nil, &resp); err != nil {
		return AccountSummary{}, err
	}
	wallet, err := strconv.ParseFloat(resp.TotalWalletBalance, 64)
	if err != nil {
		return AccountSummary{}, fmt.Errorf("parse totalWalletBalance %q: %w", resp.TotalWalletBalance, err)
	}
	unrealized, err := strconv.ParseFloat(resp.TotalUnrealizedProfit, 64)
	if err != nil {
		return AccountSummary{}, fmt.Errorf("parse totalUnrealizedProfit %q: %w", resp.TotalUnrealizedProfit, err)
	}
	// Position initial margin can be absent/empty when flat — treat as 0.
	margin, _ := strconv.ParseFloat(resp.TotalPositionInitialMargin, 64)
	return AccountSummary{Equity: wallet + unrealized, MarginUsed: margin, UpdateTime: resp.UpdateTime}, nil
}

// AccountEquity returns the pool's total equity in USDT — defined as
//
//	totalWalletBalance + totalUnrealizedProfit
//
// (which is what Binance calls totalMarginBalance when no isolated-margin
// positions exist). This is the numerator for NAV.
func (c *Client) AccountEquity(ctx context.Context) (float64, *AccountResponse, error) {
	var resp AccountResponse
	if err := c.signedGET(ctx, "/fapi/v2/account", nil, &resp); err != nil {
		return 0, nil, err
	}
	wallet, err := strconv.ParseFloat(resp.TotalWalletBalance, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("parse totalWalletBalance %q: %w", resp.TotalWalletBalance, err)
	}
	unrealized, err := strconv.ParseFloat(resp.TotalUnrealizedProfit, 64)
	if err != nil {
		return 0, nil, fmt.Errorf("parse totalUnrealizedProfit %q: %w", resp.TotalUnrealizedProfit, err)
	}
	return wallet + unrealized, &resp, nil
}
