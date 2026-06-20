<script lang="ts">
  import type { EquityPoint } from '$lib/api';
  import { fmtPct, fmtDate } from '$lib/format';

  export let points: EquityPoint[] = [];
  export let height = 120;

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

  $: dd = (() => {
    let peak = -Infinity;
    return points.map((p) => {
      peak = Math.max(peak, p.nav);
      return { x: p.taken_at, y: peak > 0 ? p.nav / peak - 1 : 0 };
    });
  })();
  $: hasData = dd.length >= 2;
  $: maxDD = hasData ? Math.min(...dd.map((d) => d.y)) : 0;
  $: currentDD = hasData ? dd[dd.length - 1].y : 0;

  const W = 1000, PT = 12, PB = 16;
  $: H = height;
  $: x0 = hasData ? dd[0].x : 0;
  $: x1 = hasData ? dd[dd.length - 1].x : 1;
  $: dMin = hasData ? Math.min(...dd.map((d) => d.y)) : -0.01;
  function mapX(t: number): number { return ((t - x0) / (x1 - x0 || 1)) * W; }
  function mapY(v: number): number { return PT + (v / (dMin || -1)) * (H - PT - PB); }
  $: pts = dd.map((d) => ({ x: mapX(d.x), y: mapY(d.y) }));
  $: ddLine = smooth(pts);
  $: ddArea = hasData ? `${smooth(pts)} L ${pts[pts.length - 1].x.toFixed(1)} ${PT} L ${pts[0].x.toFixed(1)} ${PT} Z` : '';

  let hover: { x: number; y: number; t: number; v: number } | null = null;
  function onMove(e: MouseEvent) {
    if (!hasData) return;
    const svg = e.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0, bestD = Infinity;
    for (let i = 0; i < pts.length; i++) {
      const d = Math.abs(pts[i].x - px);
      if (d < bestD) { bestD = d; best = i; }
    }
    hover = { x: pts[best].x, y: pts[best].y, t: dd[best].x, v: dd[best].y };
  }
</script>

<div class="card p-4 sm:p-5">
  <div class="flex items-start justify-between gap-4 mb-3">
    <div class="text-[13px] font-bold">水下回撤</div>
    <div class="flex gap-5 text-right">
      <div>
        <div class="label">最大回撤</div>
        <div class="font-mono text-2xl font-semibold mt-0.5 neg">{fmtPct(maxDD, 1)}</div>
      </div>
      <div>
        <div class="label">当前回撤</div>
        <div class={'font-mono text-2xl font-semibold mt-0.5 ' + (currentDD < -0.0005 ? 'neg' : 'pos')}>{fmtPct(currentDD, 1)}</div>
      </div>
    </div>
  </div>

  {#if !hasData}
    <div class="flex items-center justify-center text-ink-500 text-sm" style="height:{H}px">数据不足</div>
  {:else}
    <svg viewBox={`0 0 ${W} ${H}`} class="w-full block overflow-visible" style="height:{H}px"
      preserveAspectRatio="none" on:mousemove={onMove} on:mouseleave={() => (hover = null)} role="img" aria-label="水下回撤">
      <defs>
        <linearGradient id="ddGrad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="oklch(0.70 0.155 24)" stop-opacity="0.04" />
          <stop offset="100%" stop-color="oklch(0.70 0.155 24)" stop-opacity="0.3" />
        </linearGradient>
      </defs>
      <line x1="0" x2={W} y1={PT} y2={PT} stroke="oklch(1 0 0 / 0.10)" stroke-width="1" stroke-dasharray="4 4" />
      <path d={ddArea} fill="url(#ddGrad)" stroke="none" />
      <path d={ddLine} fill="none" stroke="oklch(0.74 0.16 24)" stroke-width="2" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
      {#if hover}
        <line x1={hover.x} x2={hover.x} y1="0" y2={H} stroke="var(--mid)" stroke-width="1" opacity="0.5" />
        <circle cx={hover.x} cy={hover.y} r="3.5" fill="oklch(0.74 0.16 24)" stroke="var(--panel)" stroke-width="1.5" />
      {/if}
    </svg>
    {#if hover}
      <div class="mt-1.5 text-xs font-mono text-ink-300 flex justify-between">
        <span>{fmtDate(hover.t, true)}</span>
        <span class="neg">{fmtPct(hover.v, 2)}</span>
      </div>
    {/if}
  {/if}
</div>
