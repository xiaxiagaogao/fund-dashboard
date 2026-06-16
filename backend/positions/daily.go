package positions

import (
	"sort"
	"time"

	"github.com/xiagao/fund-dashboard/backend/store"
)

// DayPnL is one calendar day's realized trading result, for the 复盘 heatmap.
type DayPnL struct {
	Date        string  `json:"date"` // YYYY-MM-DD in the report location
	RealizedPnL float64 `json:"realized_pnl"`
	Commission  float64 `json:"commission"`
	Net         float64 `json:"net"` // realized - commission
	Fills       int     `json:"fills"`
}

// DailyPnL buckets fills into per-day net realized PnL using loc to decide
// which calendar day each fill belongs to (the operator is in China, so days
// are bucketed in their wall-clock time, not UTC). Days with no fills are
// omitted — the heatmap fills gaps itself.
func DailyPnL(fills []store.BinanceFill, loc *time.Location) []DayPnL {
	by := map[string]*DayPnL{}
	for _, f := range fills {
		day := time.UnixMilli(f.FillTime).In(loc).Format("2006-01-02")
		d, ok := by[day]
		if !ok {
			d = &DayPnL{Date: day}
			by[day] = d
		}
		d.RealizedPnL += f.RealizedPnL
		d.Commission += f.Commission
		d.Fills++
	}
	out := make([]DayPnL, 0, len(by))
	for _, d := range by {
		d.Net = d.RealizedPnL - d.Commission
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}
