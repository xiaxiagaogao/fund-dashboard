<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { RefreshCw } from "lucide-svelte";
  import {
    api,
    type Me,
    type EquityPoint,
    type DayPnL,
    type Position,
    type StatsResponse,
  } from "$lib/api";
  import PageHeader from "$lib/components/PageHeader.svelte";
  import AsyncState from "$lib/components/AsyncState.svelte";
  import DrawdownChart from "$lib/components/DrawdownChart.svelte";
  import CalendarHeatmap from "$lib/components/CalendarHeatmap.svelte";
  import StatsCards from "$lib/components/StatsCards.svelte";
  import OpenPositions from "$lib/components/OpenPositions.svelte";
  import ClosedTrades from "$lib/components/ClosedTrades.svelte";
  import SymbolPnLBars from "$lib/components/SymbolPnLBars.svelte";
  let me: Me | null = null;
  let curve: EquityPoint[] = [],
    daily: DayPnL[] = [],
    openPositions: Position[] = [],
    closedPositions: Position[] = [];
  let stats: StatsResponse | null = null;
  let loading = true,
    refreshing = false;
  let error = "",
    positionError = "",
    chartError = "";
  async function load(initial = true) {
    if (refreshing) return;
    refreshing = true;
    if (initial) loading = true;
    error = "";
    positionError = "";
    chartError = "";
    try {
      me = await api.me();
      if (!me.is_admin) {
        await goto("/");
        return;
      }
      const from = Date.now() - 180 * 86400000;
      const [c, d, o, closed, s] = await Promise.allSettled([
        api.equityCurve(from),
        api.dailyPnl(180),
        api.openPositions(),
        api.closedPositions(200),
        api.stats(500),
      ]);
      if (c.status === "fulfilled") curve = c.value ?? [];
      else chartError = "回撤数据更新失败";
      if (d.status === "fulfilled") daily = d.value ?? [];
      else chartError += `${chartError ? "；" : ""}每日盈亏更新失败`;
      if (o.status === "fulfilled") openPositions = o.value ?? [];
      else positionError = "部分持仓或交易数据更新失败，已有数据保留。";
      if (closed.status === "fulfilled") closedPositions = closed.value ?? [];
      else positionError = "部分持仓或交易数据更新失败，已有数据保留。";
      if (s.status === "fulfilled")
        stats = { ...s.value, by_symbol: s.value.by_symbol ?? [] };
      else positionError = "部分持仓或交易数据更新失败，已有数据保留。";
    } catch (e) {
      error = e instanceof Error ? e.message : "加载失败";
    } finally {
      loading = false;
      refreshing = false;
    }
  }
  onMount(() => load());
</script>

{#if loading || error}<AsyncState {loading} {error} on:retry={() => load()} />
{:else if me?.is_admin}
  <PageHeader title="交易复盘" detail="回撤与每日盈亏 · 最近 180 天"
    ><span class="text-xs text-ink-400"
      >交易统计 · 最近 {stats?.stats.total ?? 0} 笔</span
    ><button
      class="icon-button"
      aria-label="刷新复盘数据"
      disabled={refreshing}
      on:click={() => load(false)}
      ><RefreshCw size={16} class={refreshing ? "refreshing" : ""} /></button
    ></PageHeader
  >
  {#if positionError}<div class="alert mb-5" role="alert">
      {positionError}
    </div>{/if}
  <StatsCards stats={stats?.stats ?? null} window={stats?.window ?? 0} />
  {#if chartError}<div class="alert mt-5" role="alert">{chartError}</div>{/if}
  <div class="review-charts">
    <DrawdownChart points={curve} /><CalendarHeatmap days={daily} />
  </div>
  <div class="data-section"><OpenPositions positions={openPositions} /></div>
  <div class="workspace-split border-t border-ink-700/70">
    <div class="section">
      <ClosedTrades positions={closedPositions} maxRows={8} />
    </div>
    <div class="section"><SymbolPnLBars rows={stats?.by_symbol ?? []} /></div>
  </div>
{/if}

<style>
  .review-charts {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(330px, 0.85fr);
    gap: 28px;
  }
  @media (max-width: 1150px) {
    .review-charts {
      grid-template-columns: minmax(0, 1fr);
      gap: 0;
    }
  }
</style>
