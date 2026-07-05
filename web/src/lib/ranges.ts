// Time-range presets for the 收益对比 (fund vs 大盘) module's range switcher.
// Shared between the chart component (labels) and the page (window → fetch).

export type RangeKey = '30d' | '3m' | '6m' | '1y' | 'all';

export const RANGES: { key: RangeKey; label: string; days: number | null }[] = [
  { key: '30d', label: '近30天', days: 30 },
  { key: '3m', label: '近3月', days: 90 },
  { key: '6m', label: '近半年', days: 180 },
  { key: '1y', label: '近1年', days: 365 },
  { key: 'all', label: '全部', days: null }
];

// Lower bound (unix ms) to request for a range. 'all' returns 1 (≈ epoch) rather
// than 0 so the query param is still sent — the api client omits falsy `from`,
// and an omitted `from` makes the backend fall back to its 30-day default.
export function rangeFromMs(key: RangeKey, now: number = Date.now()): number {
  const r = RANGES.find((x) => x.key === key);
  if (!r || r.days === null) return 1;
  return now - r.days * 86400000;
}
