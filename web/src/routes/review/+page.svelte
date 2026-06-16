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
  import EquityCurve from '$lib/components/EquityCurve.svelte';
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
      // Full history (snapshots only go back ~1 month; ask for plenty).
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
  <div class="mb-5 flex items-baseline justify-between">
    <h2 class="text-xl font-semibold tracking-tight">复盘</h2>
    <div class="text-xs text-ink-400">仅自己可见 · 数据源 Binance + fund.db</div>
  </div>

  <!-- ① 资金 & 回撤 -->
  <div class="mb-5">
    <EquityCurve points={curve} variant="fund" height={240} />
  </div>
  <div class="mb-6">
    <DrawdownChart points={curve} height={150} />
  </div>

  <!-- ④ 每日盈亏日历 -->
  <div class="mb-6">
    <CalendarHeatmap days={daily} />
  </div>

  {#if positionsAvailable}
    <!-- ② 交易分布 -->
    <div class="mb-6">
      <StatsCards stats={stats?.stats ?? null} window={stats?.window ?? 0} />
    </div>

    <div class="mb-6">
      <OpenPositions positions={openPositions} />
    </div>

    <!-- ③ 标的 & 平仓明细 -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
      <div class="lg:col-span-2">
        <ClosedTrades positions={closedPositions} limit={40} />
      </div>
      <div>
        <SymbolPnLBars rows={stats?.by_symbol ?? []} />
      </div>
    </div>
  {:else}
    <div class="card p-6 text-ink-400 text-sm">
      持仓/交易统计暂不可用（Binance 客户端未配置）。回撤与日历来自 fund.db，照常显示。
    </div>
  {/if}
{/if}
