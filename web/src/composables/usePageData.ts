import { getCurrentInstance, inject, onServerPrefetch, watch, type WatchSource } from "vue";
import { frameAppliersKey } from "./usePageMeta";

/**
 * Loads what a page shows. On the server the render waits for it, so the
 * page arrives whole. In the browser it runs at once and again whenever the
 * source changes, such as the filters in the address; a read the server
 * already made is not made again, because its answer came with the page.
 */
export function usePageData(load: () => Promise<unknown>, source?: WatchSource) {
  const instance = getCurrentInstance();
  const appliers = inject(frameAppliersKey, null);
  onServerPrefetch(async () => {
    await load();
    // The page's head and frame can depend on what was loaded.
    if (instance) appliers?.get(instance)?.();
  });
  if (import.meta.env.SSR) return;
  if (source) watch(source, () => void load(), { immediate: true });
  else void load();
}
