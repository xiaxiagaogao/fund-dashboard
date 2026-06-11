// Typed fetch wrappers for the Go backend's /api/* surface.
// Cookies are HttpOnly + same-origin, so `credentials: 'include'` is enough.

export type Me = {
  id: number;
  username: string;
  name: string;
  is_admin: boolean;
};

export type Summary = {
  shares: number;
  net_deposits: number;
  value_usdt: number;
  pnl_usdt: number;
  pnl_pct: number;
  latest_nav: number;
  latest_equity: number;
  snapshot_at_ms: number;
};

export type CashEvent = {
  id: number;
  type: 'deposit' | 'withdraw';
  amount_usdt: number;
  occurred_at: number;
  nav_at_event: number;
  shares_delta: number;
  shares_after: number;
  source: 'manual' | 'binance_transfer';
  binance_tx_id: string | null;
  note: string | null;
};

export type EquityPoint = {
  taken_at: number;
  total_equity: number;
  total_shares: number;
  nav: number;
  source: 'scheduled' | 'cash_event';
};

export type AggregateRow = {
  username: string;
  name: string;
  is_admin: boolean;
  shares: number;
  net_deposits: number;
  value_usdt: number;
  pnl_usdt: number;
  pnl_pct: number;
};

export type Aggregate = {
  friends: AggregateRow[];
  latest_nav: number;
  latest_equity: number;
  snapshot_at_ms: number;
};

export type FriendRow = {
  id: number;
  name: string;
  username: string;
  is_admin: boolean;
  created_at: number;
};

export type Position = {
  symbol: string;
  side: 'LONG' | 'SHORT' | string;
  quantity: number;
  entry_price: number;
  entry_time: number;
  exit_price?: number;
  exit_time?: number;
  realized_pnl: number;
  /** Only meaningful for OPEN positions (live mark-to-market). */
  unrealized_pnl?: number;
  /** Only meaningful for OPEN positions. */
  mark_price?: number;
  liquidation_price?: number;
  leverage?: number;
  status: 'OPEN' | 'CLOSED' | string;
};

export type Stats = {
  total: number;
  wins: number;
  losses: number;
  win_rate: number;
  total_pnl: number;
  avg_win_usdt: number;
  avg_loss_usdt: number;
  win_loss_ratio: number;
  avg_hold_hours: number;
  median_hold_hours: number;
};

export type SymbolPnL = {
  symbol: string;
  trades: number;
  wins: number;
  total_pnl: number;
  win_rate: number;
};

export type StatsResponse = {
  window: number;
  stats: Stats;
  by_symbol: SymbolPnL[];
};

/** One Binance fill from fund.db's own binance_fills table (admin view). */
export type RecentFill = {
  id: number;
  trade_id: number;
  order_id: number;
  symbol: string;
  side: 'BUY' | 'SELL' | string;
  position_side: string;
  price: number;
  qty: number;
  quote_qty: number;
  realized_pnl: number;
  commission: number;
  maker: boolean;
  fill_time: number;
};

export class ApiError extends Error {
  constructor(public status: number, public payload: unknown, msg: string) {
    super(msg);
  }
}

async function req<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', ...(init.headers ?? {}) },
    ...init
  });
  const text = await res.text();
  let body: unknown = null;
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
  }
  if (!res.ok) {
    const msg =
      (body && typeof body === 'object' && 'error' in body && typeof (body as any).error === 'string')
        ? (body as any).error
        : `HTTP ${res.status}`;
    throw new ApiError(res.status, body, msg);
  }
  return body as T;
}

export const api = {
  login: (username: string, password: string) =>
    req<Me>('/api/login', { method: 'POST', body: JSON.stringify({ username, password }) }),

  logout: () => req<void>('/api/logout', { method: 'POST' }),

  me: () => req<Me>('/api/me'),
  mySummary: () => req<Summary>('/api/me/summary'),
  myEvents: () => req<CashEvent[]>('/api/me/events'),

  equityCurve: (fromMs?: number, toMs?: number) => {
    const q = new URLSearchParams();
    if (fromMs) q.set('from', String(fromMs));
    if (toMs) q.set('to', String(toMs));
    const qs = q.toString();
    return req<EquityPoint[]>('/api/equity-curve' + (qs ? '?' + qs : ''));
  },

  aggregate: () => req<Aggregate>('/api/aggregate'),

  openPositions: () => req<Position[]>('/api/positions/open'),
  closedPositions: (limit = 50) => req<Position[]>('/api/positions/closed?limit=' + limit),
  stats: (window = 200) => req<StatsResponse>('/api/positions/stats?window=' + window),

  exportCsvUrl: () => '/api/me/export.csv',

  admin: {
    listFriends: () => req<FriendRow[]>('/api/admin/friends'),
    createFriend: (input: { name: string; username: string; password: string; is_admin: boolean }) =>
      req<{ id: number }>('/api/admin/friends', { method: 'POST', body: JSON.stringify(input) }),
    listCashEvents: (limit = 200) =>
      req<Array<{
        id: number; friend_id: number; username: string; name: string;
        type: 'deposit' | 'withdraw'; amount_usdt: number; occurred_at: number;
        nav_at_event: number; shares_delta: number;
        source: 'manual' | 'binance_transfer'; binance_tx_id: string; note: string;
      }>>('/api/admin/cash-events?limit=' + limit),
    cashEvent: (input: {
      username: string;
      type: 'deposit' | 'withdraw';
      amount_usdt: number;
      occurred_at_ms?: number;
      note?: string;
      manual_nav?: number;
      skip_bootstrap_check?: boolean;
    }) =>
      req<{ id: number; nav_at_event: number; shares_delta: number; equity_at_evt: number; manual_nav: boolean }>(
        '/api/admin/cash-events',
        { method: 'POST', body: JSON.stringify(input) }
      ),
    recentFills: (limit = 50) => req<RecentFill[]>('/api/admin/recent-fills?limit=' + limit),
    snapshot: () =>
      req<{ id: number; taken_at: number; total_equity: number; total_shares: number; nav: number }>(
        '/api/admin/snapshot',
        { method: 'POST' }
      )
  }
};
