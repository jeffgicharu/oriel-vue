// How money, dates and times are written across the site. Everything here is
// plain arithmetic on the API's values, never the runtime's locale or time
// zone, so a page rendered on the server reads exactly as the browser
// renders it.

const MONTHS = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

/** Writes a whole number with commas between thousands: 18,480,000. */
export function thousands(n: number): string {
  const digits = String(Math.abs(Math.trunc(n)));
  let out = "";
  for (let i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 === 0) out += ",";
    out += digits[i];
  }
  return n < 0 ? "-" + out : out;
}

// "KES" and its number are joined by a no-break space, so a line never ends
// between them.
const KES = "KES\u00a0";

/** A whole number of shillings: "KES 17,600,000". */
export function kes(amount: number): string {
  return KES + thousands(amount);
}

/** A price for headings and labels: "KES 17.60M". Prices are whole multiples of
 * KES 10,000, so two decimals are exact and a column of them stays aligned. */
export function kesShort(amount: number): string {
  if (amount >= 1_000_000_000) {
    const h = Math.round(amount / 10_000_000);
    return `${KES}${Math.floor(h / 100)}.${String(h % 100).padStart(2, "0")}B`;
  }
  if (amount >= 1_000_000) {
    const h = Math.round(amount / 10_000);
    return `${KES}${Math.floor(h / 100)}.${String(h % 100).padStart(2, "0")}M`;
  }
  if (amount >= 1_000) return `${KES}${Math.floor(amount / 1_000)}K`;
  return kes(amount);
}

/** Basis points as a percentage: 1350 is "13.5%", 700 is "7%". */
export function percent(bp: number): string {
  const whole = Math.trunc(bp / 100);
  const frac = Math.abs(bp % 100);
  if (frac === 0) return `${whole}%`;
  if (frac % 10 === 0) return `${whole}.${frac / 10}%`;
  return `${whole}.${String(frac).padStart(2, "0")}%`;
}

interface Day {
  year: number;
  month: number; // 1 to 12
  day: number;
  weekday: number; // 0 is Sunday
}

/** Reads "2028-12-15" or "2028-12-15T10:30", a day in Nairobi. */
export function parseDay(value: string): Day {
  const [y, m, d] = value.slice(0, 10).split("-").map(Number) as [number, number, number];
  return { year: y, month: m, day: d, weekday: new Date(Date.UTC(y, m - 1, d)).getUTCDay() };
}

/** "December 2028". */
export function monthYear(value: string): string {
  const d = parseDay(value);
  return `${MONTHS[d.month - 1]} ${d.year}`;
}

/** "Dec 2028". */
export function shortMonthYear(value: string): string {
  const d = parseDay(value);
  return `${MONTHS[d.month - 1]!.slice(0, 3)} ${d.year}`;
}

/** "15 Dec 2028". */
export function shortDate(value: string): string {
  const d = parseDay(value);
  return `${d.day} ${MONTHS[d.month - 1]!.slice(0, 3)} ${d.year}`;
}

/** "Friday 15 December 2028". */
export function longDate(value: string): string {
  const d = parseDay(value);
  return `${DAYS[d.weekday]} ${d.day} ${MONTHS[d.month - 1]} ${d.year}`;
}

/** "Thursday 29 October, 10:30" from "2026-10-29T10:30". */
export function longDateTime(value: string): string {
  const d = parseDay(value);
  return `${DAYS[d.weekday]} ${d.day} ${MONTHS[d.month - 1]}, ${value.slice(11, 16)}`;
}

/** "Thu", "29", "Oct" for a day in a calendar strip. */
export function dayParts(value: string): { weekday: string; day: string; month: string } {
  const d = parseDay(value);
  return { weekday: DAYS[d.weekday]!.slice(0, 3), day: String(d.day), month: MONTHS[d.month - 1]!.slice(0, 3) };
}

/** "29.10.2026", as a date is written on a drawing. */
export function drawn(value: string): string {
  const d = parseDay(value);
  return `${String(d.day).padStart(2, "0")}.${String(d.month).padStart(2, "0")}.${d.year}`;
}

/** The name of a month, 1 to 12. */
export function monthName(month: number): string {
  return MONTHS[month - 1] ?? "";
}

/** One decimal, as hours of sun are written: "5.2". */
export function oneDecimal(n: number): string {
  return (Math.round(n * 10) / 10).toFixed(1);
}

/** "7th". */
export function ordinal(n: number): string {
  const rem100 = n % 100;
  let suffix = "th";
  if (rem100 < 11 || rem100 > 13) {
    suffix = ({ 1: "st", 2: "nd", 3: "rd" } as Record<number, string>)[n % 10] ?? "th";
  }
  return `${n}${suffix}`;
}

/** "L07". */
export function levelName(level: number): string {
  return "L" + String(level).padStart(2, "0");
}

/** "1 place" or "3 places". */
export function plural(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`;
}
