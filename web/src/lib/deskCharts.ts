// The desk's charts, laid out as numbers for SVG: every home by status, the
// homes sold each month, a bar for each label, and the website's leads by
// channel. The sizes are Oriel's, so the charts read as they did there.

import { kes } from "./format";

/** A number for an SVG attribute, to one decimal place. */
export function n1(v: number): string {
  return String(Math.round(v * 10) / 10);
}

export const STATUS_ORDER = ["sold", "reserved", "held", "available", "withheld"] as const;
export type StatusKey = (typeof STATUS_ORDER)[number];

const statusWords: Record<StatusKey, string> = { sold: "Sold", reserved: "Reserved", held: "On hold", available: "Available", withheld: "Not yet released" };

export interface StatusSegment {
  status: StatusKey;
  name: string;
  n: number;
  x: number;
  w: number;
}

/** Every home as one bar, divided by status. */
export function statusBar(counts: Record<string, number>, total: number, width = 1000) {
  const segments: StatusSegment[] = [];
  let x = 0;
  for (const status of STATUS_ORDER) {
    const n = counts[status] ?? 0;
    if (n === 0 || total === 0) continue;
    const bw = (width * n) / total;
    segments.push({ status, name: statusWords[status], n, x, w: Math.max(bw - 2, 1) });
    x += bw;
  }
  const parts = segments.map((s) => `${s.n} ${s.name.toLowerCase()}`);
  return { segments, summary: `${total} homes: ${parts.join(", ")}` };
}

export interface MonthBar {
  label: string;
  n: number;
  value: number;
  x: number;
  y: number;
  w: number;
  h: number;
  current: boolean;
}

export const monthChart = { w: 560, h: 220, top: 26, base: 190 };

/** Sales by month, the latest month drawn solid. */
export function monthBars(months: { label: string; sold: number; value: number }[]) {
  const { w, top, base } = monthChart;
  const max = Math.max(1, ...months.map((m) => m.sold));
  const step = w / Math.max(months.length, 1);
  const bar = step * 0.56;
  const bars: MonthBar[] = months.map((m, i) => {
    const h = ((base - top) * m.sold) / max;
    return { label: m.label, n: m.sold, value: m.value, x: step * i + (step - bar) / 2, y: base - h, w: bar, h: Math.max(h, 0), current: i === months.length - 1 };
  });
  return { bars, summary: `Homes sold each month: ${months.map((m) => `${m.label} ${m.sold}`).join(", ")}` };
}

export function monthTitle(b: MonthBar): string {
  return `${b.label}: ${b.n} sold, ${kes(b.value)}`;
}

export interface HBar {
  label: string;
  n: number;
  y: number;
  w: number;
}

export const hbarChart = { w: 560, row: 34, labelW: 150 };

/** A horizontal bar for each label, in the order given. */
export function hBars(items: { label: string; n: number }[]) {
  const { w, row, labelW } = hbarChart;
  const max = Math.max(1, ...items.map((it) => it.n));
  const bars: HBar[] = items.map((it, i) => ({ label: it.label, n: it.n, y: row * i, w: Math.max(((w - labelW - 50) * it.n) / max, 1) }));
  return { bars, height: row * items.length, summary: items.map((it) => `${it.label} ${it.n}`).join(", ") };
}

export interface ChannelRow {
  channel: string;
  leads: number;
  viewed: number;
  reserved: number;
}

export const SERIES = [
  { key: "leads", label: "Leads", className: "chart__bar" },
  { key: "viewed", label: "Viewed", className: "chart__bar chart__bar--viewed" },
  { key: "reserved", label: "Reserved", className: "chart__bar chart__bar--reserved" },
] as const;

export const channelChart = { w: 560, labelW: 128, bar: 9, gap: 2, pad: 14 };

/**
 * The website's leads by channel: for each, three bars side by side, for the
 * leads that came, those who viewed and those who asked to reserve, all on
 * one scale so the drop from one to the next reads at a glance.
 */
export function channelBars(rows: ChannelRow[]) {
  const { w, labelW, bar, gap, pad } = channelChart;
  const max = Math.max(1, ...rows.map((r) => r.leads), ...rows.map((r) => r.viewed), ...rows.map((r) => r.reserved));
  const groupH = SERIES.length * bar + (SERIES.length - 1) * gap;
  const row = groupH + pad;
  const span = w - labelW - 44;
  const groups = rows.map((r, i) => {
    const top = row * i + pad / 2;
    return {
      channel: r.channel,
      mid: top + groupH / 2,
      bars: SERIES.map((s, j) => ({ ...s, n: r[s.key], y: top + j * (bar + gap), w: r[s.key] > 0 ? Math.max((span * r[s.key]) / max, 1.5) : 0 })),
    };
  });
  const summary = rows.map((r) => `${r.channel}: ${r.leads} leads, ${r.viewed} viewed, ${r.reserved} reserved`).join("; ");
  return { groups, height: row * rows.length, summary };
}
