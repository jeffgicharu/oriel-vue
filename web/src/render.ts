// What the web server hands the renderer for one page, and what it gets
// back. Kept apart from the renderer so the server can be type-checked
// without the Vue app.

export interface RenderInput {
  /** The path and query asked for. */
  url: string;
  siteUrl: string;
  /** Today in Nairobi, "2026-10-28". */
  today: string;
  /** The API on 127.0.0.1, such as http://127.0.0.1:8702. */
  apiBase: string;
  /** Calls the API for this visitor: their cookies, their address, a time limit. */
  fetch: (request: Request) => Promise<Response>;
}

export interface RenderOutput {
  status: number;
  /** Set when the address moved: the page is not rendered, the browser is sent on. */
  redirect?: string;
  html: string;
  head: { headTags: string; bodyTags: string; bodyTagsOpen: string; htmlAttrs: string; bodyAttrs: string };
  /** The stores as JSON, escaped for a data block in the page. */
  state: string;
  /** The components the page used, for the scripts and styles to ask for early. */
  modules: Set<string>;
  /** The API could not be reached for something the page needs. */
  unreachable: boolean;
}
