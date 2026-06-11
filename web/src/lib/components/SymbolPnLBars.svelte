<script lang="ts">
  import type { SymbolPnL } from '$lib/api';
  import { fmtSignedUSDT, fmtPct } from '$lib/format';

  export let rows: SymbolPnL[] = [];

  // Scale: use the max absolute PnL so the bar widths feel proportional.
  $: maxAbs = rows.reduce((m, r) => Math.max(m, Math.abs(r.total_pnl)), 0) || 1;
</script>

<div class="card overflow-hidden">
  <div class="px-5 py-4 border-b border-ink-800/80">
    <div class="label">按品种 PnL</div>
    <div class="stat-sub text-ink-400 mt-1">哪些品种是 alpha · 哪些是 drag</div>
  </div>
  {#if rows.length === 0}
    <div class="py-12 text-center text-ink-500 text-sm">没有已平仓数据</div>
  {:else}
    <div class="divide-y divide-ink-800/60">
      {#each rows as r}
        {@const pct = (Math.abs(r.total_pnl) / maxAbs) * 100}
        {@const isWin = r.total_pnl > 0}
        <div class="px-5 py-3 flex items-center gap-4">
          <div class="w-24 font-mono font-medium text-ink-50 text-sm">{r.symbol.replace('USDT', '')}</div>
          <div class="flex-1 relative h-7 rounded-lg bg-ink-800/40 overflow-hidden">
            <div
              class={'absolute top-0 bottom-0 left-0 ' + (isWin ? 'bg-accent-500/70' : 'bg-loss-500/70')}
              style={`width: ${pct.toFixed(2)}%`}
            ></div>
            <div class="relative h-full flex items-center justify-between px-3 text-xs font-mono">
              <span class="text-ink-100 text-shadow-sm">
                {r.trades} 单 · 胜率 {fmtPct(r.win_rate, 0)}
              </span>
              <span class={isWin ? 'pos font-semibold' : 'neg font-semibold'}>
                {fmtSignedUSDT(r.total_pnl, 2)}
              </span>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
