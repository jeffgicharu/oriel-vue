// How the sales desk writes times, people and money. The API sends times as
// Nairobi local strings ("2026-10-28T14:05"), and the desk's "now" is one
// too, so every sum here is plain arithmetic on those strings: never the
// runtime's clock or time zone, and the server's render matches the browser's.

import { kes, parseDay, percent } from "./format";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

/** Now in Nairobi, three hours ahead of UTC all year, to the minute. */
export function nairobiNow(at = Date.now()): string {
  return new Date(at + 3 * 3_600_000).toISOString().slice(0, 16);
}

/** Days since 1970 for a day in Nairobi. */
function dayCount(value: string): number {
  const d = parseDay(value);
  return Math.round(Date.UTC(d.year, d.month - 1, d.day) / 86_400_000);
}

/** Minutes since 1970 for a time in Nairobi, counted as plain numbers. */
function minuteCount(value: string): number {
  const d = parseDay(value);
  const [h, m] = (value.slice(11, 16) || "00:00").split(":").map(Number) as [number, number];
  return Math.round(Date.UTC(d.year, d.month - 1, d.day, h || 0, m || 0) / 60_000);
}

/** A day's date moved on or back: addDays("2026-10-28", -7) is "2026-10-21". */
export function addDays(day: string, n: number): string {
  return new Date((dayCount(day) + n) * 86_400_000).toISOString().slice(0, 10);
}

/** The Monday of the week a day is in. */
export function mondayOf(day: string): string {
  return addDays(day, -((parseDay(day).weekday + 6) % 7));
}

/** Is it a real day, written "2026-10-28"? */
export function isDay(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  return addDays(value, 0) === value;
}

/** A time on the desk: "Today 14:05", "Yesterday 09:30", "Tue 21 Oct 10:00", "21 Oct 2025". */
export function when(t: string, now: string): string {
  if (!t) return "";
  const clock = t.slice(11, 16);
  const diff = dayCount(now) - dayCount(t);
  if (diff === 0) return `Today ${clock}`;
  if (diff === 1) return `Yesterday ${clock}`;
  if (diff === -1) return `Tomorrow ${clock}`;
  const d = parseDay(t);
  if (d.year === parseDay(now).year) return `${DAYS[d.weekday]} ${d.day} ${MONTHS[d.month - 1]} ${clock}`;
  return `${d.day} ${MONTHS[d.month - 1]} ${d.year}`;
}

/** How long ago something was: "a minute", "5 hours", "3 days". */
export function ago(t: string, now: string): string {
  const minutes = minuteCount(now) - minuteCount(t);
  if (minutes < 60) return minutes <= 1 ? "a minute" : `${minutes} minutes`;
  if (minutes < 48 * 60) return `${Math.floor(minutes / 60)} hours`;
  return `${Math.floor(minutes / 1440)} days`;
}

/** A calendar day: "Tue 21 Oct". */
export function dayLabel(day: string): string {
  if (!day) return "";
  const d = parseDay(day);
  return `${DAYS[d.weekday]} ${d.day} ${MONTHS[d.month - 1]}`;
}

/** "Mon 10:00", for a viewing this week. */
export function weekdayTime(t: string): string {
  return `${DAYS[parseDay(t).weekday]} ${t.slice(11, 16)}`;
}

/** "2 Jan". */
export function dayMonth(t: string): string {
  const d = parseDay(t);
  return `${d.day} ${MONTHS[d.month - 1]}`;
}

/** The three letters of a day's name: "Mon". */
export function weekdayShort(day: string): string {
  return DAYS[parseDay(day).weekday] ?? "";
}

/** Morning, afternoon or evening, for the dashboard's greeting. */
export function partOfDay(now: string): string {
  const hour = Number(now.slice(11, 13));
  if (hour < 12) return "morning";
  if (hour < 17) return "afternoon";
  return "evening";
}

export function firstName(full: string): string {
  const i = full.indexOf(" ");
  return i < 0 ? full : full.slice(0, i);
}

/** A change against before: "3 more than the week before". */
export function trend(now: number, before: number, period: string): string {
  const d = now - before;
  if (d > 0) return `${d} more than the ${period} before`;
  if (d < 0) return `${-d} fewer than the ${period} before`;
  return `the same as the ${period} before`;
}

/** How long until a time, from the seconds left: "in 5 hours", "in 20 minutes". */
export function hoursUntil(seconds: number): string {
  if (seconds <= 0) return "now";
  const h = Math.floor(seconds / 3600 + 0.5);
  if (h >= 2) return `in ${h} hours`;
  const m = Math.floor(seconds / 60);
  return m <= 1 ? "in a minute" : `in ${m} minutes`;
}

/** When a hold runs out: "lapses in 5 hours". */
export function lapses(secondsLeft: number): string {
  return secondsLeft <= 0 ? "lapsing now" : `lapses ${hoursUntil(secondsLeft)}`;
}

/** A member's colour, or a quiet grey for anything that is not a plain hex colour. */
export function dotColour(c: string): string {
  return /^#[0-9a-fA-F]{3,6}$/.test(c) ? c : "#81878c";
}

export function plural(n: number, one: string, many: string): string {
  return n === 1 ? one : many;
}

export function phoneLink(phone: string): string {
  return phone.replace(/ /g, "");
}

export function buyingName(b: string): string {
  return b === "investment" ? "To let" : "To live in";
}

export function viewingStatus(s: string): string {
  return ({ booked: "Booked", attended: "Came", no_show: "Did not come", cancelled: "Cancelled" } as Record<string, string>)[s] ?? s;
}

export type Stage = "new" | "contacted" | "viewing" | "reservation" | "won" | "lost";
export const STAGES: Stage[] = ["new", "contacted", "viewing", "reservation", "won", "lost"];
const stageTitles: Record<Stage, string> = { new: "New", contacted: "Contacted", viewing: "Viewing", reservation: "Reservation", won: "Won", lost: "Lost" };

export function stageTitle(stage: string): string {
  return stageTitles[stage as Stage] ?? stage;
}

export function isStage(s: string): s is Stage {
  return (STAGES as string[]).includes(s);
}

/** A change in shillings with its sign: "+KES 120,000", "−KES 40,000", "no change". */
export function signedKes(n: number): string {
  if (n > 0) return "+" + kes(n);
  if (n < 0) return "−" + kes(-n);
  return "no change";
}

/** The change from old to now as a percentage: "+2.4%". */
export function changePercent(old: number, now: number): string {
  if (old === 0) return "";
  const bp = Math.trunc(((now - old) * 10_000) / old);
  return (bp < 0 ? "−" : "+") + percent(Math.abs(bp));
}

/** Basis points as a percentage for a form: 80 is "0.8". */
export function bpField(bp: number): string {
  return percent(bp).replace(/%$/, "");
}

/** An amount typed with or without commas: "18,480,000" or "KES 18480000". Null when it is not one. */
export function parseShillings(s: string): number | null {
  const clean = s.trim().replace(/,|\s|KES|kes|Ksh|_/g, "");
  if (!/^-?\d+$/.test(clean)) return null;
  const n = Number(clean);
  return Number.isSafeInteger(n) ? n : null;
}

/** A percentage typed by a person, in basis points: "0.8" is 80, "2.25%" is 225. */
export function parsePercentBP(s: string): number | null {
  const v = s.trim().replace(/%$/, "");
  const m = /^(\d+)(?:\.(\d{0,2}))?$/.exec(v);
  if (!m) return null;
  return Number(m[1]) * 100 + Number((m[2] ?? "").padEnd(2, "0") || "0");
}

/** A line of the activity log split around its subject, which the action marks with {}. */
export function activityWords(action: string): { before: string; after: string } {
  const i = action.indexOf("{}");
  if (i < 0) return { before: action + " ", after: "" };
  return { before: action.slice(0, i), after: action.slice(i + 2) };
}
