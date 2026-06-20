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
    type Position
  } from '$lib/api';
  import { fmtUSDT, fmtShares, fmtSignedUSDT, fmtSignedPct } from '$lib/format';
  import EquityCurve from '$lib/components/EquityCurve.svelte';
  import PositionDonuts from '$lib/components/PositionDonuts.svelte';
  import ClosedTrades from '$lib/components/ClosedTrades.svelte';

  let me: Me | null = null;
  let summary: Summary | null = null;
  let aggregate: Aggregate | null = null;
  let curve: EquityPoint[] = [];
  let events: CashEvent[] = [];
  let alloc: Allocation | null = null;
  let closedPositions: Position[] = [];
  let positionsAvailable = true;
  let loading = true;
  let error = '';

  // "我的" curve: my shares as-of each snapshot × NAV then; invest line = my
  // cumulative net deposits as-of each snapshot. events come ASC by occurred_at.
  $: mineValues = curve.map((pt) => {
    let shares = 0;
    for (const e of events) {
      if (e.occurred_at <= pt.taken_at) shares += e.shares_delta;
      else break;
    }
    return shares * pt.nav;
  });
  $: investValues = curve.map((pt) => {
    let net = 0;
    for (const e of events) {
      if (e.occurred_at <= pt.taken_at) net += e.type === 'deposit' ? e.amount_usdt : -e.amount_usdt;
      else break;
    }
    return net;
  });

  $: rankedMembers = aggregate
    ? aggregate.friends.slice().sort((a, b) => b.value_usdt - a.value_usdt)
    : [];

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
      [alloc, closedPositions] = await Promise.all([api.allocation(), api.closedPositions(50)]);
      positionsAvailable = true;
    } catch (e) {
      positionsAvailable = false;
    }
    loading = false;
  }

  async function refreshLive() {
    try {
      const [s, a, al] = await Promise.all([api.mySummary(), api.aggregate(), api.allocation()]);
      summary = s;
      aggregate = a;
      alloc = al;
    } catch {
      // keep stale data on transient failures
    }
  }

  onMount(() => {
    load();
    const h = setInterval(refreshLive, 2 * 60 * 1000);
    return () => clearInterval(h);
  });
</script>

{#if loading}
  <div class="text-ink-400 text-sm">加载中…</div>
{:else if error}
  <div class="card p-6 text-loss-400">{error}</div>
{:else if me && summary && aggregate}
  <div class="hidden md:block mb-6">
    <div class="text-[11px] text-ink-500 tracking-[0.16em] uppercase mb-1.5">朋友视图 · 我的份额</div>
    <h1 class="text-[25px] font-extrabold tracking-tight m-0">我的看板</h1>
  </div>

  <div class="flex flex-col gap-3.5 md:gap-4">
    <!-- Hero + two mini cards -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3 md:gap-4">
      <div class="col-span-2 relative overflow-hidden rounded-2xl border border-white/[0.08] p-5 md:p-6"
        style="background:linear-gradient(155deg,oklch(0.22 0.012 168 / 0.55),var(--panel))">
        <div class="absolute inset-0 pointer-events-none" style="background:radial-gradient(360px 180px at 100% 0%, oklch(0.80 0.115 168 / 0.11), transparent 60%)"></div>
        <div class="label">我的估值</div>
        <div class="flex items-baseline gap-2.5 mt-2.5">
          <div class="font-mono text-[34px] md:text-[42px] font-semibold tracking-tight leading-none">{fmtUSDT(summary.value_usdt, 2)}</div>
          <div class="text-xs md:text-[13px] text-ink-300 font-semibold">USDT</div>
        </div>
        <div class="flex items-center gap-3 mt-3.5">
          <span class={summary.pnl_pct >= 0 ? 'pill-pos' : 'pill-neg'}>{fmtSignedPct(summary.pnl_pct)}</span>
          <span class="font-mono text-xs text-ink-300">{fmtShares(summary.shares, 0)} 份额</span>
        </div>
      </div>

      <div class="card p-4 md:p-5 flex flex-col justify-center">
        <div class="label">总收益 PnL</div>
        <div class={'font-mono text-xl md:text-[28px] font-semibold mt-2 tracking-tight ' + (summary.pnl_usdt >= 0 ? 'pos' : 'neg')}>
          {fmtSignedUSDT(summary.pnl_usdt, 0)}
        </div>
        <div class="text-[10px] md:text-[11px] text-ink-500 mt-1.5 font-mono">累计投入 {fmtUSDT(summary.net_deposits, 0)}</div>
      </div>

      <div class="card p-4 md:p-5 flex flex-col justify-center">
        <div class="label">基金净值 NAV</div>
        <div class="font-mono text-xl md:text-[28px] font-semibold mt-2 tracking-tight">{summary.latest_nav.toFixed(4)}</div>
        <div class="text-[10px] md:text-[11px] text-ink-500 mt-1.5 font-mono">基金总权益 {fmtUSDT(summary.latest_equity, 0)}</div>
      </div>
    </div>

    <!-- Equity curve -->
    <EquityCurve points={curve} {mineValues} {investValues} height={220} />

    <!-- Current holdings -->
    {#if positionsAvailable}
      <PositionDonuts {alloc} loading={!alloc} />
    {/if}

    <!-- Members -->
    <div class="card p-4 sm:p-5">
      <div class="flex items-baseline justify-between mb-3">
        <div class="text-[13px] font-bold">成员对比</div>
        <div class="label">按估值</div>
      </div>
      <div class="flex flex-col gap-0.5">
        {#each rankedMembers as m, i}
          <div class={'flex items-center gap-2.5 py-2.5 px-1.5 rounded-[9px] ' + (m.username === me.username ? 'bg-accent-500/[0.05]' : '')}>
            <div class="w-4 font-mono text-xs text-ink-500 flex-none">{i + 1}</div>
            <span class="text-[13px] font-semibold truncate">{m.name}</span>
            {#if m.username === me.username}
              <span class="text-[9px] text-accent-400 border border-accent-500/30 rounded px-1 flex-none">我</span>
            {/if}
            <div class="ml-auto font-mono text-[13px] font-semibold">{fmtUSDT(m.value_usdt, 0)}</div>
            <div class="w-[58px] flex-none text-right">
              <span class={m.pnl_pct >= 0 ? 'pill-pos' : 'pill-neg'}>{fmtSignedPct(m.pnl_pct)}</span>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Trade transparency: read-only recent closes -->
    {#if positionsAvailable && closedPositions.length > 0}
      <ClosedTrades positions={closedPositions} limit={8} />
    {/if}
  </div>
{/if}
