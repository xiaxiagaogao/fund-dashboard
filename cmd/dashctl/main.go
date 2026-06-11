// dashctl is a small CLI for operating the fund-dashboard locally before the
// HTTP server / frontend exist. Use it to:
//
//	dashctl init-self  <name> <username> <password>
//	    Create the operator (you) row with is_admin=1. Done once at bootstrap.
//
//	dashctl add-friend <name> <username> <password>
//	    Create a friend account.
//
//	dashctl deposit    <username> <amount_usdt>  [--occurred-at-ms=N] [--note=...]
//	    Record a manual deposit. Pulls live Binance equity, computes NAV, mints shares.
//
//	dashctl withdraw   <username> <amount_usdt>  [--occurred-at-ms=N] [--note=...]
//	    Record a withdrawal (burns shares at current NAV).
//
//	dashctl snapshot
//	    Take a one-shot scheduled NAV snapshot now.
//
//	dashctl list-fills [--limit=N]
//	    Show recent nofx fills with bot/manual attribution. (Reads NOFX_DB_PATH.)
//
//	dashctl status
//	    Print pool summary: total equity, total shares, NAV, per-friend value.
//
// Configuration: env vars or .env file (BINANCE_API_KEY, BINANCE_API_SECRET,
// FUND_DB_PATH, NOFX_DB_PATH).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/xiagao/fund-dashboard/backend/binance"
	"github.com/xiagao/fund-dashboard/backend/config"
	"github.com/xiagao/fund-dashboard/backend/nav"
	"github.com/xiagao/fund-dashboard/backend/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	cfg, err := config.Load()
	if err != nil {
		die("load config: %v", err)
	}
	if err := os.MkdirAll("data", 0o755); err != nil {
		die("mkdir data: %v", err)
	}

	ctx := context.Background()

	switch cmd {
	case "init-self":
		runCreateFriend(ctx, cfg, args, true)
	case "add-friend":
		runCreateFriend(ctx, cfg, args, false)
	case "deposit":
		runCashEvent(ctx, cfg, args, store.EventDeposit)
	case "withdraw":
		runCashEvent(ctx, cfg, args, store.EventWithdraw)
	case "snapshot":
		runSnapshot(ctx, cfg)
	case "list-fills":
		runListFills(ctx, cfg, args)
	case "status":
		runStatus(ctx, cfg)
	case "set-password":
		runSetPassword(ctx, cfg, args)
	case "backfill-history":
		runBackfillHistory(ctx, cfg, args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `dashctl — friend-fund dashboard CLI

Subcommands:
  init-self  <name> <username> <password>
  add-friend <name> <username> <password>
  deposit    <username> <amount_usdt> [flags]
  withdraw   <username> <amount_usdt> [flags]
  snapshot
  list-fills [--limit=N]
  status
  set-password <username>             (reads new password from env $DASHCTL_PASSWORD,
                                       never prints or stores it anywhere except
                                       fund.db as a bcrypt hash)

Reads config from .env (or env vars):
  BINANCE_API_KEY, BINANCE_API_SECRET   (read-only key)
  FUND_DB_PATH=./data/fund.db
  NOFX_DB_PATH=/path/to/nofx/data.db    (only needed for list-fills)
`)
}

func openFundDB(cfg *config.Config) *sql.DB {
	db, err := store.Open(cfg.FundDBPath)
	if err != nil {
		die("open fund db: %v", err)
	}
	return db
}

func runBackfillHistory(ctx context.Context, cfg *config.Config, args []string) {
	fs := flag.NewFlagSet("backfill-history", flag.ExitOnError)
	days := fs.Int("days", 30, "how many days back to walk")
	if err := fs.Parse(args); err != nil {
		die("parse flags: %v", err)
	}
	if err := config.RequireBinance(cfg); err != nil {
		die("%v", err)
	}

	bn := binance.New(cfg.BinanceAPIKey, cfg.BinanceAPISecret)
	lookback := time.Duration(*days) * 24 * time.Hour
	since := time.Now().Add(-lookback).UnixMilli()

	fmt.Printf("walking /fapi/v1/income REALIZED_PNL for last %d days to find active symbols…\n", *days)
	syms, err := bn.WalkRealizedIncomeSymbols(ctx, lookback)
	if err != nil {
		die("walk income: %v", err)
	}
	if len(syms) == 0 {
		fmt.Println("no REALIZED_PNL income rows found in window — also adding currently OPEN positions' symbols")
	}
	// Also fold in currently OPEN positions — they may not have realized PnL yet.
	risks, err := bn.PositionRisks(ctx)
	if err != nil {
		die("position risks: %v", err)
	}
	for _, r := range risks {
		syms[r.Symbol] = struct{}{}
	}
	fmt.Printf("active symbols: %d\n", len(syms))

	db := openFundDB(cfg)
	defer db.Close()

	totalNew := 0
	totalSeen := 0
	for sym := range syms {
		// For each symbol, walk from `since` (or from the newest fill we
		// already have, whichever is later — keeps re-runs idempotent + fast).
		last, _ := store.LastFillTimeBySymbol(ctx, db, sym)
		startAt := since
		if last > startAt {
			startAt = last + 1
		}
		fills, err := bn.WalkUserTradesSince(ctx, sym, startAt)
		if err != nil {
			fmt.Printf("  %-12s ERROR: %v\n", sym, err)
			continue
		}
		newCount := 0
		for _, t := range fills {
			_, inserted, err := store.InsertFillIgnore(ctx, db, store.BinanceFill{
				BinanceTradeID: t.ID, BinanceOrderID: t.OrderID,
				Symbol: t.Symbol, Side: t.Side, PositionSide: t.PositionSide,
				Price: t.Price, Qty: t.Qty, QuoteQty: t.QuoteQty,
				RealizedPnL: t.RealizedPnL, Commission: t.Commission,
				Maker: t.Maker, Buyer: t.Buyer,
				FillTime: t.Time,
			})
			if err != nil {
				fmt.Printf("  %-12s INSERT ERROR: %v\n", sym, err)
				break
			}
			if inserted {
				newCount++
			}
		}
		totalNew += newCount
		totalSeen += len(fills)
		fmt.Printf("  %-12s pulled=%-4d inserted=%-4d\n", sym, len(fills), newCount)
	}

	if err := store.WriteAudit(ctx, db, "system", "history.backfill", map[string]any{
		"days": *days, "symbols": len(syms), "new_fills": totalNew, "seen_fills": totalSeen,
	}); err != nil {
		die("audit: %v", err)
	}
	fmt.Printf("\nbackfill complete: %d new fills inserted (%d total fetched, %d already in DB)\n",
		totalNew, totalSeen, totalSeen-totalNew)
}

func runSetPassword(ctx context.Context, cfg *config.Config, args []string) {
	if len(args) != 1 {
		die("usage: set-password <username>  (new password via env $DASHCTL_PASSWORD)")
	}
	username := args[0]
	pw := os.Getenv("DASHCTL_PASSWORD")
	if len(pw) < 8 {
		die("DASHCTL_PASSWORD env must be set, ≥8 chars")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		die("bcrypt: %v", err)
	}
	db := openFundDB(cfg)
	defer db.Close()
	res, err := db.ExecContext(ctx,
		`UPDATE friends SET password_hash = ? WHERE username = ?`, string(hash), username)
	if err != nil {
		die("update password: %v", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		die("no friend found with username %q", username)
	}
	if err := store.WriteAudit(ctx, db, "system", "friend.password_reset", map[string]any{
		"username": username,
	}); err != nil {
		die("audit: %v", err)
	}
	fmt.Printf("password updated for %s (browser sessions issued before this remain valid until cookie expires; sign out + back in to refresh)\n", username)
}

func runCreateFriend(ctx context.Context, cfg *config.Config, args []string, isAdmin bool) {
	if len(args) != 3 {
		die("usage: %s <name> <username> <password>", boolStr(isAdmin, "init-self", "add-friend"))
	}
	name, username, password := args[0], args[1], args[2]

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		die("bcrypt: %v", err)
	}
	db := openFundDB(cfg)
	defer db.Close()

	id, err := store.CreateFriend(ctx, db, name, username, string(hash), isAdmin)
	if err != nil {
		die("create friend: %v", err)
	}
	if err := store.WriteAudit(ctx, db, "system", "friend.create", map[string]any{
		"id": id, "username": username, "is_admin": isAdmin,
	}); err != nil {
		die("audit: %v", err)
	}
	fmt.Printf("created friend id=%d username=%s admin=%v\n", id, username, isAdmin)
}

func runCashEvent(ctx context.Context, cfg *config.Config, args []string, evtType string) {
	if len(args) < 2 {
		die("usage: %s <username> <amount_usdt> [--occurred-at-ms=N] [--note=...] [--manual-nav=N] [--skip-bootstrap-check]", evtType)
	}
	username := args[0]
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil || amount <= 0 {
		die("invalid amount %q", args[1])
	}

	fs := flag.NewFlagSet(evtType, flag.ExitOnError)
	occurredAt := fs.Int64("occurred-at-ms", 0, "event time in unix ms (default: now)")
	note := fs.String("note", "", "free-text note")
	manualNAV := fs.Float64("manual-nav", 0, "override: use this NAV instead of computing from live Binance equity")
	skipBootstrap := fs.Bool("skip-bootstrap-check", false, "bypass bootstrap equity-vs-amount alignment check")
	if err := fs.Parse(args[2:]); err != nil {
		die("parse flags: %v", err)
	}
	if *occurredAt == 0 {
		*occurredAt = time.Now().UnixMilli()
	}

	db := openFundDB(cfg)
	defer db.Close()

	friend, err := store.GetFriendByUsername(ctx, db, username)
	if err != nil {
		die("lookup friend: %v", err)
	}
	totalShares, err := store.TotalShares(ctx, db)
	if err != nil {
		die("total shares: %v", err)
	}

	var (
		sharesDelta, navUsed float64
		liveEquity           float64
	)

	if *manualNAV > 0 {
		// Manual override: trust the supplied NAV, skip Binance.
		navUsed = *manualNAV
		if evtType == store.EventDeposit {
			sharesDelta = amount / navUsed
		} else {
			sharesDelta = -amount / navUsed
		}
	} else {
		if err := config.RequireBinance(cfg); err != nil {
			die("%v (or pass --manual-nav=N)", err)
		}
		bn := binance.New(cfg.BinanceAPIKey, cfg.BinanceAPISecret)
		equity, _, err := bn.AccountEquity(ctx)
		if err != nil {
			die("binance account equity: %v", err)
		}
		liveEquity = equity

		if totalShares == 0 && !*skipBootstrap {
			if equity <= 0 {
				die("bootstrap requires Binance equity > 0; deposit USDT first or pass --skip-bootstrap-check")
			}
			div := equity * 0.01
			if amount > equity+div || amount < equity-div {
				die("bootstrap amount %.4f deviates from current Binance equity %.4f by >1%%; align the balance or pass --skip-bootstrap-check", amount, equity)
			}
		}

		switch evtType {
		case store.EventDeposit:
			sharesDelta, navUsed = nav.ComputeMintAfterArrival(amount, equity, totalShares)
		case store.EventWithdraw:
			burned, n, berr := nav.ComputeBurnAfterDeparture(amount, equity, totalShares)
			if berr != nil {
				die("compute burn: %v", berr)
			}
			sharesDelta, navUsed = -burned, n
		}
	}

	if evtType == store.EventWithdraw {
		myShares, _ := store.FriendShares(ctx, db, friend.ID)
		if -sharesDelta > myShares+1e-9 {
			die("friend %s only holds %.6f shares; cannot burn %.6f", username, myShares, -sharesDelta)
		}
	}

	id, err := store.InsertCashEvent(ctx, db, store.CashEventInput{
		FriendID: friend.ID, Type: evtType, AmountUSDT: amount,
		OccurredAt: *occurredAt, NAVAtEvent: navUsed, SharesDelta: sharesDelta,
		Source: store.SourceManual, Note: *note,
	})
	if err != nil {
		die("insert cash event: %v", err)
	}

	postShares := totalShares + sharesDelta
	postEquity := liveEquity
	if postEquity == 0 {
		postEquity = postShares * navUsed
	}
	if _, err := store.InsertNAVSnapshot(ctx, db, store.NAVSnapshot{
		TakenAt: *occurredAt, TotalEquityUSDT: postEquity,
		TotalShares: postShares, NAV: navUsed, Source: store.SnapshotCashEvent,
	}); err != nil && err != store.ErrDuplicateSnapshot {
		die("snapshot: %v", err)
	}

	store.WriteAudit(ctx, db, username, "cash_event.create", map[string]any{
		"id": id, "type": evtType, "amount_usdt": amount, "nav_at_event": navUsed,
		"shares_delta": sharesDelta, "live_equity": liveEquity, "manual_nav": *manualNAV > 0,
	})

	fmt.Printf("recorded %s id=%d:\n", evtType, id)
	fmt.Printf("  friend        : %s (id=%d)\n", username, friend.ID)
	fmt.Printf("  amount_usdt   : %.4f\n", amount)
	if liveEquity > 0 {
		fmt.Printf("  live equity   : %.4f USDT (post-event, from Binance)\n", liveEquity)
	} else {
		fmt.Printf("  manual NAV    : (no Binance call)\n")
	}
	fmt.Printf("  NAV at event  : %.6f\n", navUsed)
	fmt.Printf("  shares delta  : %+.6f\n", sharesDelta)
	fmt.Printf("  shares total  : %.6f\n", postShares)
}

func runSnapshot(ctx context.Context, cfg *config.Config) {
	if err := config.RequireBinance(cfg); err != nil {
		die("%v", err)
	}
	db := openFundDB(cfg)
	defer db.Close()

	bn := binance.New(cfg.BinanceAPIKey, cfg.BinanceAPISecret)
	equity, raw, err := bn.AccountEquity(ctx)
	if err != nil {
		die("binance account equity: %v", err)
	}
	totalShares, _ := store.TotalShares(ctx, db)
	currentNAV := nav.CurrentNAV(equity, totalShares)
	now := time.Now().UnixMilli()

	if _, err := store.InsertNAVSnapshot(ctx, db, store.NAVSnapshot{
		TakenAt:         now,
		TotalEquityUSDT: equity,
		TotalShares:     totalShares,
		NAV:             currentNAV,
		Source:          store.SnapshotScheduled,
	}); err != nil {
		die("snapshot insert: %v", err)
	}
	store.WriteAudit(ctx, db, "system", "snapshot.write", map[string]any{
		"taken_at": now, "equity": equity, "shares": totalShares, "nav": currentNAV,
	})
	fmt.Printf("snapshot @ %s\n", time.UnixMilli(now).Format(time.RFC3339))
	fmt.Printf("  total equity (USDT): %.4f\n", equity)
	fmt.Printf("    walletBalance    : %s\n", raw.TotalWalletBalance)
	fmt.Printf("    unrealizedProfit : %s\n", raw.TotalUnrealizedProfit)
	fmt.Printf("  total shares       : %.6f\n", totalShares)
	fmt.Printf("  NAV                : %.6f\n", currentNAV)
}

func runListFills(ctx context.Context, cfg *config.Config, args []string) {
	fs := flag.NewFlagSet("list-fills", flag.ExitOnError)
	limit := fs.Int("limit", 20, "max rows to display")
	if err := fs.Parse(args); err != nil {
		die("parse flags: %v", err)
	}
	db := openFundDB(cfg)
	defer db.Close()
	fills, err := store.ListRecentFills(ctx, db, *limit)
	if err != nil {
		die("list fills: %v", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tSYMBOL\tSIDE\tPOS\tQTY\tPRICE\tQUOTE\tPnL")
	for _, f := range fills {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.4f\t%.4f\t%.2f\t%+.4f\n",
			time.UnixMilli(f.FillTime).Format("01-02 15:04"),
			f.Symbol, f.Side, f.PositionSide, f.Qty, f.Price, f.QuoteQty, f.RealizedPnL,
		)
	}
	w.Flush()
}

func runStatus(ctx context.Context, cfg *config.Config) {
	db := openFundDB(cfg)
	defer db.Close()

	totalShares, _ := store.TotalShares(ctx, db)
	latest, _ := store.LatestNAV(ctx, db)
	friends, _ := store.ListFriends(ctx, db)

	fmt.Printf("Pool snapshot (latest persisted, taken_at=%s, source=%s)\n",
		time.UnixMilli(latest.TakenAt).Format(time.RFC3339), latest.Source)
	fmt.Printf("  total equity USDT : %.4f\n", latest.TotalEquityUSDT)
	fmt.Printf("  total shares      : %.6f\n", totalShares)
	fmt.Printf("  NAV               : %.6f\n", latest.NAV)
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "USERNAME\tNAME\tADMIN\tSHARES\tNET DEP USDT\tVALUE USDT\tPNL %")
	for _, f := range friends {
		shares, _ := store.FriendShares(ctx, db, f.ID)
		net, _ := store.FriendNetDeposits(ctx, db, f.ID)
		value := shares * latest.NAV
		pnlPct := 0.0
		if net > 0 {
			pnlPct = (value - net) / net * 100
		}
		fmt.Fprintf(w, "%s\t%s\t%v\t%.6f\t%.4f\t%.4f\t%+.2f\n",
			f.Username, f.Name, f.IsAdmin, shares, net, value, pnlPct)
	}
	w.Flush()
}

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

func boolStr(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}
