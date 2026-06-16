package positions

import (
	"testing"
	"time"

	"github.com/xiagao/fund-dashboard/backend/store"
)

func TestDailyPnL(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	// 2026-06-15 02:00 UTC = 10:00 China same day.
	// 2026-06-15 18:00 UTC = 2026-06-16 02:00 China → next day.
	d1 := time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC).UnixMilli()
	d1b := time.Date(2026, 6, 15, 5, 0, 0, 0, time.UTC).UnixMilli()
	d2 := time.Date(2026, 6, 15, 18, 0, 0, 0, time.UTC).UnixMilli()
	fills := []store.BinanceFill{
		{FillTime: d1, RealizedPnL: 10, Commission: 1},
		{FillTime: d1b, RealizedPnL: -4, Commission: 0.5},
		{FillTime: d2, RealizedPnL: 7, Commission: 0.2},
	}
	out := DailyPnL(fills, loc)
	if len(out) != 2 {
		t.Fatalf("expected 2 days, got %d: %+v", len(out), out)
	}
	// Sorted asc; first day is 06-15 with 2 fills.
	if out[0].Date != "2026-06-15" || out[0].Fills != 2 {
		t.Errorf("day0: %+v", out[0])
	}
	if math := out[0].RealizedPnL; math != 6 {
		t.Errorf("day0 realized: got %v want 6", math)
	}
	if out[0].Net != 6-1.5 {
		t.Errorf("day0 net: got %v want 4.5", out[0].Net)
	}
	// Second fill crosses to 06-16 in China time.
	if out[1].Date != "2026-06-16" || out[1].Fills != 1 {
		t.Errorf("day1: %+v", out[1])
	}
	if out[1].Net != 7-0.2 {
		t.Errorf("day1 net: got %v want 6.8", out[1].Net)
	}
}
