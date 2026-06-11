<script lang="ts">
  import { onMount } from 'svelte';
  import {
    api,
    type Me,
    type Summary,
    type Aggregate,
    type EquityPoint,
    type CashEvent,
    type Position,
    type StatsResponse
  } from '$lib/api';
  import { fmtUSDT, fmtShares, fmtSignedUSDT, fmtSignedPct, fmtRelativeTime } from '$lib/format';
  import EquityCurve from '$lib/components/EquityCurve.svelte';
  import MetricCard from '$lib/components/MetricCard.svelte';
  import FriendsTable from '$lib/components/FriendsTable.svelte';
  import EventsTable from '$lib/components/EventsTable.svelte';
  import StatsCards from '$lib/components/StatsCards.svelte';
  import OpenPositions from '$lib/components/OpenPositions.svelte';
  import ClosedTrades from '$lib/components/ClosedTrades.svelte';
  import SymbolPnLBars from '$lib/components/SymbolPnLBars.svelte';

  let me: Me | null = null;
  let summary: Summary | null = null;
  let aggregate: Aggregate | null = null;
  let curve: EquityPoint[] = [];
  let events: CashEvent[] = [];
  let openPositions: Position[] = [];
  let closedPositions: Position[] = [];
  let stats: StatsResponse | null = null;
  let nofxAvailable = true;
  let loading = true;
  let error = '';

  // Ticking clock so "X 分钟前" updates without a page refresh.
  let now = Date.now();

  async function load() {
    try {
      [me, summary, aggregate, curve, events] = await Promise.all([
        api.me(),
        api.mySummary(),
        api.aggregate(),
        api.equityCurve(),
        api.myEvents()
      ]);
    } catch (e) {
      error = e instanceof Error ? e.message : '加载失败';
      loading = false;
      return;
    }
    try {
      [openPositions, closedPositions, stats] = await Promise.all([
        api.openPositions(),
        api.closedPositions(100),
        api.stats(200)
      ]);
      nofxAvailable = true;
    } catch (e) {
      nofxAvailable = false;
    }
    loading = false;
  }

  // Lighter periodic refresh — only re-pulls the three endpoints that
  // genuinely change minute-to-minute. Heavy data (equity curve, events,
  // closed trades, stats) only changes on the hour, so reloading them
  // every 2 min would be wasteful.
  async function refreshLive() {
    try {
      const [s, a, op] = await Promise.all([
        api.mySummary(),
        api.aggregate(),
        api.openPositions()
      ]);
      summary = s;
      aggregate = a;
      openPositions = op;
    } catch {
      // Swallow transient failures — keep stale data on screen rather than blank.
    }
  }

  onMount(() => {
    load();
    const pollHandle = setInterval(refreshLive, 2 * 60 * 1000);
    const tickHandle = setInterval(() => (now = Date.now()), 1000);
    return () => {
      clearInterval(pollHandle);
      clearInterval(tickHandle);
    };
  });

  // Reactive relative-time text. The block references `now` so Svelte
  // re-runs it on every clock tick.
  let snapAgeText = '';
  $: {
    void now;
    snapAgeText = summary ? fmtRelativeTime(summary.snapshot_at_ms) : '';
  }
</script>

{#if loading}
  <div class="text-ink-400 text-sm">加载中…</div>
{:else if error}
  <div class="card p-6 text-loss-400">{error}</div>
{:else if me && summary && aggregate}
  <!-- Header -->
  <div class="mb-3 flex items-baseline justify-between">
    <h2 class="text-xl font-semibold tracking-tight">你好，{me.name}</h2>
    <div class="text-xs text-ink-400 flex items-center gap-2">
      <span class="inline-block w-1.5 h-1.5 rounded-full bg-accent-500 animate-pulse" title="live · 2 分钟轮询"></span>
      最新快照 · {snapAgeText}
    </div>
  </div>

  <!-- Per-friend hero stats -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
    <MetricCard
      label="我的估值"
      value={fmtUSDT(summary.value_usdt, 2) + ' USDT'}
      sub={fmtShares(summary.shares, 4) + ' 份额'}
      pill={fmtSignedPct(summary.pnl_pct)}
      pillSignal={summary.pnl_pct}
    />
    <MetricCard
      label="我的 PnL"
      value={fmtSignedUSDT(summary.pnl_usdt, 2) + ' USDT'}
      valueSignal={summary.pnl_usdt}
      sub={'累计投入 ' + fmtUSDT(summary.net_deposits, 2)}
    />
    <MetricCard
      label="基金 NAV / share"
      value={summary.latest_nav.toFixed(6)}
      sub={fmtSignedPct(summary.latest_nav - 1)}
    />
    <MetricCard
      label="基金总资金"
      value={fmtUSDT(summary.latest_equity, 2) + ' USDT'}
      sub={aggregate.friends.length + ' 位成员'}
    />
  </div>

  <!-- Equity curve -->
  <div class="mb-6">
    <EquityCurve points={curve} height={240} />
  </div>

  <!-- Friends table -->
  <div class="mb-8">
    <FriendsTable rows={aggregate.friends} me={me.username} />
  </div>

  <!-- Trade transparency section -->
  {#if nofxAvailable}
    <div class="flex items-baseline justify-between mb-3">
      <h3 class="text-lg font-semibold tracking-tight">交易透明度</h3>
      <div class="text-xs text-ink-400">所有成员可见 · 数据源 nofx</div>
    </div>

    <div class="mb-5">
      <StatsCards stats={stats?.stats ?? null} window={stats?.window ?? 0} />
    </div>

    <div class="mb-5">
      <OpenPositions positions={openPositions} />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-6">
      <div class="lg:col-span-2">
        <ClosedTrades positions={closedPositions} />
      </div>
      <div>
        <SymbolPnLBars rows={stats?.by_symbol ?? []} />
      </div>
    </div>
  {:else}
    <div class="card p-6 text-ink-400 text-sm mb-6">
      <div class="label mb-1">交易透明度</div>
      未挂 nofx 数据库（NOFX_DB_PATH 不可达）—— 部署到 nofx VPS 上之后这块会自动激活。
    </div>
  {/if}

  <!-- My events -->
  <div class="mb-6">
    <EventsTable {events} />
  </div>
{/if}
