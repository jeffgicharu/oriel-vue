import { createMemoryHistory, createRouter, createWebHistory, type RouterScrollBehavior } from "vue-router";
import { routes } from "./routes";

// The header stays in view, so a jump to a part of a page stops below it.
const HEADER_CLEARANCE = 88;

/**
 * Where the page scrolls to after a move:
 * - back and forward return to where the visitor was;
 * - a link to a part of a page goes to that part;
 * - a change of filters, which only changes the query, stays where it is;
 * - anything else starts at the top.
 */
export const scrollBehavior: RouterScrollBehavior = (to, from, saved) => {
  if (saved) return saved;
  if (to.hash) return { el: to.hash, top: HEADER_CLEARANCE, behavior: "smooth" };
  if (to.path === from.path) return false;
  return { top: 0 };
};

/** A router for one app: in memory while rendering on the server, on the browser's history after. */
export function makeRouter(ssr: boolean) {
  return createRouter({
    history: ssr ? createMemoryHistory() : createWebHistory(),
    routes,
    scrollBehavior,
    // Links to the page on show are marked for screen readers and styles.
    linkActiveClass: "",
    linkExactActiveClass: "",
  });
}
