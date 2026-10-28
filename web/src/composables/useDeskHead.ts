import { useHead } from "@unhead/vue";
import { toValue, useSSRContext, type MaybeRefOrGetter } from "vue";

/** The browser tab's text on the desk: "Leads · Oriel sales desk". */
export function deskTitle(title: string): string {
  return title ? `${title} · Oriel sales desk` : "Sales desk · Oriel";
}

// The desk's styles are a file of their own, shared by its frame and its
// sign-in page. The server links a page's styles from the components it
// rendered, which do not include a shared stylesheet, so it is named here.
const DESK_STYLES = "src/styles/desk.css";

/**
 * The head of a page of the sales desk: its title and the request that it is
 * never listed. Unlike the website's pages it has no canonical address and
 * nothing for link previews, since only the team can open it.
 */
export function useDeskHead(title: MaybeRefOrGetter<string>, bodyClass = "desk") {
  if (import.meta.env.SSR) {
    const ctx = useSSRContext<{ modules?: Set<string> }>();
    if (ctx) (ctx.modules ??= new Set()).add(DESK_STYLES);
  }
  useHead({
    title: () => deskTitle(toValue(title)),
    meta: [
      { name: "robots", content: "noindex, nofollow" },
      { name: "theme-color", content: "#1a1d1f" },
      { name: "format-detection", content: "telephone=no" },
    ],
    bodyAttrs: { class: bodyClass },
  });
}
