// dashboard is the long-running HTTP server.
//
//	$ dashboard
//
// Reads .env, opens fund.db, wires up the API handlers, and launches three
// scheduled jobs: hourly NAV snapshot, hourly trade-fills sync, and (later)
// daily transfer reconciliation. Dashboard is completely independent of
// nofx — Binance is the sole external data source.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiagao/fund-dashboard/backend/api"
	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/config"
	"github.com/xiagao/fund-dashboard/backend/positions"
	"github.com/xiagao/fund-dashboard/backend/scheduler"
	"github.com/xiagao/fund-dashboard/backend/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		log.Fatalf("mkdir data: %v", err)
	}

	db, err := store.Open(cfg.FundDBPath)
	if err != nil {
		log.Fatalf("open fund.db: %v", err)
	}
	defer db.Close()

	srv := &api.Server{
		DB:           db,
		JWTSecret:    cfg.JWTSecret,
		CookieSecure: os.Getenv("COOKIE_SECURE") != "false", // default true; set false for local http dev
		// Trust X-Real-IP only when explicitly told we sit behind our own nginx
		// (TRUST_PROXY_HEADERS=true in the prod .env). Default false so local dev
		// and any accidental direct exposure fall back to the raw socket peer.
		TrustProxyHeaders: os.Getenv("TRUST_PROXY_HEADERS") == "true",
	}

	// In production we ship the SvelteKit build alongside the binary. Auto-detect
	// it on disk; if absent (typical local dev where vite serves on :3100), the
	// API still works on its own.
	staticDir := envOr("STATIC_DIR", "./web_build")
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		srv.StaticDir = staticDir
		log.Printf("serving SPA from %s", staticDir)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Binance client powers: NAV snapshots, cash-event auto-NAV, and the
	// trade transparency panels. Without it, the dashboard is read-only over
	// the existing fund.db data.
	if cfg.BinanceAPIKey != "" && cfg.BinanceAPISecret != "" {
		srv.Binance = binance.New(cfg.BinanceAPIKey, cfg.BinanceAPISecret)

		// NAV snapshot every 30 minutes — equity curve grows ~48 points/day,
		// feels alive without burning Binance calls.
		snapJob := &scheduler.SnapshotJob{DB: db, Binance: srv.Binance}
		snapJob.Start(ctx, 30*time.Minute)
		log.Println("snapshot scheduler started (30min interval)")

		// Trade-fills sync every hour — closed trades aren't time-sensitive,
		// and userTrades is heavier (one call per active symbol).
		tradesJob := &scheduler.TradesSyncJob{DB: db, BN: srv.Binance}
		tradesJob.Start(ctx, time.Hour)
		log.Println("trades sync scheduler started (1h interval)")

		// Positions orchestrator reads OPEN live from Binance, CLOSED from
		// fund.db's binance_fills. No nofx coupling.
		srv.Positions = &positions.Orchestrator{
			BN:       srv.Binance,
			FundDB:   db,
			Lookback: 90 * 24 * time.Hour,
			CacheTTL: 60 * time.Second,
		}
		log.Println("positions orchestrator wired (Binance OPEN + fund.db CLOSED)")
	} else {
		log.Println("WARNING: Binance keys not set — /api/admin/snapshot, /api/admin/cash-events, AND /api/positions/* will return 503")
	}

	// Index benchmark sync uses the PUBLIC klines endpoint (no signing), so it
	// runs whether or not trading keys are set. Reuse srv.Binance if present,
	// else a keyless client just for market data.
	indexBN := srv.Binance
	if indexBN == nil {
		indexBN = binance.New("", "")
	}
	// Every 30 min (matching the NAV snapshot cadence): the current day's 1d
	// candle is live, so frequent upserts keep today's benchmark point — and
	// the "跑赢大盘" delta — current instead of up to a day stale. Past days are
	// closed candles and stay fixed.
	indexJob := &scheduler.IndexSyncJob{DB: db, BN: indexBN, Symbols: []string{"QQQUSDT", "SPYUSDT"}}
	indexJob.Start(ctx, 30*time.Minute)
	log.Println("index sync scheduler started (30min interval; QQQUSDT, SPYUSDT)")

	httpSrv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// WriteTimeout covers handler time too — positions refresh and
		// cash-event recording each wait on Binance (15s client timeout per
		// call), so leave generous headroom. nginx in front reads for 60s.
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		log.Printf("dashboard listening on %s", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()
	<-stop
	log.Println("shutdown")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
