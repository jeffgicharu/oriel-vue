import type { Pinia } from "pinia";
import { START_LOCATION, type RouteMeta, type Router } from "vue-router";
import type { DeskSession } from "@/api/desk";
import type { RequestState } from "@/api/request";
import { useDeskStore } from "@/stores/desk";

/**
 * Keeps a sign-in's onward address on the desk, as Oriel's did: a path of
 * the desk's own, never another site, and never the sign-in page again.
 */
export function safeNext(next: unknown): string {
  const s = typeof next === "string" ? next : "";
  const onDesk = s === "/desk" || /^\/desk[/?#]/.test(s);
  if (!onDesk || /[\\\r\n\t]/.test(s) || s.startsWith("/desk/sign-in")) return "/desk";
  return s;
}

export type DeskDecision = { kind: "open" } | { kind: "sign-in"; next: string } | { kind: "forbidden" };

/** The part of a route the decision reads. */
export interface DeskTarget {
  meta: RouteMeta;
  name?: string | symbol | null;
  fullPath: string;
}

/**
 * What the desk does with a visit to one of its pages:
 * - a visitor who is not signed in goes to sign in, and comes back after;
 * - a member who is not the manager, on a page for the manager alone, is
 *   told so on the page itself;
 * - anyone else is let in. When the session could not be read at all, the
 *   page opens and says what went wrong, rather than asking them to sign in.
 */
export function deskDecision(to: DeskTarget, session: RequestState<DeskSession>): DeskDecision {
  if (!to.meta.desk || to.name === "desk-sign-in") return { kind: "open" };
  if (session.status === "error" && session.error?.status === 401) return { kind: "sign-in", next: to.fullPath };
  if (!session.data) return { kind: "open" };
  if (to.meta.manager && !session.data.manager) return { kind: "forbidden" };
  return { kind: "open" };
}

// The routers whose guard sent a visitor to sign in. The server makes a router
// for each page it renders, so for the server this says whether this page's
// visitor was sent, and it answers with a temporary redirect, not a permanent one.
const sentToSignIn = new WeakSet<Router>();

/** Did the desk's guard send this router's visitor to sign in? */
export function redirectedToSignIn(router: Router): boolean {
  return sentToSignIn.has(router);
}

/** Puts the desk's guard on a router. It runs on the server as well as in the browser. */
export function installDeskGuards(router: Router, pinia: Pinia) {
  router.beforeEach(async (to, from) => {
    if (!to.meta.desk || to.name === "desk-sign-in") return true;
    const desk = useDeskStore(pinia);
    // In the browser the session is asked again on each new page, so the
    // reservations badge stays current and a session that has run out goes
    // back to sign in. The page the browser takes over from the server
    // keeps the server's answer, so it matches what was sent.
    const fresh = !import.meta.env.SSR && from !== START_LOCATION && to.path !== from.path;
    await desk.loadSession(fresh);
    if (import.meta.env.SSR || !desk.now) desk.tick();
    const decision = deskDecision(to, desk.session);
    if (decision.kind === "sign-in") {
      sentToSignIn.add(router);
      return { name: "desk-sign-in", query: { next: decision.next } };
    }
    return true;
  });

  // What a change did is said once, on the page it was made on.
  router.afterEach((to, from) => {
    if (to.meta.desk && to.path !== from.path) useDeskStore(pinia).notice = "";
  });
}
