// @vitest-environment node
import { createPinia } from "pinia";
import { createApp } from "vue";
import { describe, expect, it } from "vitest";
import { apiKey, createApi } from "@/api/client";
import type { DeskSession } from "@/api/desk";
import { useDeskStore } from "./desk";

const grace: DeskSession = {
  staff: { id: 1, username: "grace.wambui", name: "Grace Wambui", role: "manager", title: "Sales manager", colour: "#1f3bd1", initials: "GW" },
  manager: true,
  waiting: 3,
};

const json = (status: number, body: unknown, type = "application/json") => new Response(JSON.stringify(body), { status, headers: { "content-type": type } });
const problem = (status: number, kind: string, title: string) => json(status, { type: `https://oriel.example/problems/${kind}`, title, status }, "application/problem+json");

/** A desk store on a stand-in for the API that knows one member and her password. */
function desk() {
  const seen: { method: string; path: string; body: unknown }[] = [];
  let signedIn = false;
  const api = createApi({
    baseUrl: "http://oriel.test",
    fetch: async (request) => {
      const path = new URL(request.url).pathname;
      const body = request.method === "POST" ? await request.clone().json() : null;
      seen.push({ method: request.method, path, body });
      if (path !== "/api/desk/session") return problem(404, "not-found", "Not found");
      if (request.method === "POST") {
        const { username, password } = body as { username: string; password: string };
        if (username !== "grace.wambui" || password !== "the password") return problem(401, "bad-sign-in", "That username and password don’t match.");
        signedIn = true;
        return json(200, grace);
      }
      if (request.method === "DELETE") {
        signedIn = false;
        return new Response(null, { status: 204 });
      }
      return signedIn ? json(200, grace) : problem(401, "signed-out", "Please sign in again.");
    },
  });
  const app = createApp({});
  app.provide(apiKey, api);
  const pinia = createPinia();
  app.use(pinia);
  return { store: useDeskStore(pinia), seen };
}

describe("the desk's session", () => {
  it("knows nobody until they sign in, and keeps the refusal for the guard", async () => {
    const { store } = desk();
    await store.loadSession();
    expect(store.session.status).toBe("error");
    expect(store.session.error).toMatchObject({ status: 401, kind: "signed-out" });
    expect(store.staff).toBeNull();
    expect(store.manager).toBe(false);
  });

  it("signs in with a username and password, and holds who signed in", async () => {
    const { store, seen } = desk();
    const signed = await store.signIn("  grace.wambui ", "the password");
    expect(signed.staff.name).toBe("Grace Wambui");
    expect(seen[0]).toEqual({ method: "POST", path: "/api/desk/session", body: { username: "grace.wambui", password: "the password" } });
    expect(store.session).toMatchObject({ status: "ready", key: "session", error: null });
    expect(store.staff?.initials).toBe("GW");
    expect(store.manager).toBe(true);
    expect(store.waiting).toBe(3);
    // The session is already known, so the guard does not ask again.
    await store.loadSession();
    expect(seen).toHaveLength(1);
  });

  it("turns wrong details down without signing anyone in", async () => {
    const { store } = desk();
    await expect(store.signIn("grace.wambui", "not the password")).rejects.toMatchObject({ status: 401, kind: "bad-sign-in", title: "That username and password don’t match." });
    expect(store.staff).toBeNull();
  });

  it("signs out, forgetting who it was and what the last change did", async () => {
    const { store, seen } = desk();
    await store.signIn("grace.wambui", "the password");
    store.notice = "Note saved.";
    await store.signOut();
    expect(seen.at(-1)).toMatchObject({ method: "DELETE", path: "/api/desk/session" });
    expect(store.session).toMatchObject({ status: "idle", data: null });
    expect(store.staff).toBeNull();
    expect(store.notice).toBe("");
    await store.loadSession();
    expect(store.session.error?.status).toBe(401);
  });

  it("keeps its clock in Nairobi, whatever the machine's time zone", () => {
    const { store } = desk();
    store.tick(Date.UTC(2026, 9, 28, 21, 30));
    expect(store.now).toBe("2026-10-29T00:30");
    expect(store.today).toBe("2026-10-29");
  });
});
