<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import {
    api,
    type Me,
    type EquityPoint,
    type DayPnL,
    type Position,
    type StatsResponse
  } from '$lib/api';
  import DrawdownChart from '$lib/components/DrawdownChart.svelte';
  import CalendarHeatmap from '$lib/components/CalendarHeatmap.svelte';
  import StatsCards from '$lib/components/StatsCards.svelte';
  import OpenPositions from '$lib/components/OpenPositions.svelte';
  import ClosedTrades from '$lib/components/ClosedTrades.svelte';
  import SymbolPnLBars from '$lib/components/SymbolPnLBars.svelte';

  let me: Me | null = null;
  let curve: EquityPoint[] = [];
  let daily: DayPnL[] = [];
  let openPositions: Position[] = [];
  let closedPositions: Position[] = [];
  let stats: StatsResponse | null = null;
  let positionsAvailable = true;
  let loading = true;
  let error = '';

  async function load() {
    try {
      me = await api.me();
      if (!me.is_admin) {
        goto('/');
        return;
      }
      const from = Date.now() - 180 * 24 * 60 * 60 * 1000;
      [curve, daily] = await Promise.all([api.equityCurve(from), api.dailyPnl(180)]);
    } catch (e) {
      error = e instanceof Error ? e.message : '加载失败';
      loading = false;
      return;
    }
    try {
      [openPositions, closedPositions, stats] = await Promise.all([
        api.openPositions(),
        api.closedPositions(200),
        api.stats(500)
      ]);
      positionsAvailable = true;
    } catch (e) {
      positionsAvailable = false;
    }
    loading = false;
  }

  onMount(load);
</script>

{#if loading}
  <div class="text-ink-400 text-sm">加载中…</div>
{:else if error}
  <div class="card p-6 text-loss-400">{error}</div>
{:else if me}
  <div class="hidden md:block mb-6">
    <div class="text-[11px] text-ink-500 tracking-[0.16em] uppercase mb-1.5">复盘视图 · 仅管理</div>
    <h1 class="text-[25px] font-extrabold tracking-tight m-0">复盘分析</h1>
  </div>
  <div class="md:hidden mb-4">
    <h2 class="text-lg font-bold tracking-tight">复盘分析</h2>
  </div>

  <div class="flex flex-col gap-3.5 md:gap-4">
    {#if positionsAvailable}
      <StatsCards stats={stats?.stats ?? null} window={stats?.window ?? 0} />
    {/if}

    <DrawdownChart points={curve} height={130} />

    <CalendarHeatmap days={daily} />

    {#if positionsAvailable}
      <OpenPositions positions={openPositions} />

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-3.5 md:gap-4">
        <div class="lg:col-span-2"><ClosedTrades positions={closedPositions} limit={40} /></div>
        <div><SymbolPnLBars rows={stats?.by_symbol ?? []} /></div>
      </div>
    {:else}
      <div class="card p-6 text-ink-400 text-sm">
        持仓/交易统计暂不可用（Binance 客户端未配置）。回撤与日历来自 fund.db，照常显示。
      </div>
    {/if}
  </div>
{/if}
