// @vitest-environment node
import { parse } from "devalue";
import { describe, expect, it } from "vitest";
import { escapeForScript, findsUnreachable, render } from "@/entry-server";
import type { RenderInput } from "@/render";

const today = "2026-10-28";

function viewing(ref: string, name: string) {
  return {
    ref,
    name,
    email: `${name.split(" ")[0]!.toLowerCase()}@example.com`,
    phone: "+254 712 345 678",
    start: "2026-10-30T11:00",
    minutes: 45,
    party: 2,
    notes: "",
    status: "booked",
    home: null,
    agent: "Wanjiku Kamau",
    createdAt: "2026-10-28T09:00:00Z",
  };
}

/**
 * A stand-in for the API: it answers a visitor's viewing only when their own
 * cookie comes with the request, after a pause, so two renders overlap.
 */
function fakeApi(owners: Record<string, ReturnType<typeof viewing>>): RenderInput["fetch"] {
  return async (request) => {
    const cookie = request.headers.get("cookie") ?? "";
    const url = new URL(request.url);
    await new Promise((r) => setTimeout(r, Math.random() * 30));
    const json = (status: number, body: unknown) => new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json" } });
    const m = url.pathname.match(/^\/api\/viewings\/([^/]+)$/);
    if (m) {
      const v = owners[cookie];
      return v && v.ref === m[1] ? json(200, v) : json(404, { type: "x/not-found", title: "Not found", status: 404 });
    }
    return json(404, { type: "x/not-found", title: "Not found", status: 404 });
  };
}

function input(url: string, cookie: string, fetch: RenderInput["fetch"]): RenderInput {
  return {
    url,
    siteUrl: "https://oriel-vue.example.com",
    today,
    apiBase: "http://127.0.0.1:8702",
    fetch: (request) => {
      const headers = new Headers(request.headers);
      headers.set("cookie", cookie);
      return fetch(new Request(request.url, { method: request.method, headers }));
    },
  };
}

describe("rendering on the server", () => {
  it("never lets one visitor's page see another's, even rendered at the same moment", async () => {
    const amani = viewing("V-AAAAA", "Amani Wanjiru");
    const brian = viewing("V-BBBBB", "Brian Otieno");
    const api = fakeApi({ "oriel_copy=a": amani, "oriel_copy=b": brian });
    const renders = [];
    for (let i = 0; i < 25; i++) {
      renders.push(render(input(`/viewings/${amani.ref}`, "oriel_copy=a", api)).then((out) => ({ who: "amani", out })));
      renders.push(render(input(`/viewings/${brian.ref}`, "oriel_copy=b", api)).then((out) => ({ who: "brian", out })));
    }
    for (const { who, out } of await Promise.all(renders)) {
      const [mine, theirs] = who === "amani" ? ["Amani", "Brian"] : ["Brian", "Amani"];
      expect(out.status).toBe(200);
      expect(out.html).toContain(`You’re booked, ${mine}.`);
      expect(out.html).not.toContain(theirs);
      expect(out.state).not.toContain(theirs);
    }
  });

  it("answers 404 for a viewing that is not the visitor's", async () => {
    const api = fakeApi({ "oriel_copy=a": viewing("V-AAAAA", "Amani Wanjiru") });
    const out = await render(input("/viewings/V-AAAAA", "oriel_copy=b", api));
    expect(out.status).toBe(404);
    expect(out.html).toContain("This page isn’t on the plans.");
    expect(out.html).not.toContain("Amani");
  });

  it("answers 404 for an address that is no page", async () => {
    const out = await render(input("/no-such-page", "", fakeApi({})));
    expect(out.status).toBe(404);
    expect(out.head.headTags).toContain("<title>Page not found · Oriel, Riverside</title>");
  });

  it("moves a type written in lower case to its capital", async () => {
    const out = await render(input("/residences/types/b", "", fakeApi({})));
    expect(out.status).toBe(301);
    expect(out.redirect).toBe("/residences/types/B");
  });

  it("writes the head on the server, with noindex and the canonical address", async () => {
    const out = await render(input("/terms", "", fakeApi({})));
    expect(out.head.headTags).toContain("<title>Terms of use · Oriel, Riverside</title>");
    expect(out.head.headTags).toContain('<meta name="robots" content="noindex, nofollow">');
    expect(out.head.headTags).toContain('<link rel="canonical" href="https://oriel-vue.example.com/terms">');
  });

  it("notices when the API cannot be reached, so the page is a 503", async () => {
    const down: RenderInput["fetch"] = () => Promise.reject(new TypeError("fetch failed"));
    const out = await render(input("/residences", "", down));
    expect(out.unreachable).toBe(true);
  });

  it("carries the stores to the browser in a form a script element cannot be broken out of", async () => {
    const out = await render(input("/terms", "", fakeApi({})));
    expect(out.state).not.toContain("<");
    const state = parse(out.state) as { frame: { siteUrl: string; today: string } };
    expect(state.frame).toMatchObject({ siteUrl: "https://oriel-vue.example.com", today });
  });
});

describe("escaping the state for its script element", () => {
  it("leaves nothing that could close the element or end a line in a script", () => {
    const sep = String.fromCharCode(0x2028);
    const out = escapeForScript(JSON.stringify({ x: `</script><script>alert(1)</script>${sep}` }));
    expect(out).not.toContain("<");
    expect(out).not.toContain(sep);
    expect(JSON.parse(out)).toEqual({ x: `</script><script>alert(1)</script>${sep}` });
  });

  it("finds an unreachable API anywhere in the stores", () => {
    expect(findsUnreachable({ homes: { list: { status: "error", error: { status: 0 } } } })).toBe(true);
    expect(findsUnreachable({ homes: { list: { status: "error", error: { status: 404 } } } })).toBe(false);
  });
});
