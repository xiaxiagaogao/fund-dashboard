<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { fmtDate, fmtPct, despike } from '$lib/format';
  import type { EquityPoint, IndexPoint } from '$lib/api';
  import { RANGES, type RangeKey } from '$lib/ranges';

  export let points: EquityPoint[] = [];
  export let qqq: IndexPoint[] = [];
  export let spy: IndexPoint[] = [];
  export let height = 220;
  export let glow = true;
  export let range: RangeKey = '30d';
  export let loading = false;

  const dispatch = createEventDispatcher<{ range: RangeKey }>();

  const C_FUND = 'oklch(0.86 0.12 168)';
  const C_QQQ = 'oklch(0.70 0.11 250)';
  const C_SPY = 'oklch(0.80 0.10 85)';

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

  // Linear interpolation of a daily-close series at an arbitrary time. Clamps
  // at the ends. Lets us resample the (daily) index onto the (30-min) NAV
  // timestamps so all series share one x-axis and one t0 baseline.
  function interp(pts: IndexPoint[], t: number): number {
    const n = pts.length;
    if (n === 0) return NaN;
    if (t <= pts[0].t) return pts[0].close;
    if (t >= pts[n - 1].t) return pts[n - 1].close;
    for (let i = 1; i < n; i++) {
      if (pts[i].t >= t) {
        const a = pts[i - 1], b = pts[i];
        return a.close + ((t - a.t) / ((b.t - a.t) || 1)) * (b.close - a.close);
      }
    }
    return pts[n - 1].close;
  }

  const W = 1000, PT = 18, PB = 22;
  $: H = height;
  $: hasData = points.length >= 2;

  $: nav = despike(points.map((p) => p.nav));
  $: ts = points.map((p) => p.taken_at);
  $: t0 = hasData ? ts[0] : 0;
  $: t1 = hasData ? ts[ts.length - 1] : 1;

  $: hasQqq = qqq.length > 0;
  $: hasSpy = spy.length > 0;
  $: qqqBase = hasQqq ? interp(qqq, t0) : NaN;
  $: spyBase = hasSpy ? interp(spy, t0) : NaN;

  // Cumulative % return since t0 for each series, sampled at the NAV times.
  $: fundRet = hasData ? nav.map((v) => v / nav[0] - 1) : [];
  $: qqqRet = hasData && hasQqq && qqqBase > 0 ? ts.map((t) => interp(qqq, t) / qqqBase - 1) : null;
  $: spyRet = hasData && hasSpy && spyBase > 0 ? ts.map((t) => interp(spy, t) / spyBase - 1) : null;

  $: allRet = [fundRet, qqqRet, spyRet].filter((s): s is number[] => !!s);
  $: lo = hasData ? Math.min(0, ...allRet.flat()) : 0;
  $: hi = hasData ? Math.max(0, ...allRet.flat()) : 0;
  $: pad = (hi - lo) * 0.15 || 0.01;
  $: yMin = lo - pad;
  $: yMax = hi + pad;

  function mapX(t: number): number {
    return ((t - t0) / (t1 - t0 || 1)) * W;
  }
  function mapY(v: number): number {
    return H - PB - ((v - yMin) / (yMax - yMin || 1)) * (H - PT - PB);
  }
  function path(ret: number[]): string {
    return smooth(ret.map((v, i) => ({ x: mapX(ts[i]), y: mapY(v) })));
  }
  $: dot = (ret: number[]) => ({ x: mapX(ts[ts.length - 1]), y: mapY(ret[ret.length - 1]) });
  $: xTicks = hasData ? [0, 0.33, 0.66, 1].map((f) => ts[Math.round(f * (ts.length - 1))]) : [];

  // Legend / outperformance reflect the hovered index, else the latest point.
  let hoverIdx: number | null = null;
  $: idx = hoverIdx ?? (hasData ? ts.length - 1 : 0);
  $: vsQqq = qqqRet ? fundRet[idx] - qqqRet[idx] : null;
  $: vsSpy = spyRet ? fundRet[idx] - spyRet[idx] : null;

  function onMove(e: MouseEvent) {
    if (!hasData) return;
    const svg = e.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    let best = 0, bestD = Infinity;
    for (let i = 0; i < ts.length; i++) {
      const d = Math.abs(mapX(ts[i]) - px);
      if (d < bestD) { bestD = d; best = i; }
    }
    hoverIdx = best;
  }
  function vs(delta: number): string {
    return (delta >= 0 ? '跑赢' : '跑输') + ' ' + fmtPct(Math.abs(delta), 1);
  }
</script>

<div class="card p-4 sm:p-5">
  <div class="flex items-center justify-between gap-2 mb-2.5 flex-wrap">
    <div class="text-[13px] font-bold">收益对比 <span class="text-[11px] text-ink-500 font-normal ml-1">vs 大盘</span></div>
    <div class="flex items-center gap-0.5 flex-none">
      {#each RANGES as r}
        <button
          type="button"
          class={'px-2 py-0.5 rounded-md text-[11px] font-medium transition-colors ' +
            (r.key === range ? 'bg-accent-500/15 text-accent-400' : 'text-ink-500 hover:text-ink-300')}
          aria-pressed={r.key === range}
          disabled={r.key === range || loading}
          on:click={() => dispatch('range', r.key)}>{r.label}</button>
      {/each}
    </div>
  </div>

  {#if !hasData}
    <div class="flex items-center justify-center text-ink-500 text-sm" style="height:{H}px">
      还没有足够的快照数据
    </div>
  {:else}
    <div class="flex flex-wrap gap-x-4 gap-y-1 mb-2.5 text-[11px] font-mono">
      <span class="inline-flex items-center gap-1.5"><span class="w-2.5 h-1 rounded" style="background:{C_FUND}"></span><span class="text-ink-300">基金</span> <span class={fundRet[idx] >= 0 ? 'pos' : 'neg'}>{fmtPct(fundRet[idx], 1)}</span></span>
      {#if qqqRet}<span class="inline-flex items-center gap-1.5"><span class="w-2.5 h-1 rounded" style="background:{C_QQQ}"></span><span class="text-ink-300">纳指100</span> <span class={qqqRet[idx] >= 0 ? 'pos' : 'neg'}>{fmtPct(qqqRet[idx], 1)}</span></span>{/if}
      {#if spyRet}<span class="inline-flex items-center gap-1.5"><span class="w-2.5 h-1 rounded" style="background:{C_SPY}"></span><span class="text-ink-300">标普500</span> <span class={spyRet[idx] >= 0 ? 'pos' : 'neg'}>{fmtPct(spyRet[idx], 1)}</span></span>{/if}
    </div>

    <svg viewBox={`0 0 ${W} ${H}`} class="w-full block overflow-visible transition-opacity" style="height:{H}px;opacity:{loading ? 0.45 : 1}"
      preserveAspectRatio="none" on:mousemove={onMove} on:mouseleave={() => (hoverIdx = null)} role="img" aria-label="收益对比">
      <defs>
        <filter id="eqGlow" x="-20%" y="-40%" width="140%" height="180%">
          <feGaussianBlur stdDeviation="4" result="b" /><feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
        </filter>
      </defs>
      <!-- 0% baseline -->
      <line x1="0" x2={W} y1={mapY(0)} y2={mapY(0)} stroke="var(--lo)" stroke-width="1" stroke-dasharray="4 5" opacity="0.6" />
      {#if spyRet}
        <path d={path(spyRet)} fill="none" stroke={C_SPY} stroke-width="1.5" stroke-linejoin="round" vector-effect="non-scaling-stroke" opacity="0.85" />
      {/if}
      {#if qqqRet}
        <path d={path(qqqRet)} fill="none" stroke={C_QQQ} stroke-width="1.5" stroke-linejoin="round" vector-effect="non-scaling-stroke" opacity="0.85" />
      {/if}
      {#if glow}
        <path d={path(fundRet)} fill="none" stroke={C_FUND} stroke-width="2.5" vector-effect="non-scaling-stroke" filter="url(#eqGlow)" opacity="0.9" />
      {/if}
      <path d={path(fundRet)} fill="none" stroke={C_FUND} stroke-width="2.5" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
      {#if hoverIdx !== null}
        <line x1={mapX(ts[idx])} x2={mapX(ts[idx])} y1="0" y2={H} stroke="var(--mid)" stroke-width="1" opacity="0.5" />
      {/if}
      {#if spyRet}<circle cx={dot(spyRet).x} cy={dot(spyRet).y} r="2.5" fill={C_SPY} />{/if}
      {#if qqqRet}<circle cx={dot(qqqRet).x} cy={dot(qqqRet).y} r="2.5" fill={C_QQQ} />{/if}
      <circle cx={dot(fundRet).x} cy={dot(fundRet).y} r="4" fill="oklch(0.90 0.10 168)" stroke="var(--panel)" stroke-width="2" />
    </svg>

    <div class="flex justify-between items-center mt-1.5">
      <span class="text-[10px] text-ink-500 font-mono">{hoverIdx !== null ? fmtDate(ts[idx], true) : (xTicks.length ? fmtDate(t0) + ' → ' + fmtDate(t1) : '')}</span>
      <span class="text-[11px] font-mono">
        {#if vsQqq !== null}<span class={vsQqq >= 0 ? 'pos' : 'neg'}>纳指 {vs(vsQqq)}</span>{/if}
        {#if vsQqq !== null && vsSpy !== null}<span class="text-ink-600"> · </span>{/if}
        {#if vsSpy !== null}<span class={vsSpy >= 0 ? 'pos' : 'neg'}>标普 {vs(vsSpy)}</span>{/if}
      </span>
    </div>
  {/if}
</div>
