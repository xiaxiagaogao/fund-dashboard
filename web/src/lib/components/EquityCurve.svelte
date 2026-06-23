<script lang="ts">
  import { fmtDate, despike } from '$lib/format';
  import type { EquityPoint } from '$lib/api';

  export let points: EquityPoint[] = [];
  export let height = 200;
  export let glow = true;

  // Catmull-Rom → cubic bézier smoothing (from the design canvas).
  function smooth(pts: { x: number; y: number }[]): string {
    if (pts.length < 2) return pts.length ? `M${pts[0].x} ${pts[0].y}` : '';
    let d = `M${pts[0].x.toFixed(1)} ${pts[0].y.toFixed(1)}`;
    for (let i = 0; i < pts.length - 1; i++) {
      const p0 = pts[i - 1] || pts[i], p1 = pts[i], p2 = pts[i + 1], p3 = pts[i + 2] || p2;
      const c1x = p1.x + (p2.x - p0.x) / 6, c1y = p1.y + (p2.y - p0.y) / 6;
      const c2x = p2.x - (p3.x - p1.x) / 6, c2y = p2.y - (p3.y - p1.y) / 6;
      d += ` C ${c1x.toFixed(1)} ${c1y.toFixed(1)}, ${c2x.toFixed(1)} ${c2y.toFixed(1)}, ${p2.x.toFixed(1)} ${p2.y.toFixed(1)}`;
    }
    return d;
  }

  const W = 1000, PT = 16, PB = 22;
  $: H = height;
  $: hasData = points.length >= 2;

  // NAV series with deposit-timing spikes filtered out (see format.despike).
  $: nav = despike(points.map((p) => p.nav));

  $: x0 = hasData ? points[0].taken_at : 0;
  $: x1 = hasData ? points[points.length - 1].taken_at : 1;
  // Always include the par line (NAV = 1.0) in range so it stays visible.
  $: lo = hasData ? Math.min(...nav, 1.0) : 0;
  $: hi = hasData ? Math.max(...nav, 1.0) : 1;
  $: pad = (hi - lo) * 0.12 || 1;
  $: yMin = lo - pad;
  $: yMax = hi + pad;

  function mapX(t: number): number {
    return ((t - x0) / (x1 - x0 || 1)) * W;
  }
  function mapY(v: number): number {
    return H - PB - ((v - yMin) / (yMax - yMin || 1)) * (H - PT - PB);
  }
  $: topPts = points.map((p, i) => ({ x: mapX(p.taken_at), y: mapY(nav[i]) }));
  $: eqLine = smooth(topPts);
  $: eqArea = hasData
    ? `${smooth(topPts)} L ${topPts[topPts.length - 1].x.toFixed(1)} ${H} L ${topPts[0].x.toFixed(1)} ${H} Z`
    : '';
  $: endDot = hasData ? topPts[topPts.length - 1] : { x: 0, y: 0 };
  $: xTicks = hasData ? [0, 0.33, 0.66, 1].map((f) => points[Math.round(f * (points.length - 1))].taken_at) : [];

  let hover: { x: number; y: number; t: number; v: number } | null = null;
  function onMove(e: MouseEvent) {
    if (!hasData) return;
    const svg = e.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0, bestD = Infinity;
    for (let i = 0; i < topPts.length; i++) {
      const d = Math.abs(topPts[i].x - px);
      if (d < bestD) { bestD = d; best = i; }
    }
    hover = { x: topPts[best].x, y: topPts[best].y, t: points[best].taken_at, v: nav[best] };
  }
</script>

<div class="card p-4 sm:p-5">
  <div class="flex items-center justify-between gap-4 mb-3">
    <div class="text-[13px] font-bold">净值曲线 <span class="text-[11px] text-ink-500 font-normal ml-1">基金 NAV</span></div>
    <div class="hidden sm:flex items-center gap-1.5 text-[11px] text-ink-500">
      <span class="w-3.5 border-t border-dashed" style="border-color:var(--lo)"></span>本金 1.0
    </div>
  </div>

  {#if !hasData}
    <div class="flex items-center justify-center text-ink-500 text-sm" style="height:{H}px">
      还没有足够的快照数据
    </div>
  {:else}
    <svg viewBox={`0 0 ${W} ${H}`} class="w-full block overflow-visible" style="height:{H}px"
      preserveAspectRatio="none" on:mousemove={onMove} on:mouseleave={() => (hover = null)} role="img" aria-label="净值曲线">
      <defs>
        <linearGradient id="eqGrad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="oklch(0.80 0.115 168)" stop-opacity="0.30" />
          <stop offset="100%" stop-color="oklch(0.80 0.115 168)" stop-opacity="0.01" />
        </linearGradient>
        <filter id="eqGlow" x="-20%" y="-40%" width="140%" height="180%">
          <feGaussianBlur stdDeviation="4" result="b" /><feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>
      <path d={eqArea} fill="url(#eqGrad)" stroke="none" />
      <line x1="0" x2={W} y1={mapY(1.0)} y2={mapY(1.0)} stroke="var(--lo)" stroke-width="1" stroke-dasharray="4 5" opacity="0.6" />
      {#if glow}
        <path d={eqLine} fill="none" stroke="oklch(0.84 0.12 168)" stroke-width="2.5" vector-effect="non-scaling-stroke" filter="url(#eqGlow)" opacity="0.9" />
      {/if}
      <path d={eqLine} fill="none" stroke="oklch(0.86 0.12 168)" stroke-width="2.5" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
      {#if hover}
        <line x1={hover.x} x2={hover.x} y1="0" y2={H} stroke="var(--mid)" stroke-width="1" opacity="0.5" />
      {/if}
      <circle cx={(hover ?? endDot).x} cy={(hover ?? endDot).y} r="4" fill="oklch(0.90 0.10 168)" stroke="var(--panel)" stroke-width="2" />
    </svg>
    <div class="flex justify-between mt-1.5">
      {#if hover}
        <span class="text-[10px] text-ink-400 font-mono">{fmtDate(hover.t, true)}</span>
        <span class="text-[10px] text-ink-100 font-mono">{hover.v.toFixed(4)}</span>
      {:else}
        {#each xTicks as t}<span class="text-[10px] text-ink-500 font-mono">{fmtDate(t)}</span>{/each}
      {/if}
    </div>
  {/if}
</div>
