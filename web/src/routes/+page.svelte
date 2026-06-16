<script lang="ts">
  import { onMount } from 'svelte';
  import {
    api,
    type Me,
    type Summary,
    type Aggregate,
    type EquityPoint,
    type CashEvent,
    type Allocation,
    type Position,
    type StatsResponse
  } from '$lib/api';
  import { fmtUSDT, fmtShares, fmtSignedUSDT, fmtSignedPct, fmtRelativeTime } from '$lib/format';
  import EquityCurve from '$lib/components/EquityCurve.svelte';
  import MetricCard from '$lib/components/MetricCard.svelte';
  import PositionDonuts from '$lib/components/PositionDonuts.svelte';
  import FriendsTable from '$lib/components/FriendsTable.svelte';
  import ClosedTrades from '$lib/components/ClosedTrades.svelte';
  import SymbolPnLBars from '$lib/components/SymbolPnLBars.svelte';

  let me: Me | null = null;
  let summary: Summary | null = null;
  let aggregate: Aggregate | null = null;
  let curve: EquityPoint[] = [];
  let events: CashEvent[] = [];
  let alloc: Allocation | null = null;
  let closedPositions: Position[] = [];
  let stats: StatsResponse | null = null;
  let positionsAvailable = true;
  let loading = true;
  let error = '';

  // Collapsible secondary sections — folded by default to keep the mobile view calm.
  let showReview = false;
  let showMembers = false;

  // Ticking clock so "X 分钟前" updates without a page refresh.
  let now = Date.now();

  // "我的" curve: at each snapshot, my shares as-of that moment × the NAV then.
  // events come back ordered by occurred_at ASC.
  $: mineValues = curve.map((pt) => {
    let shares = 0;
    for (const e of events) {
      if (e.occurred_at <= pt.taken_at) shares += e.shares_delta;
      else break;
    }
    return shares * pt.nav;
  });

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
      [alloc, closedPositions, stats] = await Promise.all([
        api.allocation(),
        api.closedPositions(100),
        api.stats(200)
      ]);
      positionsAvailable = true;
    } catch (e) {
      positionsAvailable = false;
    }
    loading = false;
  }

  // Light periodic refresh — only the things that move minute-to-minute.
  async function refreshLive() {
    try {
      const [s, a, al] = await Promise.all([api.mySummary(), api.aggregate(), api.allocation()]);
      summary = s;
      aggregate = a;
      alloc = al;
    } catch {
      // Keep stale data on screen rather than blanking on a transient failure.
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
      {snapAgeText}
    </div>
  </div>

  <!-- Hero: just my two numbers -->
  <div class="grid grid-cols-2 gap-3 sm:gap-4 mb-5">
    <MetricCard
      label="我的估值"
      value={fmtUSDT(summary.value_usdt, 2)}
      sub={fmtShares(summary.shares, 4) + ' 份额'}
      pill={fmtSignedPct(summary.pnl_pct)}
      pillSignal={summary.pnl_pct}
    />
    <MetricCard
      label="我的 PnL"
      value={fmtSignedUSDT(summary.pnl_usdt, 2)}
      valueSignal={summary.pnl_usdt}
      sub={'累计投入 ' + fmtUSDT(summary.net_deposits, 2)}
    />
  </div>

  <!-- Equity curve: 我的 / 基金 -->
  <div class="mb-5">
    <EquityCurve points={curve} {mineValues} variant="friend" height={220} />
  </div>

  <!-- Current holdings — two donuts -->
  {#if positionsAvailable}
    <div class="mb-5">
      <PositionDonuts {alloc} loading={!alloc} />
    </div>
  {/if}

  <!-- Collapsible: 复盘明细 (closed trades + per-symbol) -->
  {#if positionsAvailable}
    <div class="card overflow-hidden mb-4">
      <button
        class="w-full flex items-center gap-2 px-5 py-3.5 text-left table-row-hover"
        on:click={() => (showReview = !showReview)}
      >
        <span class="label">复盘明细</span>
        <span class="text-[11px] text-ink-500">平仓记录 · 标的盈亏</span>
        <svg
          class={'ml-auto w-4 h-4 text-ink-400 transition-transform ' + (showReview ? 'rotate-180' : '')}
          viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
        ><path d="M6 9l6 6 6-6" stroke-linecap="round" stroke-linejoin="round" /></svg>
      </button>
      {#if showReview}
        <div class="px-3 pb-4 pt-1 border-t border-ink-800/60">
          <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
            <div class="lg:col-span-2">
              <ClosedTrades positions={closedPositions} />
            </div>
            <div>
              <SymbolPnLBars rows={stats?.by_symbol ?? []} />
            </div>
          </div>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Collapsible: 成员对比 -->
  <div class="card overflow-hidden mb-4">
    <button
      class="w-full flex items-center gap-2 px-5 py-3.5 text-left table-row-hover"
      on:click={() => (showMembers = !showMembers)}
    >
      <span class="label">成员对比</span>
      <span class="text-[11px] text-ink-500">{aggregate.friends.length} 位成员 · 互相可见</span>
      <svg
        class={'ml-auto w-4 h-4 text-ink-400 transition-transform ' + (showMembers ? 'rotate-180' : '')}
        viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
      ><path d="M6 9l6 6 6-6" stroke-linecap="round" stroke-linejoin="round" /></svg>
    </button>
    {#if showMembers}
      <div class="px-3 pb-4 pt-1 border-t border-ink-800/60">
        <FriendsTable rows={aggregate.friends} me={me.username} />
      </div>
    {/if}
  </div>
{/if}
