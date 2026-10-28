// The building as it stands: the overview every page's frame reads, the
// construction programme, and the tower's geometry for the drawings.

import { defineStore } from "pinia";
import { reactive } from "vue";
import { unwrap, useApi, type BuildingModel, type Overview, type Progress, type Schedule } from "@/api/client";
import { idle, load } from "@/api/request";

export const useSiteStore = defineStore("site", () => {
  const api = useApi();
  const overview = reactive(idle<Overview>());
  const progress = reactive(idle<Progress>());
  const model = reactive(idle<BuildingModel>());
  const examples = reactive(idle<Schedule[]>());
  const privacy = reactive(idle<{ hasCopy: boolean }>());

  /** Each payment plan worked out on the buying page's example price. */
  function loadExamples(price: number) {
    return load(examples, String(price), async () => (await unwrap(api.GET("/api/payment-plans", { params: { query: { price } } }))).schedules);
  }

  /** Whether this browser has sent anything there is to delete. */
  function loadPrivacy(force = false) {
    return load(privacy, "privacy", () => unwrap(api.GET("/api/privacy")), force);
  }

  function loadOverview(force = false) {
    return load(overview, "overview", () => unwrap(api.GET("/api/overview")), force);
  }

  function loadProgress(force = false) {
    return load(progress, "progress", () => unwrap(api.GET("/api/progress")), force);
  }

  function loadModel(force = false) {
    return load(model, "model", () => unwrap(api.GET("/api/building")), force);
  }

  return { overview, progress, model, examples, privacy, loadOverview, loadProgress, loadModel, loadExamples, loadPrivacy };
});
