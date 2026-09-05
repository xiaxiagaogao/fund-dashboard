<script lang="ts">
  import { Wallet, Clock3 } from "lucide-svelte";
  import type { Allocation } from "$lib/api";
  import { fmtUSDT, fmtPct, fmtDate } from "$lib/format";
  import AllocationDonut from "./AllocationDonut.svelte";
  export let alloc: Allocation | null = null;
  export let loading = false;
  const colors = [
    "var(--pos)",
    "var(--benchmark-a)",
    "var(--benchmark-b)",
    "#b8a4cf",
    "#d795aa",
    "#71b7ad",
    "#bfc98d",
    "#daab86",
    "#9caccb",
    "#b88f86",
    "#a7b7a1",
    "#82b5c7",
    "#c3becf",
  ];
  let capitalSelected: string | null = null;
  let capitalHovered: string | null = null;
  let holdingSelected: string | null = null;
  let holdingHovered: string | null = null;
  $: usage =
    alloc && alloc.equity > 0 ? alloc.margin_used / alloc.equity : null;
  $: holdings = [...(alloc?.positions ?? [])].sort(
    (a, b) => b.notional - a.notional,
  );
  $: capitalItems = alloc
    ? [
        {
          id: "margin",
          label: `已用保证金 ${fmtUSDT(alloc.margin_used)} USDT`,
          value: Math.max(0, alloc.margin_used),
          color: "var(--pos)",
        },
        {
          id: "cash",
          label: `闲置资金 ${fmtUSDT(alloc.free_cash)} USDT`,
          value: Math.max(0, alloc.free_cash),
          color: "#4a565b",
        },
      ]
    : [];
  $: holdingItems = holdings.map((position, index) => ({
    ...position,
    id: `${position.symbol}:${position.side}`,
    name: position.symbol.replace(/USDT$/, ""),
    label: `${position.symbol.replace(/USDT$/, "")} ${position.side === "LONG" ? "多" : "空"}，占比 ${fmtPct(position.pct, 1)}，${fmtUSDT(position.notional)} USDT`,
    value: position.notional,
    color: colors[index % colors.length],
  }));
  $: capitalActive = capitalHovered ?? capitalSelected;
  $: holdingActive = holdingHovered ?? holdingSelected;
  $: activeCapital = capitalItems.find((item) => item.id === capitalActive);
  $: activeHolding = holdingItems.find((item) => item.id === holdingActive);
  $: if (
    holdingSelected &&
    !holdingItems.some((item) => item.id === holdingSelected)
  )
    holdingSelected = null;
  function selectCapital(id: string) {
    capitalSelected = capitalSelected === id ? null : id;
  }
  function selectHolding(id: string) {
    holdingSelected = holdingSelected === id ? null : id;
  }
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
    <div class="allocation-charts">
      <div class="chart-column">
        <h3>保证金占用</h3>
        <AllocationDonut
          items={alloc.equity > 0 ? capitalItems : []}
          label="保证金与闲置资金分布"
          activeId={capitalActive}
          selectedId={capitalSelected}
          centerLabel={activeCapital
            ? activeCapital.id === "margin"
              ? "已用保证金"
              : "闲置资金"
            : "占总权益"}
          centerValue={activeCapital
            ? fmtUSDT(
                activeCapital.id === "cash"
                  ? alloc.free_cash
                  : alloc.margin_used,
                0,
              )
            : usage === null
              ? "--"
              : fmtPct(usage, 1)}
          centerDetail={activeCapital ? "USDT" : "保证金"}
          on:inspect={(event) => (capitalHovered = event.detail)}
          on:select={(event) => selectCapital(event.detail)}
        />
      </div>
      <div class="chart-column">
        <h3>名义持仓分布</h3>
        <AllocationDonut
          items={holdingItems}
          label="各标的名义持仓占比"
          activeId={holdingActive}
          selectedId={holdingSelected}
          centerLabel={activeHolding ? activeHolding.name : "名义敞口"}
          centerValue={activeHolding
            ? fmtPct(activeHolding.pct, 1)
            : holdings.length
              ? fmtUSDT(alloc.notional, 0)
              : "空仓"}
          centerDetail={activeHolding
            ? `${fmtUSDT(activeHolding.notional, 0)} USDT`
            : holdings.length
              ? "USDT"
              : ""}
          on:inspect={(event) => (holdingHovered = event.detail)}
          on:select={(event) => selectHolding(event.detail)}
        />
      </div>
    </div>
    <div class="capital-legend">
      {#each capitalItems as item}
        <button
          class="capital-row"
          class:highlighted={capitalActive === item.id}
          aria-pressed={capitalSelected === item.id}
          on:pointerenter={() => (capitalHovered = item.id)}
          on:pointerleave={() => (capitalHovered = null)}
          on:focus={() => (capitalHovered = item.id)}
          on:blur={() => (capitalHovered = null)}
          on:click={() => selectCapital(item.id)}
        >
          <span class="legend-name"
            ><span class="swatch" style:background={item.color}
            ></span>{item.id === "margin" ? "已用保证金" : "闲置资金"}</span
          >
          <span class:neg={item.id === "cash" && alloc.free_cash < 0}
            ><span class="number"
              >{fmtUSDT(
                item.id === "cash" ? alloc.free_cash : alloc.margin_used,
                0,
              )}</span
            > <span class="unit">USDT</span></span
          >
        </button>
      {/each}
    </div>
    {#if alloc.free_cash < 0}<p class="capital-warning">
        保证金超出权益 {fmtUSDT(-alloc.free_cash)} USDT
      </p>{/if}
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
      <h3>持仓构成</h3>
      <span>{holdings.length} 个标的</span>
    </div>
    {#if holdings.length === 0}<div class="empty-state min-h-24">当前空仓</div>
    {:else}<div class="allocation-rows">
        {#each holdingItems as item}
          <button
            class="holding-row"
            class:highlighted={holdingActive === item.id}
            aria-label={item.label}
            aria-pressed={holdingSelected === item.id}
            on:pointerenter={() => (holdingHovered = item.id)}
            on:pointerleave={() => (holdingHovered = null)}
            on:focus={() => (holdingHovered = item.id)}
            on:blur={() => (holdingHovered = null)}
            on:click={() => selectHolding(item.id)}
          >
            <span class="legend-name"
              ><span class="swatch" style:background={item.color}></span><span
                class="symbol">{item.name}</span
              ><span class="side" class:neg={item.side === "SHORT"}
                >{item.side === "LONG"
                  ? "多"
                  : item.side === "SHORT"
                    ? "空"
                    : item.side}</span
              ></span
            >
            <span class="number">{fmtPct(item.pct, 1)}</span>
          </button>
        {/each}
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
  .allocation-charts {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    margin: 0 -4px 8px;
  }
  .chart-column {
    min-width: 0;
  }
  .chart-column h3 {
    font-size: 11px;
    color: var(--mid);
    text-align: center;
    font-weight: 500;
    margin-bottom: 8px;
  }
  .capital-legend {
    display: grid;
    gap: 2px;
  }
  .capital-row,
  .holding-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    min-height: 36px;
    gap: 8px;
    padding: 7px 5px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    font-size: 11px;
    color: var(--mid);
    text-align: left;
    transition: background 150ms;
  }
  .capital-row:hover,
  .holding-row:hover,
  .highlighted {
    background: var(--panel2);
  }
  .legend-name {
    display: flex;
    align-items: center;
    gap: 7px;
    min-width: 0;
  }
  .swatch {
    display: inline-block;
    width: 7px;
    height: 7px;
    flex: none;
    border-radius: 2px;
  }
  .unit {
    font-size: 9px;
    color: var(--lo);
  }
  .exposure {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px;
    border-block: 1px solid var(--line);
    padding: 16px 0;
    margin-top: 14px;
  }
  .exposure strong {
    display: block;
    font-size: 18px;
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
    align-items: center;
    gap: 8px;
    margin-top: 18px;
    color: var(--lo);
    font-size: 10px;
  }
  .allocation-label h3 {
    color: var(--mid);
    font-weight: 500;
    font-size: 11px;
  }
  .allocation-rows {
    max-height: 216px;
    overflow: auto;
    margin-top: 9px;
    padding: 3px;
    margin-inline: -3px;
  }
  .symbol {
    font-size: 11px;
    font-weight: 600;
    overflow-wrap: anywhere;
  }
  .side {
    font-size: 9px;
    color: var(--lo);
  }
  .allocation-time {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--lo);
    font-size: 10px;
    margin-top: 18px;
  }
  .capital-warning {
    color: var(--neg);
    font-size: 11px;
    margin-top: 8px;
  }
  @media (max-width: 900px) {
    .allocation-charts {
      max-width: 440px;
      margin-inline: auto;
      gap: 24px;
    }
    .capital-row,
    .holding-row {
      min-height: 42px;
      padding-inline: 8px;
      font-size: 12px;
    }
    .allocation-rows {
      max-height: 260px;
    }
  }
</style>
