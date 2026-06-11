package store

import (
	"context"
	"testing"
)

func TestInsertFillIgnore_Dedup(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/fund.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	f := BinanceFill{
		BinanceTradeID: 12345, BinanceOrderID: 999, Symbol: "TSLAUSDT",
		Side: "BUY", PositionSide: "LONG",
		Price: 100, Qty: 1, QuoteQty: 100,
		FillTime: 1_700_000_000_000,
	}
	_, inserted, err := InsertFillIgnore(ctx, db, f)
	if err != nil || !inserted {
		t.Fatalf("first insert: err=%v inserted=%v", err, inserted)
	}
	// Duplicate trade_id should be silently ignored.
	_, inserted2, err := InsertFillIgnore(ctx, db, f)
	if err != nil {
		t.Fatal(err)
	}
	if inserted2 {
		t.Error("second insert with same trade_id should NOT mark inserted=true")
	}
}

func TestListFillsSince_TimeFilter(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/fund.db"
	db, _ := Open(path)
	defer db.Close()

	fills := []BinanceFill{
		{BinanceTradeID: 1, Symbol: "A", Side: "BUY", PositionSide: "LONG", Price: 1, Qty: 1, QuoteQty: 1, FillTime: 1000},
		{BinanceTradeID: 2, Symbol: "B", Side: "BUY", PositionSide: "LONG", Price: 1, Qty: 1, QuoteQty: 1, FillTime: 2000},
		{BinanceTradeID: 3, Symbol: "C", Side: "BUY", PositionSide: "LONG", Price: 1, Qty: 1, QuoteQty: 1, FillTime: 3000},
	}
	for _, f := range fills {
		if _, _, err := InsertFillIgnore(ctx, db, f); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ListFillsSince(ctx, db, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 fills since 2000, got %d", len(got))
	}
	if got[0].FillTime != 2000 || got[1].FillTime != 3000 {
		t.Errorf("order wrong: %+v", got)
	}
}

func TestLastFillTimeBySymbol(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/fund.db"
	db, _ := Open(path)
	defer db.Close()

	for i, ts := range []int64{1000, 2000, 3000} {
		if _, _, err := InsertFillIgnore(ctx, db, BinanceFill{
			BinanceTradeID: int64(100 + i), Symbol: "X", Side: "BUY", PositionSide: "LONG",
			Price: 1, Qty: 1, QuoteQty: 1, FillTime: ts,
		}); err != nil {
			t.Fatal(err)
		}
	}
	ts, err := LastFillTimeBySymbol(ctx, db, "X")
	if err != nil {
		t.Fatal(err)
	}
	if ts != 3000 {
		t.Errorf("want 3000, got %d", ts)
	}
	// Unknown symbol → 0
	ts2, _ := LastFillTimeBySymbol(ctx, db, "Y")
	if ts2 != 0 {
		t.Errorf("unknown symbol should return 0, got %d", ts2)
	}
}
