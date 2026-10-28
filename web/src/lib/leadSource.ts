// Where a visitor came from: the campaign in the link they followed, or the
// site that sent them. The first arrival is kept for 90 days; the last is the
// arrival that started this visit. Both travel with every enquiry, viewing,
// reservation and WhatsApp tap, and the API works out the channel from them.
//
// Only this browser keeps them, and only these few fields: no identifier, no
// tracking script, nothing sent anywhere until the visitor sends a form.

import type { Source, Touch } from "@/api/client";

const FIRST = "oriel.source.first";
const LAST = "oriel.source.last";
const KEEP_FIRST_DAYS = 90;

/** The touch for this arrival, from the address and the referring page. */
export function touchFrom(url: URL, referrer: string, now: Date): Touch {
  const p = url.searchParams;
  let ref = "";
  try {
    if (referrer && new URL(referrer).host !== url.host) ref = referrer;
  } catch {
    // An unreadable referrer is no referrer.
  }
  return {
    source: (p.get("utm_source") ?? "").slice(0, 60),
    medium: (p.get("utm_medium") ?? "").slice(0, 60),
    campaign: (p.get("utm_campaign") ?? "").slice(0, 80),
    referrer: ref.slice(0, 300),
    landing: url.pathname.slice(0, 120),
    at: now.toISOString(),
  };
}

function hasCampaign(t: Touch): boolean {
  return t.source !== "" || t.medium !== "" || t.campaign !== "";
}

function readTouch(storage: Storage | undefined, key: string): Touch | null {
  try {
    const raw = storage?.getItem(key);
    if (!raw) return null;
    const t = JSON.parse(raw) as Touch;
    return typeof t.at === "string" && typeof t.landing === "string" ? t : null;
  } catch {
    return null;
  }
}

function writeTouch(storage: Storage | undefined, key: string, t: Touch) {
  try {
    storage?.setItem(key, JSON.stringify(t));
  } catch {
    // Storage may be unavailable; the forms then send this arrival alone.
  }
}

function storages(): { local?: Storage; session?: Storage } {
  try {
    return { local: window.localStorage, session: window.sessionStorage };
  } catch {
    return {};
  }
}

/**
 * Records this arrival. The first touch is kept unless it is older than 90
 * days. The last touch is set at the start of each visit, and again whenever
 * the visitor arrives through a new campaign link.
 */
export function recordArrival(url: URL, referrer: string, now = new Date()): void {
  const { local, session } = storages();
  const touch = touchFrom(url, referrer, now);
  const first = readTouch(local, FIRST);
  if (!first || now.getTime() - Date.parse(first.at) > KEEP_FIRST_DAYS * 86_400_000) writeTouch(local, FIRST, touch);
  const last = readTouch(session, LAST);
  if (!last || hasCampaign(touch) || (touch.referrer !== "" && touch.referrer !== last.referrer)) writeTouch(session, LAST, touch);
}

/** Where the visitor came from, for a form to send. */
export function currentSource(): Source | undefined {
  if (typeof window === "undefined") return undefined;
  const { local, session } = storages();
  const last = readTouch(session, LAST) ?? readTouch(local, FIRST);
  const first = readTouch(local, FIRST) ?? last;
  return first && last ? { first, last } : undefined;
}
