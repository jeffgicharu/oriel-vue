import { createHead, renderSSRHead } from "@unhead/vue/server";
import { stringify } from "devalue";
import { renderToString, type SSRContext } from "vue/server-renderer";
import { createApi } from "./api/client";
import { createOriel } from "./app";
import type { RenderInput, RenderOutput } from "./render";
import { redirectedToSignIn } from "./router/guards";
import { useFrameStore } from "./stores/frame";

export type { RenderInput, RenderOutput };

/** Escapes JSON for a script element: nothing in it can end the element. */
export function escapeForScript(json: string): string {
  return json.replace(/</g, "\\u003C").replace(/\u2028/g, "\\u2028").replace(/\u2029/g, "\\u2029");
}

/** Does any request in the stores' state say the API could not be reached? */
export function findsUnreachable(value: unknown, depth = 0): boolean {
  if (depth > 6 || value === null || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  if (v.status === "error" && typeof v.error === "object" && v.error !== null && (v.error as { status?: unknown }).status === 0) return true;
  return Object.values(v).some((child) => findsUnreachable(child, depth + 1));
}

/** Renders one page for one visitor, with a fresh app. */
export async function render(input: RenderInput): Promise<RenderOutput> {
  const api = createApi({ baseUrl: input.apiBase, fetch: input.fetch });
  const { app, pinia, router } = createOriel(api, true);
  const head = createHead();
  app.use(head);

  const frame = useFrameStore(pinia);
  frame.siteUrl = input.siteUrl;
  frame.today = input.today;
  frame.release = __RELEASE__;

  await router.push(input.url);
  await router.isReady();
  const route = router.currentRoute.value;
  if (route.redirectedFrom) {
    // A page that has moved, such as a type written in lower case, has moved
    // for good. Being sent to sign in depends on who is asking, so it is not.
    const status = redirectedToSignIn(router) ? 302 : 301;
    return { status, redirect: route.fullPath, html: "", head: emptyHead, state: "", modules: new Set(), unreachable: false };
  }

  const ctx: SSRContext & { status?: number; modules?: Set<string> } = { status: route.name === "not-found" ? 404 : 200 };
  const html = await renderToString(app, ctx);
  const tags = renderSSRHead(head);
  return {
    status: ctx.status ?? 200,
    html,
    head: tags,
    state: escapeForScript(stringify(pinia.state.value)),
    modules: ctx.modules ?? new Set(),
    unreachable: findsUnreachable(pinia.state.value),
  };
}

const emptyHead = { headTags: "", bodyTags: "", bodyTagsOpen: "", htmlAttrs: "", bodyAttrs: "" };
