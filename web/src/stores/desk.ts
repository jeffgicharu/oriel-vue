// The sales desk: who is signed in, the time its pages count from, what the
// last change did, and the pages every member reads: the dashboard, the team
// and the activity log.

import { defineStore } from "pinia";
import { computed, reactive, ref } from "vue";
import { unwrap, useApi } from "@/api/client";
import type { DeskActivity, DeskDashboard, DeskSession, DeskStaff } from "@/api/desk";
import { idle, load } from "@/api/request";
import { nairobiNow } from "@/lib/desk";

export type ActivityKind = "" | "staff" | "visitor" | "system";

export const useDeskStore = defineStore("desk", () => {
  const api = useApi();
  const session = reactive(idle<DeskSession>());
  // Now in Nairobi, to the minute. The server sets it as it renders and the
  // browser takes it over with the page, so both write "3 hours ago" alike;
  // once the page is running, the browser moves it on each minute.
  const now = ref("");
  // What the last change did, shown at the top of the page.
  const notice = ref("");
  const team = reactive(idle<DeskStaff[]>());
  const dashboard = reactive(idle<DeskDashboard>());
  const activity = reactive(idle<DeskActivity>());

  const staff = computed(() => session.data?.staff ?? null);
  const manager = computed(() => session.data?.manager === true);
  const waiting = computed(() => session.data?.waiting ?? 0);
  const today = computed(() => now.value.slice(0, 10));

  function tick(at = Date.now()) {
    now.value = nairobiNow(at);
  }

  /** Who is signed in. A refusal (signed out) is kept, so the guard can send them to sign in. */
  function loadSession(force = false) {
    return load(session, "session", () => unwrap(api.GET("/api/desk/session")), force);
  }

  async function signIn(username: string, password: string): Promise<DeskSession> {
    const signed = await unwrap(api.POST("/api/desk/session", { body: { username: username.trim(), password } }));
    Object.assign(session, { status: "ready", key: "session", data: signed, error: null });
    return signed;
  }

  async function signOut() {
    await unwrap(api.DELETE("/api/desk/session"));
    Object.assign(session, idle<DeskSession>());
    notice.value = "";
  }

  function loadTeam(force = false) {
    return load(team, "team", async () => (await unwrap(api.GET("/api/desk/team"))).team, force);
  }

  function loadDashboard(force = false) {
    return load(dashboard, "dashboard", () => unwrap(api.GET("/api/desk/dashboard")), force);
  }

  function loadActivity(kind: ActivityKind, offset: number, force = false) {
    const query = { ...(kind ? { kind } : {}), ...(offset > 0 ? { offset } : {}) };
    return load(activity, `${kind}:${offset}`, () => unwrap(api.GET("/api/desk/activity", { params: { query } })), force);
  }

  return { session, now, notice, team, dashboard, activity, staff, manager, waiting, today, tick, loadSession, signIn, signOut, loadTeam, loadDashboard, loadActivity };
});
