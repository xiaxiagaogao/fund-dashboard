package positions

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/store"
	_ "modernc.org/sqlite"
)

// Opt-in forensic check against a real fund.db snapshot. It asserts the
// invariant that was violated in production: every USDT Binance ever realized
// must appear somewhere in the dashboard's output — either on a completed round
// trip or on the open position that banked it via partial closes.
//
//	FUND_DB=/path/to/fund.db go test ./backend/positions/ -run TestReconcile -count=1 -v
func TestReconcile_RealSnapshotMatchesBinance(t *testing.T) {
	path := os.Getenv("FUND_DB")
	if path == "" {
		t.Skip("set FUND_DB to a fund.db snapshot to run the reconciliation")
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	fills, err := store.ListFillsSince(context.Background(), db, 0)
	if err != nil {
		t.Fatal(err)
	}
	var truthPnL, truthFees float64
	for _, f := range fills {
		truthPnL += f.RealizedPnL
		truthFees += f.Commission
	}

	closedCycles, openResiduals := Derive(fillsToUserTrades(fills))
	closed := buildClosedViews(closedCycles)

	// Open positions as the API now reports them: realized PnL carried over from
	// the reconstructed residual (positionRisk only supplies mark/unrealized).
	var openRealized, openFees float64
	for _, l := range openResiduals {
		openRealized += l.RealizedPnL
		openFees += l.Commission
	}
	var closedRealized, closedFees float64
	for _, v := range closed {
		closedRealized += v.RealizedPnL
		closedFees += v.Commission
	}

	fmt.Printf("\n===== RECONCILIATION (%d fills) =====\n", len(fills))
	fmt.Printf("Binance realized (truth) : %+.4f   fees %.4f   NET %+.4f\n",
		truthPnL, truthFees, truthPnL-truthFees)
	fmt.Printf("closed round trips (%3d) : %+.4f   fees %.4f\n", len(closed), closedRealized, closedFees)
	fmt.Printf("open positions     (%3d) : %+.4f   fees %.4f  <- banked by partial closes\n",
		len(openResiduals), openRealized, openFees)
	fmt.Printf("dashboard total          : %+.4f   fees %.4f   NET %+.4f\n",
		closedRealized+openRealized, closedFees+openFees,
		closedRealized+openRealized-(closedFees+openFees))
	st := ComputeStats(closed)
	fmt.Printf("--- panels after the fix ---\n")
	fmt.Printf("平仓记录 : %d 笔 · 净 %+.2f · 费 %.2f\n", len(closed), closedRealized-closedFees, closedFees)
	fmt.Printf("胜率     : %.1f%% (%d 胜 / %d 负, 净口径)\n", st.WinRate*100, st.Wins, st.Losses)
	fmt.Printf("持仓已实现: %+.2f (净)\n", openRealized-openFees)
	fmt.Printf("=====================================\n")

	if d := math.Abs(closedRealized + openRealized - truthPnL); d > 1e-6 {
		t.Errorf("realized PnL does not reconcile: dashboard %.6f vs Binance %.6f (off by %.6f)",
			closedRealized+openRealized, truthPnL, d)
	}
	if d := math.Abs(closedFees + openFees - truthFees); d > 1e-6 {
		t.Errorf("fees do not reconcile: dashboard %.6f vs Binance %.6f (off by %.6f)",
			closedFees+openFees, truthFees, d)
	}
}
