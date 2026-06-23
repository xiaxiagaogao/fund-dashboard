// Display formatting helpers. All numeric outputs use tabular-nums so columns line up.

export function fmtUSDT(n: number, dp = 2): string {
  if (!isFinite(n)) return '—';
  const s = n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp });
  return s;
}

export function fmtShares(n: number, dp = 4): string {
  if (!isFinite(n)) return '—';
  return n.toLocaleString('en-US', { minimumFractionDigits: dp, maximumFractionDigits: dp });
}

export function fmtPct(n: number, dp = 2): string {
  if (!isFinite(n)) return '—';
  return (n * 100).toFixed(dp) + '%';
}

export function fmtSignedPct(n: number, dp = 2): string {
  if (!isFinite(n)) return '—';
  const v = (n * 100).toFixed(dp);
  return (n >= 0 ? '+' : '') + v + '%';
}

export function fmtSignedUSDT(n: number, dp = 2): string {
  if (!isFinite(n)) return '—';
  const v = Math.abs(n).toLocaleString('en-US', {
    minimumFractionDigits: dp,
    maximumFractionDigits: dp
  });
  return (n >= 0 ? '+' : '−') + v;
}

export function fmtDate(ms: number, withTime = false): string {
  if (!ms) return '—';
  const d = new Date(ms);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  if (!withTime) return `${y}-${m}-${day}`;
  const hh = String(d.getHours()).padStart(2, '0');
  const mm = String(d.getMinutes()).padStart(2, '0');
  return `${y}-${m}-${day} ${hh}:${mm}`;
}

/**
 * Holding-duration formatter. Picks the most readable unit for the magnitude.
 * 45 min, 3.2h, 1.7d, 12d — never returns mixed units.
 */
export function fmtDuration(ms: number): string {
  if (!isFinite(ms) || ms <= 0) return '—';
  const sec = ms / 1000;
  if (sec < 60) return `${sec.toFixed(0)} 秒`;
  const min = sec / 60;
  if (min < 60) return `${min.toFixed(0)} 分`;
  const hr = min / 60;
  if (hr < 24) return `${hr.toFixed(1)} 小时`;
  const day = hr / 24;
  if (day < 14) return `${day.toFixed(1)} 天`;
  return `${day.toFixed(0)} 天`;
}

export function fmtRelativeTime(ms: number): string {
  const now = Date.now();
  const diff = (now - ms) / 1000;
  if (diff < 60) return '刚刚';
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`;
  return `${Math.floor(diff / 86400)} 天前`;
}

/**
 * Removes isolated single-point spikes from a series. A point is a spike only
 * if it deviates from BOTH neighbors by more than `relThresh` in the SAME
 * direction (i.e. it pokes out and immediately reverts). Such points come from
 * deposit-timing drift in the NAV snapshots — equity and share count briefly
 * out of sync — not from real market moves (which persist into the next point,
 * so the next-neighbor deviation stays small and the point isn't flagged).
 * Spikes are replaced with the average of their neighbors.
 */
export function despike(values: number[], relThresh = 0.05): number[] {
  if (values.length < 3) return values.slice();
  const out = values.slice();
  for (let i = 1; i < values.length - 1; i++) {
    const a = values[i - 1], b = values[i], c = values[i + 1];
    if (a <= 0 || c <= 0) continue;
    const dPrev = (b - a) / a;
    const dNext = (b - c) / c;
    if (Math.abs(dPrev) > relThresh && Math.abs(dNext) > relThresh && Math.sign(dPrev) === Math.sign(dNext)) {
      out[i] = (a + c) / 2;
    }
  }
  return out;
}

export function pnlClass(n: number): string {
  if (Math.abs(n) < 1e-9) return 'text-ink-300';
  return n > 0 ? 'pos' : 'neg';
}

export function pillClass(n: number): string {
  if (Math.abs(n) < 1e-9) return 'pill-neutral';
  return n > 0 ? 'pill-pos' : 'pill-neg';
}
