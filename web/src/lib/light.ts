// A home's light study in words and as a chart: the hours of direct sun on a
// typical day in each month, morning and afternoon.

import type { HomeDetail } from "@/api/client";
import { monthName, oneDecimal } from "./format";

type Light = HomeDetail["light"];

export function lightSummary(s: Light): string {
  return `About ${oneDecimal(s.averageDay)} hours of direct sun on an average day, most in ${monthName(s.best)} and least in ${monthName(s.worst)}.`;
}

export function morningOrEvening(s: Light): string {
  if (s.morningShare >= 0.62) return "Mostly morning light";
  if (s.morningShare <= 0.38) return "Mostly afternoon and evening light";
  return "Light through the morning and the afternoon";
}

export const chart = { w: 640, h: 210, top: 18, base: 180, left: 34 };

export interface Bar {
  month: number;
  x: number;
  w: number;
  morningY: number;
  morningH: number;
  afternoonY: number;
  afternoonH: number;
  total: number;
  morning: number;
  afternoon: number;
  labelled: boolean;
}

export function bars(s: Light): { bars: Bar[]; max: number } {
  let max = 8;
  for (const m of s.months) {
    const total = m.morning + m.afternoon;
    if (total > max) max = Math.ceil(total / 2) * 2;
  }
  const scale = (chart.base - chart.top) / max;
  const slot = (chart.w - chart.left - 8) / 12;
  const out = s.months.map((m, i): Bar => {
    const morningH = m.morning * scale;
    const morningY = chart.base - morningH;
    const afternoonH = m.afternoon * scale;
    // A 2 px gap separates the two parts of a bar.
    const afternoonY = m.morning === 0 ? chart.base - afternoonH : morningY - afternoonH - 2;
    return {
      month: i + 1,
      x: chart.left + i * slot + slot * 0.2,
      w: slot * 0.6,
      morningY,
      morningH,
      afternoonY,
      afternoonH,
      total: m.morning + m.afternoon,
      morning: m.morning,
      afternoon: m.afternoon,
      labelled: m.month === s.best || m.month === s.worst,
    };
  });
  return { bars: out, max };
}

export function chartY(v: number, max: number): number {
  return chart.base - (v * (chart.base - chart.top)) / max;
}

const f1 = (v: number) => v.toFixed(1);

/** A bar with 4 px rounded ends at the top only. */
export function barPath(x: number, y: number, w: number, h: number, roundTop: boolean): string {
  if (h <= 0) return "";
  const r = Math.min(4, w / 2, h);
  if (!roundTop) return `M${f1(x)} ${f1(y)}h${f1(w)}v${f1(h)}h-${f1(w)}z`;
  return `M${f1(x)} ${f1(y + h)}v-${f1(h - r)}q0 -${f1(r)} ${f1(r)} -${f1(r)}h${f1(w - 2 * r)}q${f1(r)} 0 ${f1(r)} ${f1(r)}v${f1(h - r)}z`;
}
