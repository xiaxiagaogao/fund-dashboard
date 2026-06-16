<script lang="ts">
  import type { EquityPoint } from '$lib/api';
  import { fmtPct, fmtDate } from '$lib/format';

  export let points: EquityPoint[] = [];
  export let height = 150;

  // Underwater curve: drawdown at each point = (nav - runningMax) / runningMax,
  // always ≤ 0. The deepest point is the max drawdown.
  $: dd = (() => {
    let peak = -Infinity;
    return points.map((p) => {
      peak = Math.max(peak, p.nav);
      const d = peak > 0 ? (p.nav - peak) / peak : 0;
      return { x: p.taken_at, y: d };
    });
  })();

  $: hasData = dd.length >= 2;
  $: maxDD = hasData ? Math.min(...dd.map((d) => d.y)) : 0;
  $: currentDD = hasData ? dd[dd.length - 1].y : 0;

  $: xMin = hasData ? dd[0].x : 0;
  $: xMax = hasData ? dd[dd.length - 1].x : 1;
  $: yMin = hasData ? Math.min(...dd.map((d) => d.y)) : -0.01; // most negative
  $: yLo = yMin * 1.1 - 0.001; // a little headroom below the trough

  const W = 1000;
  $: H = height;
  function xS(x: number): number {
    const span = xMax - xMin || 1;
    return ((x - xMin) / span) * (W - 24) + 12;
  }
  // y=0 (no drawdown) sits at the top; deeper drawdown goes down.
  function yS(y: number): number {
    const span = 0 - yLo || 1;
    return ((0 - y) / span) * (H - 20) + 8;
  }
  $: line = dd.map((d, i) => `${i === 0 ? 'M' : 'L'} ${xS(d.x).toFixed(1)} ${yS(d.y).toFixed(1)}`).join(' ');
  $: area = hasData ? `${line} L ${xS(xMax).toFixed(1)} ${yS(0).toFixed(1)} L ${xS(xMin).toFixed(1)} ${yS(0).toFixed(1)} Z` : '';

  let hover: { x: number; y: number; pt: { x: number; y: number } } | null = null;
  function onMove(e: MouseEvent) {
    if (!hasData) return;
    const svg = e.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0, bestD = Infinity;
    for (let i = 0; i < dd.length; i++) {
      const d = Math.abs(xS(dd[i].x) - px);
      if (d < bestD) { bestD = d; best = i; }
    }
    hover = { x: xS(dd[best].x), y: yS(dd[best].y), pt: dd[best] };
  }
</script>

<div class="card p-5">
  <div class="flex items-start justify-between gap-4 mb-3">
    <div>
      <div class="label">回撤（水下）</div>
      <div class="stat-sub text-ink-400 mt-1">从历史高点的回落 · NAV 口径</div>
    </div>
    <div class="flex gap-5 text-right">
      <div>
        <div class="label">最大回撤</div>
        <div class="stat-value text-2xl mt-1 neg">{fmtPct(maxDD, 1)}</div>
      </div>
      <div>
        <div class="label">当前回撤</div>
        <div class={'stat-value text-2xl mt-1 ' + (currentDD < -0.0005 ? 'neg' : 'pos')}>{fmtPct(currentDD, 1)}</div>
      </div>
    </div>
  </div>

  {#if !hasData}
    <div class="h-[{H}px] flex items-center justify-center text-ink-500 text-sm">数据不足</div>
  {:else}
    <svg viewBox={`0 0 ${W} ${H}`} class="w-full" style="height: {H}px" preserveAspectRatio="none"
      on:mousemove={onMove} on:mouseleave={() => (hover = null)} role="img" aria-label="drawdown">
      <defs>
        <linearGradient id="ddgrad" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color="#ef4444" stop-opacity="0.05" />
          <stop offset="100%" stop-color="#ef4444" stop-opacity="0.32" />
        </linearGradient>
      </defs>
      <line x1="12" x2={W - 12} y1={yS(0)} y2={yS(0)} stroke="#3f3f46" stroke-width="1" opacity="0.7" />
      <path d={area} fill="url(#ddgrad)" />
      <path d={line} fill="none" stroke="#f87171" stroke-width="1.5" stroke-linejoin="round" />
      {#if hover}
        <line x1={hover.x} x2={hover.x} y1="0" y2={H} stroke="#52525b" stroke-width="1" />
        <circle cx={hover.x} cy={hover.y} r="3.5" fill="#f87171" stroke="#09090b" stroke-width="1.2" />
      {/if}
    </svg>
    {#if hover}
      <div class="mt-2 text-xs font-mono text-ink-300 flex justify-between">
        <span>{fmtDate(hover.pt.x, true)}</span>
        <span class="neg">{fmtPct(hover.pt.y, 2)}</span>
      </div>
    {/if}
  {/if}
</div>
