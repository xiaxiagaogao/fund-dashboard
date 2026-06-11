<script lang="ts">
  import type { Position } from '$lib/api';
  import { fmtDate, fmtDuration, fmtSignedUSDT, fmtPct } from '$lib/format';

  export let positions: Position[] = [];
  export let limit = 20;

  $: visible = positions.slice(0, limit);
  $: hasMore = positions.length > limit;
</script>

<div class="card overflow-hidden">
  <div class="px-5 py-4 border-b border-ink-800/80">
    <div class="label">最近平仓</div>
    <div class="stat-sub text-ink-400 mt-1">入/出 · 已实现 PnL · 持仓时长</div>
  </div>
  {#if visible.length === 0}
    <div class="py-12 text-center text-ink-500 text-sm">还没有已平仓的交易</div>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="text-ink-400 text-xs uppercase tracking-wider">
          <tr>
            <th class="text-left py-2.5 px-5 font-medium">Symbol</th>
            <th class="text-left py-2.5 px-3 font-medium">方向</th>
            <th class="text-right py-2.5 px-3 font-medium">入场</th>
            <th class="text-right py-2.5 px-3 font-medium">出场</th>
            <th class="text-right py-2.5 px-3 font-medium">变动 %</th>
            <th class="text-right py-2.5 px-3 font-medium">PnL</th>
            <th class="text-right py-2.5 px-5 font-medium">持仓时长</th>
          </tr>
        </thead>
        <tbody>
          {#each visible as p}
            {@const isLong = p.side === 'LONG' || p.side === 'BUY'}
            {@const priceChange = p.entry_price > 0 ? (p.exit_price - p.entry_price) / p.entry_price : 0}
            {@const effectiveChange = isLong ? priceChange : -priceChange}
            <tr class="table-row-hover border-t border-ink-800/60">
              <td class="py-3 px-5 font-mono font-medium text-ink-50">{p.symbol}</td>
              <td class="py-3 px-3">
                <span class={isLong ? 'pill-pos' : 'pill-neg'}>{isLong ? '多' : '空'}</span>
              </td>
              <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-200">{p.entry_price.toFixed(4)}</td>
              <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-200">{p.exit_price.toFixed(4)}</td>
              <td class={'py-3 px-3 text-right font-mono tabular-nums ' + (effectiveChange > 0 ? 'pos' : effectiveChange < 0 ? 'neg' : 'text-ink-300')}>
                {(effectiveChange > 0 ? '+' : '') + fmtPct(effectiveChange, 2)}
              </td>
              <td class={'py-3 px-3 text-right font-mono tabular-nums font-medium ' + (p.realized_pnl > 0 ? 'pos' : p.realized_pnl < 0 ? 'neg' : 'text-ink-300')}>
                {fmtSignedUSDT(p.realized_pnl, 4)}
              </td>
              <td class="py-3 px-5 text-right">
                <div class="font-mono tabular-nums text-ink-100">{fmtDuration((p.exit_time ?? 0) - p.entry_time)}</div>
                <div class="text-[11px] font-mono text-ink-500">{fmtDate(p.exit_time, true)}</div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if hasMore}
      <div class="px-5 py-3 border-t border-ink-800/60 text-center">
        <button class="btn-link text-xs" on:click={() => (limit = positions.length)}>
          展开全部 {positions.length} 条
        </button>
      </div>
    {/if}
  {/if}
</div>
