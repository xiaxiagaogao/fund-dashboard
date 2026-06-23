package scheduler

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// IndexSyncJob keeps fund.db's index_prices current with the tokenized index
// perps used to benchmark the fund. Uses the public klines endpoint, so it
// works even when no trading API keys are configured.
type IndexSyncJob struct {
	DB      *sql.DB
	BN      *binance.Client
	Symbols []string // e.g. ["QQQUSDT", "SPYUSDT"]
}

// RunOnce fetches recent daily closes for each symbol and upserts them.
// On an empty table this backfills ~500 days; on later runs it just refreshes
// the tail (idempotent).
func (j *IndexSyncJob) RunOnce(ctx context.Context) error {
	for _, sym := range j.Symbols {
		closes, err := j.BN.DailyCloses(ctx, sym, 500)
		if err != nil {
			log.Printf("index sync: %s: %v", sym, err)
			continue
		}
		n := 0
		for _, k := range closes {
			if err := store.UpsertIndexClose(ctx, j.DB, sym, k.OpenTime, k.Close); err != nil {
				log.Printf("index sync: upsert %s: %v", sym, err)
				break
			}
			n++
		}
		log.Printf("index sync: %s upserted %d daily closes", sym, n)
	}
	return nil
}

// Start runs an immediate sync then repeats every `interval` (daily) until ctx
// is cancelled.
func (j *IndexSyncJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		if err := j.RunOnce(ctx); err != nil {
			log.Printf("index sync: initial run: %v", err)
		}
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := j.RunOnce(ctx); err != nil {
					log.Printf("index sync: run: %v", err)
				}
			}
		}
	}()
}
