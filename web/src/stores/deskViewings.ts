// The sales suite's agenda: the viewings of a week, what came of each, and
// for the manager the opening hours and the days the suite is closed.

import { defineStore } from "pinia";
import { reactive } from "vue";
import { unwrap, useApi } from "@/api/client";
import type { DeskAgenda } from "@/api/desk";
import { idle, load } from "@/api/request";

export const useDeskViewingsStore = defineStore("deskViewings", () => {
  const api = useApi();
  const agenda = reactive(idle<DeskAgenda>());

  function loadAgenda(from: string, to: string, force = false) {
    return load(agenda, `${from}:${to}`, () => unwrap(api.GET("/api/desk/viewings", { params: { query: { from, to } } })), force);
  }

  async function recordOutcome(ref: string, outcome: "attended" | "no_show", notes: string) {
    const done = await unwrap(api.POST("/api/desk/viewings/{ref}/outcome", { params: { path: { ref } }, body: { outcome, notes } }));
    const list = agenda.data?.viewings ?? [];
    const i = list.findIndex((v) => v.ref === ref);
    if (i >= 0) list[i] = done.viewing;
    return done;
  }

  async function saveHours(days: number[], times: string[], capacity: number) {
    const saved = await unwrap(api.PUT("/api/desk/viewings/hours", { body: { days, times, capacity } }));
    if (agenda.data) agenda.data.hours = saved.hours;
    return saved;
  }

  async function closeDay(day: string, reason: string) {
    const done = await unwrap(api.POST("/api/desk/viewings/blocks", { body: { day, reason } }));
    if (agenda.data) agenda.data.blocks = done.blocks;
    return done;
  }

  async function openDay(day: string) {
    const done = await unwrap(api.DELETE("/api/desk/viewings/blocks/{day}", { params: { path: { day } } }));
    if (agenda.data) agenda.data.blocks = done.blocks;
    return done;
  }

  return { agenda, loadAgenda, recordOutcome, saveHours, closeDay, openDay };
});
