// Synthetic, in-memory data for the explicitly enabled local design preview.
// This middleware is never registered for a production build or normal dev.
/** @returns {import('vite').Plugin} */
export function previewApi() {
  const day = 86_400_000;
  const now = Date.now();
  const curve = Array.from({ length: 361 }, (_, i) => {
    const nav =
      1 + i * 0.00053 + Math.sin(i / 17) * 0.023 + Math.sin(i / 4.7) * 0.004;
    return {
      taken_at: now - ((360 - i) * day) / 2,
      nav,
      total_shares: 100000,
      total_equity: nav * 100000,
      source: "scheduled",
    };
  });
  const latest = curve[curve.length - 1];
  const friends = [
    {
      id: 1,
      name: "XG",
      username: "operator",
      is_admin: true,
      shares: 52000,
      active: true,
      created_at: now - 180 * day,
    },
    {
      id: 2,
      name: "成员 A",
      username: "member",
      is_admin: false,
      shares: 28000,
      active: true,
      created_at: now - 160 * day,
    },
    {
      id: 3,
      name: "成员 B",
      username: "member-b",
      is_admin: false,
      shares: 20000,
      active: true,
      created_at: now - 120 * day,
    },
  ];
  const aggregate = () => ({
    friends: friends.map((f) => ({
      ...f,
      net_deposits: f.shares,
      value_usdt: f.shares * latest.nav,
      pnl_usdt: f.shares * (latest.nav - 1),
      pnl_pct: latest.nav - 1,
    })),
    latest_nav: latest.nav,
    latest_equity: latest.total_equity,
    snapshot_at_ms: now - 8 * 60_000,
  });
  const open = [
    {
      symbol: "BTCUSDT",
      side: "LONG",
      quantity: 0.6,
      entry_price: 92400,
      mark_price: 95120,
      unrealized_pnl: 1632,
      realized_pnl: 268.4,
      commission: 28.2,
      leverage: 3,
      entry_time: now - 3.4 * day,
      status: "OPEN",
    },
    {
      symbol: "ETHUSDT",
      side: "LONG",
      quantity: 12,
      entry_price: 3248,
      mark_price: 3195,
      unrealized_pnl: -636,
      realized_pnl: 0,
      commission: 18.6,
      leverage: 3,
      entry_time: now - 1.8 * day,
      status: "OPEN",
    },
    {
      symbol: "SOLUSDT",
      side: "SHORT",
      quantity: 90,
      entry_price: 182.6,
      mark_price: 176.2,
      unrealized_pnl: 576,
      realized_pnl: 120,
      commission: 9.4,
      leverage: 2,
      entry_time: now - 0.7 * day,
      status: "OPEN",
    },
  ];
  const closed = Array.from({ length: 32 }, (_, i) => ({
    symbol: ["BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"][i % 4],
    side: i % 3 === 0 ? "SHORT" : "LONG",
    quantity: 1 + i / 10,
    entry_price: [90800, 3020, 164, 608][i % 4],
    exit_price: [92200, 3180, 161, 630][i % 4],
    entry_time: now - (i * 2 + 3) * day,
    exit_time: now - (i * 2 + 1) * day,
    realized_pnl: i % 3 ? 345.2 + i * 12 : -164.8 - i * 5,
    commission: 5.3 + i / 2,
    status: "CLOSED",
  }));
  const daily = Array.from({ length: 180 }, (_, i) => {
    const d = new Date(now - (179 - i) * day);
    const date = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Shanghai",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).format(d);
    const realized_pnl = Math.sin(i * 1.8) * 480 + 110;
    return {
      date,
      realized_pnl,
      commission: 12.6,
      net: realized_pnl - 12.6,
      fills: i % 6 === 0 ? 0 : 4 + (i % 9),
    };
  }).filter((d) => d.fills > 0);
  const ledger = friends.map((f, i) => ({
    id: i + 1,
    friend_id: f.id,
    username: f.username,
    name: f.name,
    type: "deposit",
    amount_usdt: f.shares,
    occurred_at: f.created_at,
    nav_at_event: 1,
    shares_delta: f.shares,
    source: "manual",
    note: "演示记录",
  }));
  /** @param {{ realized_pnl: number, commission: number }} p */
  const net = (p) => p.realized_pnl - p.commission;

  return {
    name: "xg-local-preview",
    configureServer(server) {
      server.middlewares.use(async (req, res, next) => {
        const url = new URL(req.url ?? "/", "http://localhost");
        if (!url.pathname.startsWith("/api/")) return next();
        /** @param {unknown} body */
        const send = (body, status = 200) => {
          res.statusCode = status;
          res.setHeader("Content-Type", "application/json");
          res.end(JSON.stringify(body));
        };
        /** @type {Record<string, any>} */
        let input = {};
        try {
          if (req.method === "POST") {
            let body = "";
            for await (const chunk of req) body += chunk;
            input = body ? JSON.parse(body) : {};
          }
        } catch {
          return send({ error: "请求格式无效" }, 400);
        }
        const role =
          req.headers.cookie
            ?.split(";")
            .map((s) => s.trim())
            .find((s) => s.startsWith("xg_preview_role="))
            ?.split("=")[1] ?? "admin";
        const me = role === "member" ? friends[1] : friends[0];
        const path = url.pathname;
        if (path === "/api/login") {
          const selected = input.username === "member" ? "member" : "admin";
          res.setHeader(
            "Set-Cookie",
            `xg_preview_role=${selected}; Path=/; HttpOnly; SameSite=Lax`,
          );
          return send(selected === "member" ? friends[1] : friends[0]);
        }
        if (path === "/api/logout") {
          res.setHeader(
            "Set-Cookie",
            "xg_preview_role=signed-out; Path=/; HttpOnly; SameSite=Lax",
          );
          return send({});
        }
        if (role === "signed-out") return send({ error: "请先登录" }, 401);
        if (path.startsWith("/api/admin/") && !me.is_admin)
          return send({ error: "需要管理员权限" }, 403);
        if (path === "/api/me") return send(me);
        if (path === "/api/me/summary") {
          const row = aggregate().friends.find((f) => f.id === me.id);
          return send({
            ...row,
            latest_nav: latest.nav,
            latest_equity: latest.total_equity,
            snapshot_at_ms: now - 8 * 60_000,
          });
        }
        if (path === "/api/aggregate") return send(aggregate());
        if (path === "/api/equity-curve")
          return send(
            curve.filter(
              (p) =>
                !url.searchParams.get("from") ||
                p.taken_at >= Number(url.searchParams.get("from")),
            ),
          );
        if (path === "/api/index-prices")
          return send({
            qqq: curve.map((p, i) => ({
              t: p.taken_at,
              close: 450 * (1 + i * 0.00021 + Math.sin(i / 24) * 0.012),
            })),
            spy: curve.map((p, i) => ({
              t: p.taken_at,
              close: 560 * (1 + i * 0.00013 + Math.sin(i / 28) * 0.006),
            })),
          });
        if (path === "/api/positions/open") return send(open);
        if (path === "/api/positions/closed")
          return send(
            closed.slice(0, Number(url.searchParams.get("limit") ?? 50)),
          );
        if (path === "/api/positions/allocation") {
          const notional = open.reduce(
            (sum, p) => sum + p.quantity * p.mark_price,
            0,
          );
          return send({
            equity: latest.total_equity,
            margin_used: 37650,
            free_cash: latest.total_equity - 37650,
            leverage: notional / latest.total_equity,
            notional,
            update_time: now - 45_000,
            positions: open.map((p) => ({
              symbol: p.symbol,
              side: p.side,
              notional: p.quantity * p.mark_price,
              pct: (p.quantity * p.mark_price) / notional,
            })),
          });
        }
        if (path === "/api/positions/daily-pnl") return send(daily);
        if (path === "/api/positions/stats") {
          const wins = closed.filter((p) => net(p) > 0);
          const losses = closed.filter((p) => net(p) < 0);
          const avgWin = wins.reduce((s, p) => s + net(p), 0) / wins.length;
          const avgLoss =
            losses.reduce((s, p) => s + net(p), 0) / losses.length;
          return send({
            window: Number(url.searchParams.get("window") ?? 500),
            stats: {
              total: closed.length,
              wins: wins.length,
              losses: losses.length,
              win_rate: wins.length / closed.length,
              total_pnl: closed.reduce((s, p) => s + net(p), 0),
              avg_win_usdt: avgWin,
              avg_loss_usdt: avgLoss,
              win_loss_ratio: avgWin / Math.abs(avgLoss),
              avg_hold_hours: 48,
              median_hold_hours: 48,
            },
            by_symbol: [...new Set(closed.map((p) => p.symbol))].map(
              (symbol) => {
                const ps = closed.filter((p) => p.symbol === symbol);
                return {
                  symbol,
                  trades: ps.length,
                  wins: ps.filter((p) => net(p) > 0).length,
                  total_pnl: ps.reduce((s, p) => s + net(p), 0),
                  win_rate: ps.filter((p) => net(p) > 0).length / ps.length,
                };
              },
            ),
          });
        }
        if (path === "/api/admin/friends" && req.method === "POST") {
          const friend = {
            name: String(input.name ?? ""),
            username: String(input.username ?? ""),
            is_admin: !!input.is_admin,
            id: friends.length + 1,
            active: true,
            created_at: Date.now(),
            shares: 0,
          };
          friends.push(friend);
          return send({ id: friend.id });
        }
        if (path === "/api/admin/friends") return send(friends);
        if (/^\/api\/admin\/friends\/\d+\/active$/.test(path)) {
          const friend = friends.find(
            (f) => f.id === Number(path.split("/")[4]),
          );
          if (!friend) return send({ error: "成员不存在" }, 404);
          friend.active = !!input.active;
          return send({ id: friend.id, active: friend.active });
        }
        if (path === "/api/admin/cash-events" && req.method === "POST") {
          const nav = Number(input.manual_nav) || latest.nav;
          const shares =
            (Number(input.amount_usdt) / nav) *
            (input.type === "withdraw" ? -1 : 1);
          const friend = friends.find((f) => f.username === input.username);
          if (!friend) return send({ error: "请选择成员" }, 400);
          ledger.unshift({
            id: ledger.length + 1,
            friend_id: friend.id,
            username: friend.username,
            name: friend.name,
            type: input.type,
            amount_usdt: Number(input.amount_usdt),
            occurred_at: Date.now(),
            nav_at_event: nav,
            shares_delta: shares,
            source: "manual",
            note: input.note ?? "",
          });
          return send({
            nav_at_event: nav,
            shares_delta: shares,
            equity_at_evt: latest.total_equity,
            manual_nav: !!input.manual_nav,
          });
        }
        if (path === "/api/admin/cash-events") return send(ledger);
        if (path === "/api/admin/recent-fills")
          return send(
            closed
              .slice(0, 12)
              .map((p, i) => ({
                id: i + 1,
                trade_id: 2000 + i,
                order_id: 1000 + i,
                symbol: p.symbol,
                side: i % 2 ? "BUY" : "SELL",
                position_side: p.side,
                price: p.exit_price,
                qty: p.quantity,
                quote_qty: p.quantity * p.exit_price,
                realized_pnl: p.realized_pnl,
                commission: p.commission,
                maker: false,
                fill_time: p.exit_time,
              })),
          );
        if (path === "/api/admin/snapshot")
          return send({ ...latest, taken_at: Date.now() });
        if (path === "/api/me/events")
          return send(ledger.filter((r) => r.username === me.username));
        return send({ error: "本地预览未提供此接口" }, 404);
      });
    },
  };
}
