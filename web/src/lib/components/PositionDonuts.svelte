<script lang="ts">
  import { Wallet, Clock3 } from "lucide-svelte";
  import type { Allocation } from "$lib/api";
  import { fmtUSDT, fmtPct, fmtDate } from "$lib/format";
  export let alloc: Allocation | null = null;
  export let loading = false;
  const colors = [
    "var(--pos)",
    "var(--benchmark-a)",
    "var(--benchmark-b)",
    "#bab0d3",
    "#c2b8a6",
  ];
  $: usage =
    alloc && alloc.equity > 0 ? alloc.margin_used / alloc.equity : null;
  $: holdings = [...(alloc?.positions ?? [])].sort(
    (a, b) => b.notional - a.notional,
  );
</script>

<section class="section" aria-label="资金配置">
  <div class="section-head">
    <h2>资金配置</h2>
    <Wallet size={15} class="text-ink-400" />
  </div>
  {#if loading}<div
      class="skeleton h-56"
      aria-label="正在加载资金配置"
      role="status"
    ></div>
  {:else if !alloc}<div class="empty-state">资金配置暂不可用</div>
  {:else}
    <div class="capital-values">
      <div>
        <div class="label">已用保证金</div>
        <div class="number capital-value">
          {fmtUSDT(alloc.margin_used, 0)}<span> USDT</span>
        </div>
      </div>
      <div class="text-right">
        <div class="label">占权益</div>
        <div class="number capital-value">
          {usage === null ? "—" : fmtPct(usage, 1)}
        </div>
      </div>
    </div>
    <div
      class="capital-track"
      role="img"
      aria-label={`保证金占用 ${usage === null ? "未知" : fmtPct(usage, 1)}`}
    >
      <div style:width={`${Math.max(0, Math.min(1, usage ?? 0)) * 100}%`}></div>
    </div>
    <div class="capital-free">
      <span>闲置资金</span><span class="number"
        >{fmtUSDT(alloc.free_cash, 0)}
        <span class="text-ink-500">USDT</span></span
      >
    </div>
    <div class="exposure">
      <div>
        <span class="label">全仓杠杆</span><strong class="number"
          >{alloc.leverage.toFixed(2)}<span>×</span></strong
        >
      </div>
      <div>
        <span class="label">名义敞口</span><strong class="number"
          >{fmtUSDT(alloc.notional, 0)}</strong
        >
      </div>
    </div>
    <div class="allocation-label">
      名义持仓分布 <span>{holdings.length} 个</span>
    </div>
    {#if holdings.length === 0}<div class="empty-state min-h-24">当前空仓</div>
    {:else}<div class="allocation-rows">
        {#each holdings as p, i}<div class="allocation-row">
            <div class="allocation-readout">
              <span class="symbol"
                >{p.symbol.replace(/USDT$/, "")}<span class="side"
                  >{p.side === "LONG"
                    ? "多"
                    : p.side === "SHORT"
                      ? "空"
                      : p.side}</span
                ></span
              ><span class="number">{fmtPct(p.pct, 1)}</span>
            </div>
            <div class="allocation-track">
              <div
                style:width={`${Math.max(0, Math.min(1, p.pct)) * 100}%`}
                style:background={p.side === "SHORT"
                  ? "var(--neg)"
                  : colors[i % colors.length]}
              ></div>
            </div>
          </div>{/each}
      </div>{/if}
    <div class="allocation-time">
      <Clock3 size={11} /><span
        >{alloc.update_time
          ? fmtDate(alloc.update_time, true)
          : "更新时间未知"}</span
      >
    </div>
  {/if}
</section>

<style>
  .capital-values {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
  .capital-value {
    font-size: 18px;
    margin-top: 8px;
  }
  .capital-value span {
    font-size: 10px;
    color: var(--lo);
  }
  .capital-track {
    height: 5px;
    background: var(--panel2);
    margin: 18px 0 12px;
    overflow: hidden;
    border-radius: 1px;
  }
  .capital-track > div {
    height: 100%;
    background: var(--pos);
  }
  .capital-free {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 11px;
    color: var(--mid);
  }
  .exposure {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px;
    border-block: 1px solid var(--line);
    padding: 17px 0;
    margin-top: 20px;
  }
  .exposure strong {
    font-size: 18px;
    display: block;
    font-weight: 500;
    margin-top: 8px;
  }
  .exposure strong span {
    margin-left: 3px;
    color: var(--lo);
  }
  .allocation-label {
    display: flex;
    justify-content: space-between;
    color: var(--mid);
    font-size: 11px;
    margin-top: 19px;
  }
  .allocation-label span {
    color: var(--lo);
  }
  .allocation-rows {
    display: grid;
    gap: 13px;
    margin-top: 16px;
    max-height: 210px;
    overflow: auto;
  }
  .allocation-readout {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 11px;
    gap: 12px;
  }
  .symbol {
    font-size: 12px;
    font-weight: 600;
  }
  .side {
    font-size: 10px;
    color: var(--lo);
    margin-left: 8px;
    font-weight: 400;
  }
  .allocation-track {
    height: 3px;
    background: var(--panel2);
    margin-top: 6px;
  }
  .allocation-track > div {
    height: 100%;
  }
  .allocation-time {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--lo);
    font-size: 10px;
    margin-top: 20px;
  }
</style>
