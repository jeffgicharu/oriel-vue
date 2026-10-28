import { useHead } from "@unhead/vue";
import { computed, toValue, watchEffect, type MaybeRefOrGetter } from "vue";
import ogImage from "@/assets/img/og-oriel.png";
import { useFrameStore, type Sheet } from "@/stores/frame";

export interface PageMeta {
  /** The page's own title; the brand is added to it. Empty on the home page. */
  title?: string;
  description: string;
  /** The canonical path, such as /residences. */
  path: string;
  image?: string;
  imageAlt?: string;
  type?: string;
  jsonLd?: Record<string, unknown>[];
  sheet?: Sheet;
  /** The main navigation item the page belongs to. */
  nav?: string;
  headerOver?: boolean;
  quiet?: boolean;
  bodyClass?: string;
}

export const DEFAULT_TITLE = "Oriel · 86 homes in Riverside, Nairobi";
const DEFAULT_ALT = "Oriel: 86 homes designed around the light, in Riverside, Nairobi. Selling off-plan from KES 11.4M.";

export function fullTitle(title?: string): string {
  return title ? `${title} · Oriel, Riverside` : DEFAULT_TITLE;
}

/**
 * Sets a page's head: its title and description, its canonical address, the
 * tags link previews read, its structured data, and the request that search
 * engines leave it out until indexing is decided. The same tags are written
 * on the server, so a crawler or a link preview reads them without running
 * any script. It also tells the site's frame which section the page is in.
 */
export function usePageMeta(input: MaybeRefOrGetter<PageMeta>) {
  const frame = useFrameStore();
  const meta = computed(() => toValue(input));
  const canonical = computed(() => frame.siteUrl + (meta.value.path || "/"));
  const image = computed(() => {
    const src = meta.value.image ?? ogImage;
    return src.startsWith("http") ? src : frame.siteUrl + src;
  });
  const title = computed(() => fullTitle(meta.value.title));
  const description = computed(() => meta.value.description);
  const alt = computed(() => meta.value.imageAlt ?? DEFAULT_ALT);

  useHead({
    title,
    meta: [
      { name: "description", content: description },
      { name: "robots", content: "noindex, nofollow" },
      { property: "og:site_name", content: "Oriel" },
      { property: "og:type", content: () => meta.value.type ?? "website" },
      { property: "og:title", content: title },
      { property: "og:description", content: description },
      { property: "og:url", content: canonical },
      { property: "og:image", content: image },
      { property: "og:image:width", content: "1200" },
      { property: "og:image:height", content: "630" },
      { property: "og:image:alt", content: alt },
      { property: "og:locale", content: "en_KE" },
      { name: "twitter:card", content: "summary_large_image" },
      { name: "twitter:title", content: title },
      { name: "twitter:description", content: description },
      { name: "twitter:image", content: image },
      { name: "twitter:image:alt", content: alt },
    ],
    link: [{ rel: "canonical", href: canonical }],
    script: () =>
      (meta.value.jsonLd ?? []).map((data, i) => ({
        type: "application/ld+json" as const,
        key: `ld-${i}`,
        textContent: data,
      })),
    bodyAttrs: { class: () => meta.value.bodyClass ?? "" },
  });

  watchEffect(() => {
    const m = meta.value;
    frame.page.nav = m.nav ?? "";
    frame.page.sheet = m.sheet ?? { number: "A-000", name: "General arrangement" };
    frame.page.headerOver = m.headerOver ?? false;
    frame.page.quiet = m.quiet ?? false;
  });
}
