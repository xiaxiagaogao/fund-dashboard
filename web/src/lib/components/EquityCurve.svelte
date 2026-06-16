<script lang="ts">
  import { fmtUSDT, fmtDate, fmtSignedPct } from '$lib/format';
  import type { EquityPoint } from '$lib/api';

  export let points: EquityPoint[] = [];
  /**
   * 'friend' shows a 我的 / 基金 toggle (my USDT value vs NAV performance);
   * 'fund' shows the operator's NAV / Equity toggle.
   */
  export let variant: 'friend' | 'fund' = 'fund';
  /** My per-point value (shares-as-of-point × NAV), aligned 1:1 with points. Only used in the 'friend' variant. */
  export let mineValues: number[] | null = null;
  /** Which series to draw. */
  export let mode: 'mine' | 'nav' | 'equity' = variant === 'friend' ? 'mine' : 'nav';
  export let height = 220;

  $: isMoney = mode !== 'nav'; // mine + equity are USDT; nav is per-share

  $: data = points.map((p, i) => ({
    x: p.taken_at,
    y: mode === 'nav' ? p.nav : mode === 'mine' ? (mineValues?.[i] ?? 0) : p.total_equity,
    src: p.source
  }));

  $: hasData = data.length >= 2;
  $: first = hasData ? data[0].y : 0;
  $: last = hasData ? data[data.length - 1].y : 0;

  // The pill ALWAYS shows performance (NAV-based PnL) so it stays meaningful
  // whether the user is looking at the equity curve (where deltas include
  // deposit/withdraw steps) or the NAV curve. Otherwise a $1000 deposit on a
  // $1000 pool would render the pill as "+100%" even though no money was made.
  $: navFirst = hasData ? points[0].nav : 0;
  $: navLast = hasData ? points[points.length - 1].nav : 0;
  $: deltaPct = navFirst > 0 ? (navLast - navFirst) / navFirst : 0;

  // Bounds with a tiny padding so the curve doesn't kiss the edges.
  $: xMin = hasData ? data[0].x : 0;
  $: xMax = hasData ? data[data.length - 1].x : 1;
  $: yVals = data.map((d) => d.y);
  $: yMin = hasData ? Math.min(...yVals) : 0;
  $: yMax = hasData ? Math.max(...yVals) : 1;
  $: ySpan = yMax - yMin || 1;
  $: yLo = yMin - ySpan * 0.08;
  $: yHi = yMax + ySpan * 0.08;

  const W = 1000; // virtual coords; SVG scales via viewBox
  $: H = height;

  function xS(x: number): number {
    const span = xMax - xMin || 1;
    return ((x - xMin) / span) * (W - 24) + 12;
  }
  function yS(y: number): number {
    const span = yHi - yLo || 1;
    return H - ((y - yLo) / span) * (H - 24) - 12;
  }

  $: line = data.map((d, i) => `${i === 0 ? 'M' : 'L'} ${xS(d.x).toFixed(1)} ${yS(d.y).toFixed(1)}`).join(' ');
  $: area =
    hasData
      ? `${line} L ${xS(data[data.length - 1].x).toFixed(1)} ${H} L ${xS(data[0].x).toFixed(1)} ${H} Z`
      : '';

  // hover state
  let hover: { x: number; y: number; pt: { x: number; y: number; src: string } } | null = null;

  function onMove(e: MouseEvent) {
    if (!hasData) return;
    const svg = e.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((e.clientX - rect.left) / rect.width) * W;
    // nearest by x
    let best = 0;
    let bestD = Infinity;
    for (let i = 0; i < data.length; i++) {
      const d = Math.abs(xS(data[i].x) - px);
      if (d < bestD) {
        bestD = d;
        best = i;
      }
    }
    hover = { x: xS(data[best].x), y: yS(data[best].y), pt: data[best] };
  }
  function onLeave() {
    hover = null;
  }
</script>

<div class="card p-5">
  <div class="flex items-start justify-between gap-4 mb-4">
    <div>
      <div class="label">
        {#if mode === 'mine'}我的估值（USDT）{:else if mode === 'nav'}NAV / 每份额价值{:else}基金总资金（USDT）{/if}
      </div>
      <div class="flex items-baseline gap-3 mt-1">
        <div class="stat-value">
          {isMoney ? fmtUSDT(last, 2) : last.toFixed(6)}
        </div>
        {#if hasData}
          <span class={deltaPct >= 0 ? 'pill-pos' : 'pill-neg'}>
            {fmtSignedPct(deltaPct)}
          </span>
        {/if}
      </div>
      <div class="stat-sub mt-1 text-ink-400">
        {#if mode === 'mine'}
          我的份额 × NAV · 含我的入金/赎回
        {:else if mode === 'nav'}
          反映真实表现 · 不受入金影响
        {:else}
          总规模 · 包含成员入金/赎回事件
        {/if}
      </div>
      <div class="stat-sub mt-0.5 text-ink-500 text-[11px]">
        {data.length} 个数据点{#if hasData} · {fmtDate(xMin, true)} → {fmtDate(xMax, true)}{/if}
      </div>
    </div>
    <div class="flex gap-1">
      {#if variant === 'friend'}
        <button
          class={'btn px-2.5 py-1 text-xs ' + (mode === 'mine' ? 'btn-primary' : 'btn-ghost')}
          on:click={() => (mode = 'mine')}>我的</button
        >
        <button
          class={'btn px-2.5 py-1 text-xs ' + (mode === 'nav' ? 'btn-primary' : 'btn-ghost')}
          on:click={() => (mode = 'nav')}>基金</button
        >
      {:else}
        <button
          class={'btn px-2.5 py-1 text-xs ' + (mode === 'equity' ? 'btn-primary' : 'btn-ghost')}
          on:click={() => (mode = 'equity')}>Equity</button
        >
        <button
          class={'btn px-2.5 py-1 text-xs ' + (mode === 'nav' ? 'btn-primary' : 'btn-ghost')}
          on:click={() => (mode = 'nav')}>NAV</button
        >
      {/if}
    </div>
  </div>

  {#if !hasData}
    <div class="h-[{H}px] flex items-center justify-center text-ink-500 text-sm">
      还没有足够的快照数据 —— 等下一次 hourly snapshot 或者手工触发一次
    </div>
  {:else}
    <svg
      viewBox={`0 0 ${W} ${H}`}
      class="w-full"
      style="height: {H}px"
      preserveAspectRatio="none"
      on:mousemove={onMove}
      on:mouseleave={onLeave}
      role="img"
      aria-label="equity curve"
    >
      <defs>
        <linearGradient id="grad" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color="#84cc16" stop-opacity="0.28" />
          <stop offset="100%" stop-color="#84cc16" stop-opacity="0" />
        </linearGradient>
      </defs>

      <!-- baseline reference (first point) -->
      <line
        x1="12"
        x2={W - 12}
        y1={yS(first)}
        y2={yS(first)}
        stroke="#3f3f46"
        stroke-dasharray="3 5"
        stroke-width="1"
        opacity="0.6"
      />

      <path d={area} fill="url(#grad)" />
      <path d={line} fill="none" stroke="#a3e635" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />

      <!-- cash event markers -->
      {#each data as d}
        {#if d.src === 'cash_event'}
          <circle cx={xS(d.x)} cy={yS(d.y)} r="3.5" fill="#e4e4e7" stroke="#18181b" stroke-width="1.2" />
        {/if}
      {/each}

      {#if hover}
        <line x1={hover.x} x2={hover.x} y1="0" y2={H} stroke="#52525b" stroke-width="1" />
        <circle cx={hover.x} cy={hover.y} r="4" fill="#a3e635" stroke="#09090b" stroke-width="1.5" />
      {/if}
    </svg>

    {#if hover}
      <div class="mt-2 text-xs font-mono text-ink-300 flex justify-between">
        <span>{fmtDate(hover.pt.x, true)}{hover.pt.src === 'cash_event' ? ' · 入金事件' : ''}</span>
        <span class="text-ink-100">{isMoney ? fmtUSDT(hover.pt.y, 2) + ' USDT' : hover.pt.y.toFixed(6)}</span>
      </div>
    {/if}
  {/if}
</div>
