package api

import (
	"net/http"
	"time"

	"github.com/xiagao/fund-dashboard/backend/store"
)

// indexSymbols maps the friendly key returned to the frontend to the Binance
// tokenized-perp symbol stored in index_prices.
var indexSymbols = []struct{ Key, Symbol string }{
	{"qqq", "QQQUSDT"},
	{"spy", "SPYUSDT"},
}

// GET /api/index-prices?from=ms&to=ms
//
// Daily closes of the benchmark index perps in the window, for the friend
// dashboard's "vs 大盘" comparison. Returns a few days before `from` too so the
// client has a baseline close at or before the curve's start. All friends may
// see it.
func (s *Server) handleIndexPrices(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UnixMilli()
	from := parseInt64Default(r.URL.Query().Get("from"), now-30*24*60*60*1000)
	to := parseInt64Default(r.URL.Query().Get("to"), now)
	// Pad the lower bound so the client can pick a baseline close <= from.
	lo := from - 5*24*60*60*1000

	out := make(map[string][]map[string]any, len(indexSymbols))
	for _, sym := range indexSymbols {
		closes, err := store.ListIndexCloses(r.Context(), s.DB, sym.Symbol, lo, to)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "index query failed: "+err.Error())
			return
		}
		arr := make([]map[string]any, 0, len(closes))
		for _, c := range closes {
			arr = append(arr, map[string]any{"t": c.DayMs, "close": c.Close})
		}
		out[sym.Key] = arr
	}
	writeJSON(w, http.StatusOK, out)
}
