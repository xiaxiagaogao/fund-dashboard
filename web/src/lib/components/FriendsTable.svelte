<script lang="ts">
  import { fmtUSDT, fmtShares, fmtSignedPct, fmtSignedUSDT } from '$lib/format';
  import type { AggregateRow } from '$lib/api';

  export let rows: AggregateRow[] = [];
  /** Highlight which username is "me". */
  export let me: string = '';
</script>

<div class="card overflow-hidden">
  <div class="px-5 py-4 border-b border-ink-800/80">
    <div class="label">基金成员</div>
    <div class="stat-sub text-ink-400 mt-1">朋友间相互可见 · 按当前 NAV 估值</div>
  </div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead class="text-ink-400 text-xs uppercase tracking-wider">
        <tr>
          <th class="text-left py-2.5 px-5 font-medium">成员</th>
          <th class="text-right py-2.5 px-3 font-medium">份额</th>
          <th class="text-right py-2.5 px-3 font-medium">投入 (USDT)</th>
          <th class="text-right py-2.5 px-3 font-medium">当前估值</th>
          <th class="text-right py-2.5 px-3 font-medium">PnL (USDT)</th>
          <th class="text-right py-2.5 px-5 font-medium">PnL %</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as r}
          <tr class={'table-row-hover border-t border-ink-800/60 ' + (r.username === me ? 'bg-accent-500/[0.04]' : '')}>
            <td class="py-3 px-5">
              <div class="flex items-center gap-2">
                <span class="text-ink-50">{r.name}</span>
                {#if r.username === me}<span class="pill-neutral">我</span>{/if}
                {#if r.is_admin}<span class="pill-neutral">admin</span>{/if}
              </div>
              <div class="text-[11px] text-ink-500 font-mono">{r.username}</div>
            </td>
            <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-200">{fmtShares(r.shares, 4)}</td>
            <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-200">{fmtUSDT(r.net_deposits, 2)}</td>
            <td class="py-3 px-3 text-right font-mono tabular-nums text-ink-50">{fmtUSDT(r.value_usdt, 2)}</td>
            <td class={'py-3 px-3 text-right font-mono tabular-nums ' + (r.pnl_usdt > 0 ? 'pos' : r.pnl_usdt < 0 ? 'neg' : 'text-ink-300')}>
              {fmtSignedUSDT(r.pnl_usdt, 2)}
            </td>
            <td class="py-3 px-5 text-right">
              <span class={r.pnl_pct > 0 ? 'pill-pos' : r.pnl_pct < 0 ? 'pill-neg' : 'pill-neutral'}>
                {fmtSignedPct(r.pnl_pct)}
              </span>
            </td>
          </tr>
        {:else}
          <tr>
            <td colspan="6" class="py-8 text-center text-ink-500 text-sm">还没有成员</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
