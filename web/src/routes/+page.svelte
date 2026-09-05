<script lang="ts">
  import { onMount } from "svelte";
  import { RefreshCw, Clock3, ArrowUpRight } from "lucide-svelte";
  import {
    api,
    type Me,
    type Summary,
    type Aggregate,
    type EquityPoint,
    type IndexPrices,
    type Allocation,
    type Position,
  } from "$lib/api";
  import {
    fmtUSDT,
    fmtShares,
    fmtSignedUSDT,
    fmtSignedPct,
    fmtDate,
    fmtRelativeTime,
    pnlClass,
  } from "$lib/format";
  import { rangeFromMs, type RangeKey } from "$lib/ranges";
  import PageHeader from "$lib/components/PageHeader.svelte";
  import AsyncState from "$lib/components/AsyncState.svelte";
  import EquityCurve from "$lib/components/EquityCurve.svelte";
  import PositionDonuts from "$lib/components/PositionDonuts.svelte";
  import ClosedTrades from "$lib/components/ClosedTrades.svelte";
  import OpenPositions from "$lib/components/OpenPositions.svelte";
  let me: Me | null = null;
  let summary: Summary | null = null;
  let aggregate: Aggregate | null = null;
  let curve: EquityPoint[] = [];
  let index: IndexPrices = { qqq: [], spy: [] };
  let alloc: Allocation | null = null;
  let closedPositions: Position[] = [];
  let openPositions: Position[] = [];
  let loading = true,
    refreshing = false,
    curveLoading = false;
  let error = "",
    refreshError = "",
    rangeError = "",
    closedError = "",
    openError = "";
  let range: RangeKey = "30d";
  let now = Date.now();
  $: rankedMembers =
    aggregate?.friends.slice().sort((a, b) => b.value_usdt - a.value_usdt) ??
    [];
  $: fundPnl = aggregate?.friends.reduce((s, f) => s + f.pnl_usdt, 0) ?? 0;
  $: fundDeposits =
    aggregate?.friends.reduce((s, f) => s + f.net_deposits, 0) ?? 0;
  $: stale =
    !!summary?.snapshot_at_ms && now - summary.snapshot_at_ms > 90 * 60_000;
  async function fetchWindow(r: RangeKey) {
    const from = rangeFromMs(r);
    const [c, ix] = await Promise.all([
      api.equityCurve(from),
      api.indexPrices(from).catch(() => ({ qqq: [], spy: [] })),
    ]);
    curve = c ?? [];
    index = { qqq: ix.qqq ?? [], spy: ix.spy ?? [] };
  }
  async function onRange(e: CustomEvent<RangeKey>) {
    if (e.detail === range || curveLoading || refreshing) return;
    curveLoading = true;
    rangeError = "";
    try {
      await fetchWindow(e.detail);
      range = e.detail;
    } catch {
      rangeError = "所选区间加载失败，当前仍显示上一次的数据。";
    } finally {
      curveLoading = false;
    }
  }
  async function load(initial = true) {
    if (refreshing) return;
    if (initial) loading = true;
    refreshing = true;
    error = "";
    refreshError = "";
    try {
      const [m, s, a] = await Promise.all([
        api.me(),
        api.mySummary(),
        api.aggregate(),
      ]);
      me = m;
      summary = s;
      aggregate = { ...a, friends: a.friends ?? [] };
      await fetchWindow(range);
      const [al, cp, op] = await Promise.allSettled([
        api.allocation(),
        api.closedPositions(50),
        m.is_admin ? api.openPositions() : Promise.resolve([]),
      ]);
      alloc = al.status === "fulfilled" ? al.value : null;
      closedError = cp.status === "rejected" ? "平仓记录暂不可用" : "";
      openError = op.status === "rejected" ? "持仓明细暂不可用" : "";
      if (cp.status === "fulfilled") closedPositions = cp.value ?? [];
      if (op.status === "fulfilled") openPositions = op.value ?? [];
      now = Date.now();
    } catch (e) {
      if (initial) error = e instanceof Error ? e.message : "加载失败";
      else refreshError = "更新失败，当前显示上一次成功获取的数据。";
    } finally {
      loading = false;
      refreshing = false;
    }
  }
  onMount(() => {
    load();
    const refresh = setInterval(() => load(false), 120_000);
    const clock = setInterval(() => (now = Date.now()), 60_000);
    return () => {
      clearInterval(refresh);
      clearInterval(clock);
    };
  });
</script>

{#if loading || error}<AsyncState {loading} {error} on:retry={() => load()} />
{:else if me && summary && aggregate}
  <PageHeader
    title={me.is_admin ? "基金总览" : "我的资产"}
    detail={me.is_admin ? "基金表现与当前资金敞口" : `${me.name} · 资产与收益`}
  >
    <div
      class="snapshot"
      class:stale
      title={fmtDate(summary.snapshot_at_ms, true)}
    >
      <Clock3 size={13} /><span
        >快照 {summary.snapshot_at_ms
          ? fmtRelativeTime(summary.snapshot_at_ms)
          : "暂无"}{stale ? " · 待更新" : ""}</span
      >
    </div>
    <button
      class="icon-button"
      aria-label="刷新数据"
      disabled={refreshing || curveLoading}
      on:click={() => load(false)}
      ><RefreshCw size={16} class={refreshing ? "refreshing" : ""} /></button
    >
  </PageHeader>
  {#if refreshError}<div class="alert mb-5" role="alert">
      {refreshError}
    </div>{/if}
  <div class="metric-strip">
    <div class="metric">
      <div class="metric-label">
        {me.is_admin ? "基金总权益" : "我的估值"}
        <span class="text-ink-500 ml-1">USDT</span>
      </div>
      <div class="metric-number">
        {fmtUSDT(me.is_admin ? aggregate.latest_equity : summary.value_usdt)}
      </div>
      <div class="metric-note">
        {me.is_admin
          ? `${aggregate.friends.length} 位成员`
          : `${fmtShares(summary.shares, 2)} 份额`}
      </div>
    </div>
    <div class="metric">
      <div class="metric-label">
        {me.is_admin ? "全员净收益" : "我的累计收益"}
        <span class="text-ink-500 ml-1">USDT</span>
      </div>
      <div
        class={"metric-number " +
          pnlClass(me.is_admin ? fundPnl : summary.pnl_usdt)}
      >
        {fmtSignedUSDT(me.is_admin ? fundPnl : summary.pnl_usdt)}
      </div>
      <div class="metric-note">
        净投入 <span class="number"
          >{fmtUSDT(me.is_admin ? fundDeposits : summary.net_deposits, 0)}</span
        >
      </div>
    </div>
    <div class="metric">
      <div class="metric-label">基金单位净值</div>
      <div class="metric-number">{summary.latest_nav.toFixed(4)}</div>
      <div class="metric-note">
        NAV{#if !me.is_admin}
          · 总权益 <span class="number"
            >{fmtUSDT(summary.latest_equity, 0)}</span
          >{/if}
      </div>
    </div>
    <div class="metric">
      <div class="metric-label">{me.is_admin ? "我的估值" : "我的收益率"}</div>
      <div
        class={"metric-number " +
          (!me.is_admin ? pnlClass(summary.pnl_pct) : "")}
      >
        {me.is_admin
          ? fmtUSDT(summary.value_usdt)
          : fmtSignedPct(summary.pnl_pct)}
      </div>
      <div class="metric-note">
        {#if me.is_admin}<span class={pnlClass(summary.pnl_pct)}
            >{fmtSignedPct(summary.pnl_pct)}</span
          >
          · <span class="number">{fmtShares(summary.shares, 0)}</span> 份额
          <div class="mt-1">净收益 <span class={"number " + pnlClass(summary.pnl_usdt)}>{fmtSignedUSDT(summary.pnl_usdt)}</span></div>{:else}累计收益
          / 净投入{/if}
      </div>
    </div>
  </div>
  {#if rangeError}<div role="alert" class="alert mt-4">{rangeError}</div>{/if}
  <div class="workspace-split">
    <div>
      <EquityCurve
        points={curve}
        qqq={index.qqq}
        spy={index.spy}
        {range}
        loading={curveLoading || refreshing}
        on:range={onRange}
      />
    </div>
    <div><PositionDonuts {alloc} /></div>
  </div>
  {#if me.is_admin}<div class="data-section">
      {#if openError}<AsyncState
          error={openError}
          on:retry={() => load(false)}
        />{:else}<OpenPositions positions={openPositions} />{/if}
    </div>{/if}
  <div class="workspace-split border-t border-ink-700/70">
    <div class="section">
      {#if closedError}<AsyncState
          error={closedError}
          on:retry={() => load(false)}
        />{:else}<ClosedTrades positions={closedPositions} maxRows={6} />{/if}
    </div>
    <section class="section" aria-label="成员资产">
      <div class="section-head">
        <h2>成员资产</h2>
        <span class="section-meta">USDT</span>
      </div>
      <div class="member-list">
        {#each rankedMembers as m, i}<div class="member-row">
            <span class="member-rank number"
              >{String(i + 1).padStart(2, "0")}</span
            >
            <div class="min-w-0">
              <div class="member-name">
                {m.name}{#if m.username === me.username}<span
                    class="text-[10px] text-ink-400 ml-2">我</span
                  >{/if}
              </div>
              <div class={"text-[11px] number mt-1 " + pnlClass(m.pnl_pct)}>
                {fmtSignedPct(m.pnl_pct)}
              </div>
            </div>
            <span class="number text-xs ml-auto">{fmtUSDT(m.value_usdt)}</span>
          </div>{/each}
      </div>
      {#if me.is_admin}<a href="/admin" class="member-link"
          >成员与资金管理 <ArrowUpRight size={14} /></a
        >{/if}
    </section>
  </div>
{/if}

<style>
  .snapshot {
    color: var(--lo);
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 11px;
  }
  .snapshot.stale {
    color: var(--benchmark-b);
  }
  .member-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 0;
    border-bottom: 1px solid var(--panel2);
  }
  .member-row:first-child {
    padding-top: 4px;
  }
  .member-row:last-child {
    border-bottom: 0;
  }
  .member-rank {
    color: var(--lo);
    font-size: 10px;
  }
  .member-name {
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  .member-link {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 22px;
    color: var(--mid);
    font-size: 11px;
    padding: 8px 0;
  }
  .member-link:hover {
    color: var(--pos);
  }
</style>
