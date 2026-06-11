package binance

import (
	"context"
	"fmt"
	"strconv"
)

// AccountResponse is the subset of /fapi/v2/account fields the dashboard cares
// about. Binance returns numeric values as JSON strings.
type AccountResponse struct {
	TotalWalletBalance     string `json:"totalWalletBalance"`
	TotalUnrealizedProfit  string `json:"totalUnrealizedProfit"`
	TotalMarginBalance     string `json:"totalMarginBalance"`
	AvailableBalance       string `json:"availableBalance"`
	UpdateTime             int64  `json:"updateTime"`
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
