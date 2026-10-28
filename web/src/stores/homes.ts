// The homes: the residences list for the filters in the address, each home's
// page, the types, and homes side by side. Homes already read are kept, so
// going back to one is instant.

import { defineStore } from "pinia";
import { reactive } from "vue";
import { unwrap, useApi, type HomeDetail, type HomeList, type Plan, type Schedule, type TypeDetail } from "@/api/client";
import { idle, load, type RequestState } from "@/api/request";
import { filterQuery, type HomeFilter } from "@/lib/filters";

export const useHomesStore = defineStore("homes", () => {
  const api = useApi();
  const list = reactive(idle<HomeList>());
  const details = reactive<Record<string, RequestState<HomeDetail>>>({});
  const types = reactive<Record<string, RequestState<TypeDetail>>>({});
  const compared = reactive(idle<HomeDetail[]>());
  const plans = reactive(idle<Record<"A" | "B" | "C" | "D" | "P", Plan>>());
  const shortlisted = reactive(idle<HomeList>());

  /** The homes on a visitor's shortlist, in its order. */
  function loadShortlisted(ids: string[], force = false) {
    const key = ids.join(",");
    return load(shortlisted, key, () => unwrap(api.GET("/api/homes", { params: { query: { ids: key } } })), force);
  }

  /** Every type's plan, for the drawings on the residences list. */
  function loadPlans() {
    return load(plans, "plans", () => unwrap(api.GET("/api/plans")));
  }

  function loadList(filter: HomeFilter, force = false) {
    const query = filterQuery(filter);
    const key = new URLSearchParams(Object.entries(query).map(([k, v]) => [k, String(v)])).toString();
    return load(list, key, () => unwrap(api.GET("/api/homes", { params: { query } })), force);
  }

  function detail(id: string): RequestState<HomeDetail> {
    details[id] ??= idle<HomeDetail>();
    return details[id];
  }

  function loadHome(id: string, force = false) {
    return load(detail(id), id, () => unwrap(api.GET("/api/homes/{id}", { params: { path: { id } } })), force);
  }

  function typeOf(code: string): RequestState<TypeDetail> {
    types[code] ??= idle<TypeDetail>();
    return types[code];
  }

  function loadType(code: string, force = false) {
    return load(typeOf(code), code, () => unwrap(api.GET("/api/types/{code}", { params: { path: { code: code as "A" } } })), force);
  }

  function loadCompared(ids: string[], force = false) {
    const key = ids.join(",");
    if (!key) {
      compared.key = "";
      compared.status = "ready";
      compared.data = [];
      compared.error = null;
      return Promise.resolve();
    }
    return load(compared, key, async () => (await unwrap(api.GET("/api/compare", { params: { query: { homes: key } } }))).homes, force);
  }

  /** One plan's schedule with the buyer's own mortgage rate and term. */
  function schedule(id: string, plan: "staged" | "cash" | "mortgage", rate?: number, years?: number): Promise<Schedule> {
    return unwrap(api.GET("/api/homes/{id}/schedule", { params: { path: { id }, query: { plan, rate, years } } }));
  }

  return { list, details, types, compared, plans, shortlisted, loadShortlisted, loadPlans, loadList, detail, loadHome, typeOf, loadType, loadCompared, schedule };
});
