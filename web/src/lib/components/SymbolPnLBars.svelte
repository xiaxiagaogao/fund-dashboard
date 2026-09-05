<script lang="ts">
  import type { SymbolPnL } from "$lib/api";
  import { fmtSignedUSDT, fmtPct, pnlClass } from "$lib/format";
  export let rows: SymbolPnL[] = [];
  $: ordered = [...rows].sort((a, b) => b.total_pnl - a.total_pnl);
  $: maxAbs = Math.max(1, ...rows.map((r) => Math.abs(r.total_pnl)));
</script>

<section aria-label="标的贡献">
  <div class="section-head">
    <h2>标的贡献</h2>
    <span class="section-meta">净收益 · USDT</span>
  </div>
  {#if rows.length === 0}<div class="empty-state">
      没有已平仓数据
    </div>{:else}<div class="symbol-rows">
      {#each ordered as row}<div class="symbol-row">
          <div class="flex items-center justify-between gap-3">
            <strong class="text-xs font-semibold"
              >{row.symbol.replace(/USDT$/, "")}</strong
            ><span class={"number text-xs " + pnlClass(row.total_pnl)}
              >{fmtSignedUSDT(row.total_pnl)}</span
            >
          </div>
          <div class="bar-track">
            <div
              style:width={`${(Math.abs(row.total_pnl) / maxAbs) * 100}%`}
              style:background={row.total_pnl >= 0
                ? "var(--pos)"
                : "var(--neg)"}
            ></div>
          </div>
          <div class="text-[10px] text-ink-400 mt-2">
            {row.trades} 笔 · 胜率
            <span class="number">{fmtPct(row.win_rate, 0)}</span>
          </div>
        </div>{/each}
    </div>{/if}
</section>

<style>
  .symbol-rows {
    display: grid;
    gap: 22px;
    max-height: 490px;
    overflow: auto;
  }
  .bar-track {
    height: 4px;
    background: var(--panel2);
    margin-top: 10px;
  }
  .bar-track > div {
    height: 100%;
  }
</style>
