<script lang="ts">
  import { Search, ChevronLeft, ChevronRight } from "lucide-svelte";
  import type { Position } from "$lib/api";
  import {
    fmtDuration,
    fmtSignedUSDT,
    fmtUSDT,
    fmtDate,
    pnlClass,
  } from "$lib/format";
  export let positions: Position[] = [];
  export let maxRows = 8;
  let query = "";
  let currentPage = 0;
  const net = (p: Position) => p.realized_pnl - (p.commission ?? 0);
  $: totalFees = positions.reduce((s, p) => s + (p.commission ?? 0), 0);
  $: totalNet = positions.reduce((s, p) => s + net(p), 0);
  $: filtered = positions.filter((p) =>
    p.symbol.toLowerCase().includes(query.trim().toLowerCase()),
  );
  $: pageCount = Math.max(1, Math.ceil(filtered.length / maxRows));
  $: safePage = Math.min(currentPage, pageCount - 1);
  $: visible = filtered.slice(safePage * maxRows, (safePage + 1) * maxRows);
</script>

<section aria-label="平仓记录">
  <div class="section-head">
    <h2>平仓记录</h2>
    <label class="trade-search"
      ><Search size={13} /><input
        aria-label="搜索平仓标的"
        placeholder="搜索标的"
        bind:value={query}
        on:input={() => (currentPage = 0)}
      /></label
    >
  </div>
  <div class="trade-totals">
    <span>最近 {positions.length} 笔</span><span
      >净收益 <span class={"number " + pnlClass(totalNet)}
        >{fmtSignedUSDT(totalNet)}</span
      ></span
    ><span
      >手续费 <span class="number text-ink-300">{fmtUSDT(totalFees)}</span
      ></span
    >
  </div>
  {#if filtered.length === 0}<div class="empty-state">
      {query ? "没有匹配的标的" : "还没有已平仓的交易"}
    </div>
  {:else}<div class="trade-list">
      {#each visible as p}<div class="trade-row">
          <div class="trade-symbol">
            <div class="flex items-center gap-2">
              <strong>{p.symbol.replace(/USDT$/, "")}</strong><span
                class={p.side === "LONG" || p.side === "BUY"
                  ? "pill-pos"
                  : "pill-neg"}
                >{p.side === "LONG" || p.side === "BUY" ? "多" : "空"}</span
              >
            </div>
            <div class="trade-secondary number">
              {fmtDate(p.exit_time ?? 0).slice(5)}
            </div>
          </div>
          <div class="trade-price">
            <div class="number">
              {fmtUSDT(p.entry_price, p.entry_price < 10 ? 3 : 1)}
              <span class="text-ink-500">→</span>
              {fmtUSDT(p.exit_price ?? 0, (p.exit_price ?? 0) < 10 ? 3 : 1)}
            </div>
            <div class="trade-secondary">
              {fmtDuration((p.exit_time ?? 0) - p.entry_time)}
            </div>
          </div>
          <div class="trade-pnl">
            <div class={"number " + pnlClass(net(p))}>
              {fmtSignedUSDT(net(p))}
            </div>
            <div class="trade-secondary">
              费 <span class="number">{fmtUSDT(p.commission ?? 0)}</span>
            </div>
          </div>
        </div>{/each}
    </div>
    <div class="pagination">
      <span
        >{safePage * maxRows + 1}–{Math.min(
          (safePage + 1) * maxRows,
          filtered.length,
        )} / {filtered.length}</span
      >
      <div>
        <button
          class="icon-button"
          aria-label="上一页平仓记录"
          disabled={safePage === 0}
          on:click={() => (currentPage = safePage - 1)}
          ><ChevronLeft size={16} /></button
        ><span class="number">{safePage + 1} / {pageCount}</span><button
          class="icon-button"
          aria-label="下一页平仓记录"
          disabled={safePage >= pageCount - 1}
          on:click={() => (currentPage = safePage + 1)}
          ><ChevronRight size={16} /></button
        >
      </div>
    </div>{/if}
</section>

<style>
  .trade-search {
    display: flex;
    align-items: center;
    gap: 7px;
    color: var(--lo);
    border: 1px solid var(--line);
    border-radius: 4px;
    padding: 7px 9px;
  }
  .trade-search:focus-within {
    border-color: var(--pos);
  }
  .trade-search input {
    background: transparent;
    border: 0;
    outline: 0;
    color: var(--hi);
    font-size: 11px;
    width: 90px;
  }
  .trade-search input::placeholder {
    color: var(--lo);
  }
  .trade-totals {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 17px;
    font-size: 10px;
    color: var(--lo);
    padding-bottom: 13px;
  }
  .trade-row {
    display: grid;
    grid-template-columns: 1fr 1.4fr 1fr;
    align-items: center;
    gap: 10px;
    padding: 14px 0;
    border-top: 1px solid var(--panel2);
  }
  .trade-symbol strong {
    font-size: 13px;
    font-weight: 600;
  }
  .trade-price {
    text-align: center;
    font-size: 11px;
    color: var(--mid);
  }
  .trade-pnl {
    text-align: right;
    font-size: 12px;
  }
  .trade-secondary {
    font-size: 10px;
    color: var(--lo);
    margin-top: 5px;
  }
  .pagination {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    border-top: 1px solid var(--line);
    padding-top: 10px;
    margin-top: 4px;
    color: var(--lo);
    font-size: 10px;
  }
  .pagination > div {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .pagination :global(.icon-button) {
    width: 30px;
    height: 34px;
  }
  @media (max-width: 500px) {
    .trade-row {
      grid-template-columns: 1fr 1.1fr 1fr;
      gap: 7px;
    }
    .trade-price {
      font-size: 10px;
    }
    .trade-pnl {
      font-size: 11px;
    }
  }
</style>
