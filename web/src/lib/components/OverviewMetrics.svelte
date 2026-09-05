<script lang="ts">
  import {
    WalletCards,
    TrendingUp,
    ChartNoAxesCombined,
    UserRound,
    UsersRound,
    Layers3,
  } from "lucide-svelte";
  import { line } from "d3-shape";
  import type { Me, Summary, Aggregate, EquityPoint } from "$lib/api";
  import {
    fmtUSDT,
    fmtShares,
    fmtSignedUSDT,
    fmtSignedPct,
    pnlClass,
  } from "$lib/format";
  import { RANGES, type RangeKey } from "$lib/ranges";

  export let me: Me;
  export let summary: Summary;
  export let aggregate: Aggregate;
  export let curve: EquityPoint[] = [];
  export let range: RangeKey = "30d";

  $: pnl = me.is_admin
    ? aggregate.friends.reduce((sum, member) => sum + member.pnl_usdt, 0)
    : summary.pnl_usdt;
  $: deposits = me.is_admin
    ? aggregate.friends.reduce((sum, member) => sum + member.net_deposits, 0)
    : summary.net_deposits;
  $: history = curve
    .filter(
      (point) => Number.isFinite(point.nav) && Number.isFinite(point.taken_at),
    )
    .slice()
    .sort((a, b) => a.taken_at - b.taken_at);
  $: low = Math.min(...history.map((point) => point.nav));
  $: high = Math.max(...history.map((point) => point.nav));
  $: start = history[0]?.taken_at ?? 0;
  $: duration = (history[history.length - 1]?.taken_at ?? 0) - start;
  $: navLine =
    history.length > 1
      ? line<EquityPoint>()
          .x((point) => 2 + ((point.taken_at - start) / (duration || 1)) * 80)
          .y((point) =>
            high === low ? 15 : 27 - ((point.nav - low) / (high - low)) * 24,
          )(history)
      : null;
  $: rangeLabel = RANGES.find((item) => item.key === range)?.label ?? "";
</script>

<section
  class="overview-metrics"
  aria-label={me.is_admin ? "基金核心指标" : "个人资产指标"}
>
  <article class="overview-card overview-primary">
    <div class="overview-heading">
      <h2>{me.is_admin ? "基金总权益" : "我的估值"}</h2>
      <WalletCards size={19} strokeWidth={1.6} />
    </div>
    <div class="overview-value number">
      {fmtUSDT(me.is_admin ? aggregate.latest_equity : summary.value_usdt)}
    </div>
    <div class="overview-footer">
      <span class="unit">USDT</span><span class="footer-detail"
        >{#if me.is_admin}<UsersRound size={12} />{aggregate.friends.length} 位成员{:else}<Layers3
            size={12}
          />{fmtShares(summary.shares, 2)} 份额{/if}</span
      >
    </div>
  </article>
  <article
    class="overview-card overview-pnl"
    class:negative={pnl < 0}
    class:positive={pnl > 0}
  >
    <div class="overview-heading">
      <h2>{me.is_admin ? "全员净收益" : "我的累计收益"}</h2>
      <TrendingUp size={19} strokeWidth={1.6} />
    </div>
    <div class={"overview-value number " + pnlClass(pnl)}>
      {fmtSignedUSDT(pnl)}
    </div>
    <div class="overview-footer">
      <span class="unit">USDT</span><span class="footer-detail"
        >净投入 <span class="number">{fmtUSDT(deposits, 0)}</span></span
      >
    </div>
  </article>
  <article class="overview-card overview-nav">
    <div class="overview-heading">
      <h2>基金单位净值</h2>
      <ChartNoAxesCombined size={19} strokeWidth={1.6} />
    </div>
    <div class="overview-value number">{summary.latest_nav.toFixed(4)}</div>
    <div class="overview-footer nav-footer">
      <span class="unit">NAV · {rangeLabel}</span>
      {#if navLine}<svg
          class="nav-history"
          viewBox="0 0 84 30"
          role="img"
          aria-label={`${rangeLabel}单位净值走势`}
          ><path
            d={navLine}
            fill="none"
            stroke="var(--benchmark-b)"
            stroke-width="1.7"
            stroke-linejoin="round"
            stroke-linecap="round"
          /></svg
        >{/if}
    </div>
    {#if !me.is_admin}<div class="overview-subnote">
        总权益 <span class="number">{fmtUSDT(summary.latest_equity, 0)}</span> USDT
      </div>{/if}
  </article>
  <article class="overview-card overview-personal">
    <div class="overview-heading">
      <h2>{me.is_admin ? "我的估值" : "我的收益率"}</h2>
      <UserRound size={19} strokeWidth={1.6} />
    </div>
    <div
      class={"overview-value number " +
        (!me.is_admin ? pnlClass(summary.pnl_pct) : "")}
    >
      {me.is_admin
        ? fmtUSDT(summary.value_usdt)
        : fmtSignedPct(summary.pnl_pct)}
    </div>
    <div class="overview-footer personal-footer">
      {#if me.is_admin}<span
          class={"return-chip number " + pnlClass(summary.pnl_pct)}
          >{fmtSignedPct(summary.pnl_pct)}</span
        ><span class="footer-detail"
          ><span class="number">{fmtShares(summary.shares, 0)}</span> 份额</span
        >{:else}<span>累计收益 / 净投入</span>{/if}
    </div>
    {#if me.is_admin}<div class="overview-subnote">
        USDT · 净收益 <span class={"number " + pnlClass(summary.pnl_usdt)}
          >{fmtSignedUSDT(summary.pnl_usdt)}</span
        >
      </div>{/if}
  </article>
</section>

<style>
  .overview-metrics {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 12px;
    padding-bottom: 24px;
    border-bottom: 1px solid var(--line);
  }
  .overview-card {
    min-width: 0;
    min-height: 166px;
    padding: 18px 18px 14px;
    border: 1px solid var(--line);
    border-radius: 8px;
    background: var(--panel);
  }
  .overview-primary {
    background: #1a2421;
    border-color: #354a41;
  }
  .overview-pnl.negative {
    background: #211b1c;
    border-color: #433334;
  }
  .overview-pnl.positive {
    background: #1c2421;
    border-color: #35433b;
  }
  .overview-personal {
    background: #1b2026;
    border-color: #343e49;
  }
  .overview-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    color: var(--lo);
  }
  .overview-heading h2 {
    font-size: 12px;
    font-weight: 500;
    color: var(--mid);
  }
  .overview-primary .overview-heading {
    color: var(--pos);
  }
  .overview-nav .overview-heading {
    color: var(--benchmark-b);
  }
  .overview-personal .overview-heading {
    color: var(--benchmark-a);
  }
  .overview-heading :global(svg) {
    flex: none;
  }
  .overview-value {
    font-size: 25px;
    font-weight: 500;
    line-height: 1.25;
    margin-top: 17px;
    overflow-wrap: anywhere;
  }
  .overview-primary .overview-value {
    color: var(--pos-bright);
    font-weight: 600;
  }
  .overview-footer {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 6px;
    min-height: 24px;
    margin-top: 12px;
    color: var(--mid);
    font-size: 11px;
  }
  .unit {
    color: var(--lo);
    font-size: 10px;
    white-space: nowrap;
  }
  .overview-primary .unit,
  .overview-primary .footer-detail {
    color: #a8c2b6;
  }
  .footer-detail {
    display: inline-flex;
    align-items: center;
    gap: 5px;
  }
  .overview-subnote {
    color: var(--lo);
    font-size: 10px;
    margin-top: 5px;
    overflow-wrap: anywhere;
  }
  .nav-history {
    width: 72px;
    height: 26px;
    flex: none;
  }
  .return-chip {
    padding: 2px 5px;
    border-radius: 4px;
    background: var(--panel2);
    font-size: 11px;
  }
  .return-chip.pos {
    background: var(--pos-tint);
  }
  .return-chip.neg {
    background: var(--neg-tint);
  }
  @media (max-width: 1150px) {
    .overview-card {
      padding: 16px 13px 13px;
    }
    .overview-value {
      font-size: 23px;
    }
  }
  @media (max-width: 1023px) {
    .overview-metrics {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  @media (max-width: 600px) {
    .overview-metrics {
      gap: 10px;
      padding-bottom: 20px;
    }
    .overview-card {
      min-height: 156px;
      padding: 15px 13px 12px;
    }
    .overview-value {
      font-size: 22px;
      margin-top: 16px;
    }
    .overview-heading h2 {
      font-size: 11px;
    }
    .overview-footer {
      font-size: 10px;
      gap: 4px;
    }
    .nav-history {
      width: 58px;
    }
  }
</style>
