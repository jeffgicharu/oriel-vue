import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { defaultFilter, filterRouteQuery, isNarrowed, parseFilter, type HomeFilter } from "@/lib/filters";

/**
 * The residences filters, read from and written to the address. Changing a
 * filter replaces the address rather than adding to the history, so the back
 * button leaves the list instead of undoing each click.
 */
export function useHomeFilters() {
  const route = useRoute();
  const router = useRouter();
  const filter = computed(() => parseFilter(route.query));
  const narrowed = computed(() => isNarrowed(filter.value));

  function update(patch: Partial<HomeFilter>) {
    return router.replace({ query: filterRouteQuery({ ...filter.value, ...patch }), hash: route.hash });
  }

  function reset() {
    return router.replace({ query: filterRouteQuery({ ...defaultFilter, sort: filter.value.sort }) });
  }

  return { filter, narrowed, update, reset };
}
