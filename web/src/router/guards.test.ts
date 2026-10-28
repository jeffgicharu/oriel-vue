// @vitest-environment node
import { createPinia } from "pinia";
import { createApp, defineComponent } from "vue";
import { createMemoryHistory, createRouter, type RouteMeta } from "vue-router";
import { describe, expect, it } from "vitest";
import { apiKey, createApi } from "@/api/client";
import type { DeskSession } from "@/api/desk";
import { idle, type RequestState } from "@/api/request";
import { deskDecision, installDeskGuards, redirectedToSignIn, safeNext } from "./guards";

const member = (manager: boolean): DeskSession => ({
  staff: { id: manager ? 1 : 2, username: "x", name: manager ? "Grace Wambui" : "Daniel Otieno", role: manager ? "manager" : "agent", title: "", colour: "#2e6a43", initials: "" },
  manager,
  waiting: 0,
});

const ready = (s: DeskSession): RequestState<DeskSession> => ({ status: "ready", key: "session", data: s, error: null });
const refused = (status: number): RequestState<DeskSession> => ({
  status: "error",
  key: "session",
  data: null,
  error: { status, kind: status === 401 ? "signed-out" : "busy", title: "", detail: "", fields: {} },
});
const to = (fullPath: string, meta: RouteMeta, name = "desk-leads") => ({ fullPath, meta, name });

describe("who may open a page of the desk", () => {
  it("sends a visitor who is not signed in to sign in, and back to the page after", () => {
    expect(deskDecision(to("/desk/leads?view=table&stage=new", { desk: true }), refused(401))).toEqual({ kind: "sign-in", next: "/desk/leads?view=table&stage=new" });
  });

  it("tells an agent that pricing is for the sales manager, on the page itself", () => {
    expect(deskDecision(to("/desk/pricing", { desk: true, manager: true }, "desk-pricing"), ready(member(false)))).toEqual({ kind: "forbidden" });
    expect(deskDecision(to("/desk/pricing", { desk: true, manager: true }, "desk-pricing"), ready(member(true)))).toEqual({ kind: "open" });
    expect(deskDecision(to("/desk/leads", { desk: true }), ready(member(false)))).toEqual({ kind: "open" });
  });

  it("never guards the sign-in page or the website", () => {
    expect(deskDecision(to("/desk/sign-in", { desk: true }, "desk-sign-in"), refused(401))).toEqual({ kind: "open" });
    expect(deskDecision(to("/residences", {}, "residences"), refused(401))).toEqual({ kind: "open" });
  });

  it("opens the page to say what went wrong when the session could not be read at all", () => {
    expect(deskDecision(to("/desk", { desk: true }, "desk"), refused(503))).toEqual({ kind: "open" });
    expect(deskDecision(to("/desk", { desk: true }, "desk"), idle<DeskSession>())).toEqual({ kind: "open" });
  });
});

describe("where a sign-in goes next", () => {
  it("keeps a page of the desk, with its filters", () => {
    expect(safeNext("/desk")).toBe("/desk");
    expect(safeNext("/desk/leads?view=table&source=Referral")).toBe("/desk/leads?view=table&source=Referral");
    expect(safeNext("/desk/stack?home=0704")).toBe("/desk/stack?home=0704");
  });

  it("goes to the dashboard for anything else", () => {
    for (const next of ["", "https://example.com/desk", "//example.com/desk", "/desktop", "/residences", "/desk\\..\\x", "/desk/\r\nx", "/desk/sign-in?next=/desk", undefined, ["/desk/leads"], 42]) {
      expect(safeNext(next)).toBe("/desk");
    }
  });
});

describe("the guard on a router", () => {
  const Page = defineComponent({ render: () => null });

  function routerFor(session: (() => Response) | null) {
    const api = createApi({
      baseUrl: "http://oriel.test",
      fetch: async () => (session ? session() : new Response(JSON.stringify({ type: "x/signed-out", title: "Please sign in again.", status: 401 }), { status: 401, headers: { "content-type": "application/problem+json" } })),
    });
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: "/", name: "home", component: Page },
        { path: "/desk/sign-in", name: "desk-sign-in", component: Page, meta: { desk: true } },
        { path: "/desk/leads", name: "desk-leads", component: Page, meta: { desk: true } },
        { path: "/desk/pricing", name: "desk-pricing", component: Page, meta: { desk: true, manager: true } },
      ],
    });
    const pinia = createPinia();
    installDeskGuards(router, pinia);
    const app = createApp(Page);
    app.provide(apiKey, api);
    app.use(pinia);
    app.use(router);
    return router;
  }

  it("moves a signed-out visitor to sign in, marked so the server answers 302", async () => {
    const router = routerFor(null);
    await router.push("/desk/leads?view=table");
    expect(router.currentRoute.value.name).toBe("desk-sign-in");
    expect(router.currentRoute.value.query.next).toBe("/desk/leads?view=table");
    expect(router.currentRoute.value.redirectedFrom?.fullPath).toBe("/desk/leads?view=table");
    expect(redirectedToSignIn(router)).toBe(true);
  });

  it("lets a member in, and an agent onto pricing to read that it is the manager's", async () => {
    const agent = () => new Response(JSON.stringify(member(false)), { status: 200, headers: { "content-type": "application/json" } });
    const router = routerFor(agent);
    await router.push("/desk/pricing");
    expect(router.currentRoute.value.name).toBe("desk-pricing");
    expect(redirectedToSignIn(router)).toBe(false);
  });

  it("leaves the website alone", async () => {
    const router = routerFor(null);
    await router.push("/");
    expect(router.currentRoute.value.name).toBe("home");
    expect(redirectedToSignIn(router)).toBe(false);
  });
});
