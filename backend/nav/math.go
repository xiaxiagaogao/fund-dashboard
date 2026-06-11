// Package nav implements the NAV-method share accounting math.
//
// The whole dashboard's correctness rides on these three functions. Touching them
// without re-running TestCanonicalScenario is a load-bearing mistake.
package nav

import "errors"

// ErrNoSharesOutstanding is returned when a withdrawal is attempted while the pool
// has no outstanding shares (i.e., NAV is undefined).
var ErrNoSharesOutstanding = errors.New("nav: cannot burn shares when total_shares is zero")

// CurrentNAV returns the per-share NAV given the pool's total equity and total
// outstanding shares. If totalShares is zero (or negative — treated identically),
// NAV is anchored at 1.0 so the very first deposit mints exactly amount_usdt shares.
func CurrentNAV(totalEquity, totalShares float64) float64 {
	if totalShares <= 0 {
		return 1.0
	}
	return totalEquity / totalShares
}

// ComputeMint returns the shares minted for a deposit and the NAV that was used.
// The NAV is determined by the pool state *before* the deposit hits.
//
// Pass the equity that EXCLUDES the just-arrived deposit. If you're reading
// live Binance equity at admin-record time (after the friend's transfer has
// already settled), use ComputeMintAfterArrival instead.
func ComputeMint(amountUSDT, totalEquityPre, totalShares float64) (sharesMinted, navAtEvent float64) {
	navAtEvent = CurrentNAV(totalEquityPre, totalShares)
	sharesMinted = amountUSDT / navAtEvent
	return
}

// ComputeMintAfterArrival is the realistic admin-records-after-transfer path:
// the live Binance equity ALREADY includes the just-arrived deposit, so we
// subtract it to get the pre-deposit equity before computing NAV.
//
// Skips the subtraction when total_shares == 0 (bootstrap): caller is
// responsible for ensuring amount == live_equity (or accepting the divergence).
func ComputeMintAfterArrival(amountUSDT, liveEquityPostArrival, totalShares float64) (sharesMinted, navAtEvent float64) {
	if totalShares <= 0 {
		return amountUSDT, 1.0
	}
	preEquity := liveEquityPostArrival - amountUSDT
	navAtEvent = preEquity / totalShares
	sharesMinted = amountUSDT / navAtEvent
	return
}

// ComputeBurn returns the shares burned for a withdrawal and the NAV used.
// Pass the equity that INCLUDES the soon-to-leave amount (i.e., pre-withdrawal).
// Returns ErrNoSharesOutstanding when called against an empty pool.
func ComputeBurn(amountUSDT, totalEquityPre, totalShares float64) (sharesBurned, navAtEvent float64, err error) {
	if totalShares <= 0 {
		return 0, 0, ErrNoSharesOutstanding
	}
	navAtEvent = totalEquityPre / totalShares
	sharesBurned = amountUSDT / navAtEvent
	return
}

// ComputeBurnAfterDeparture is the realistic admin-records-after-transfer path:
// the live Binance equity is ALREADY post-withdrawal (USDT has already left
// the account). Add the amount back to reconstruct pre-withdrawal equity.
func ComputeBurnAfterDeparture(amountUSDT, liveEquityPostDeparture, totalShares float64) (sharesBurned, navAtEvent float64, err error) {
	if totalShares <= 0 {
		return 0, 0, ErrNoSharesOutstanding
	}
	preEquity := liveEquityPostDeparture + amountUSDT
	navAtEvent = preEquity / totalShares
	sharesBurned = amountUSDT / navAtEvent
	return
}
