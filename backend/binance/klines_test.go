package binance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Binance returns klines as heterogeneous arrays (numbers + quoted strings).
// Verify DailyCloses pulls openTime (index 0) and close (index 4) correctly and
// skips malformed rows.
func TestDailyCloses_Parsing(t *testing.T) {
	body := `[
		[1717200000000,"730.00","740.00","725.00","736.52","123.4",1717286399999,"0",10,"0","0","0"],
		[1717286400000,"736.50","720.00","700.00","716.68","234.5",1717372799999,"0",11,"0","0","0"],
		[1717372800000,"716.00","-1","0","0","0",1,"0",0,"0","0","0"]
	]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fapi/v1/klines" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("interval") != "1d" || r.URL.Query().Get("symbol") != "QQQUSDT" {
			t.Errorf("unexpected query %s", r.URL.RawQuery)
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c := New("", "")
	c.SetBaseURL(srv.URL)
	out, err := c.DailyCloses(context.Background(), "QQQUSDT", 500)
	if err != nil {
		t.Fatalf("DailyCloses: %v", err)
	}
	// Third row has close "0" → dropped.
	if len(out) != 2 {
		t.Fatalf("got %d closes, want 2 (zero-close dropped): %+v", len(out), out)
	}
	if out[0].OpenTime != 1717200000000 || out[0].Close != 736.52 {
		t.Errorf("row0: %+v", out[0])
	}
	if out[1].OpenTime != 1717286400000 || out[1].Close != 716.68 {
		t.Errorf("row1: %+v", out[1])
	}
}
