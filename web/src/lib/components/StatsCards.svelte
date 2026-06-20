<script lang="ts">
  import type { Stats } from '$lib/api';
  import { fmtPct, fmtSignedUSDT, fmtDuration } from '$lib/format';

  export let stats: Stats | null = null;
  export let window: number = 0;
</script>

<div class="grid grid-cols-2 lg:grid-cols-4 gap-3 md:gap-4">
  <div class="card p-4 md:p-5">
    <div class="label">胜率</div>
    <div class={'font-mono tabular-nums text-2xl font-semibold text-ink-50 mt-2 ' + ((stats?.win_rate ?? 0) >= 0.5 ? 'pos' : 'neg')}>
      {stats ? fmtPct(stats.win_rate, 1) : '—'}
    </div>
    <div class="stat-sub text-ink-400 mt-2">
      {#if stats}
        {stats.wins} 胜 · {stats.losses} 败 · {stats.total} 单
      {:else}—{/if}
    </div>
  </div>

  <div class="card p-4 md:p-5">
    <div class="label">累计已实现 PnL</div>
    <div class={'font-mono tabular-nums text-2xl font-semibold text-ink-50 mt-2 ' + ((stats?.total_pnl ?? 0) > 0 ? 'pos' : (stats?.total_pnl ?? 0) < 0 ? 'neg' : '')}>
      {stats ? fmtSignedUSDT(stats.total_pnl, 2) : '—'}
    </div>
    <div class="stat-sub text-ink-400 mt-2">USDT · 最近 {window} 单</div>
  </div>

  <div class="card p-4 md:p-5">
    <div class="label">赢赔比</div>
    <div class={'font-mono tabular-nums text-2xl font-semibold text-ink-50 mt-2 ' + ((stats?.win_loss_ratio ?? 0) >= 1 ? 'pos' : 'neg')}>
      {stats && stats.win_loss_ratio > 0 ? stats.win_loss_ratio.toFixed(2) : '—'}
    </div>
    <div class="stat-sub text-ink-400 mt-2">
      {#if stats}
        avg win {fmtSignedUSDT(stats.avg_win_usdt, 1)} / avg loss {fmtSignedUSDT(stats.avg_loss_usdt, 1)}
      {:else}—{/if}
    </div>
  </div>

  <div class="card p-4 md:p-5">
    <div class="label">平均持仓时长</div>
    <div class="font-mono tabular-nums text-2xl font-semibold text-ink-50 mt-2">
      {stats ? fmtDuration((stats.avg_hold_hours || 0) * 3600_000) : '—'}
    </div>
    <div class="stat-sub text-ink-400 mt-2">
      中位 {stats ? fmtDuration((stats.median_hold_hours || 0) * 3600_000) : '—'}
    </div>
  </div>
</div>
