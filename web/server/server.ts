// Oriel's web server. It renders every page on the server from the API, so a
// visitor, a crawler or a link preview gets the whole page before any script
// runs, and the browser carries on from there as a single-page app.
//
// It listens on 127.0.0.1 only; nginx is the way in. It holds no data and no
// secrets: everything it shows comes from its own slot's API on 127.0.0.1.
//
//   node server.mjs           serve (ORIEL_WEB_PORT, ORIEL_API_PORT, ORIEL_SITE_URL)
//   node server.mjs version   print the release's commit id

import { readdirSync, readFileSync, statSync } from "node:fs";
import http from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import type { RenderInput, RenderOutput } from "../src/render";
import { allowedCookies, clientAddress, nairobiToday, pageHeaders, preloadLinks, staticHeaders } from "./http";

declare const __RELEASE__: string;
const release = __RELEASE__;

if (process.argv[2] === "version") {
  console.log(release);
  process.exit(0);
}

const here = path.dirname(fileURLToPath(import.meta.url));
const clientDir = path.join(here, "client");

const port = Number(process.env.ORIEL_WEB_PORT);
const apiPort = Number(process.env.ORIEL_API_PORT);
const siteUrl = (process.env.ORIEL_SITE_URL ?? "").replace(/\/+$/, "");
// In CI the web server stands in for nginx and passes /api on to the API.
const proxyApi = process.env.ORIEL_PROXY_API === "1";
if (!Number.isInteger(port) || port < 1024 || !Number.isInteger(apiPort) || apiPort < 1024 || !/^https?:\/\/[^/]+$/.test(siteUrl)) {
  console.error("oriel-web: ORIEL_WEB_PORT, ORIEL_API_PORT and ORIEL_SITE_URL must be set");
  process.exit(64);
}
const apiBase = `http://127.0.0.1:${apiPort}`;

// How long the API may take for one read, and a page for its whole render.
const API_TIMEOUT_MS = 3_000;
const RENDER_TIMEOUT_MS = 8_000;

const template = readFileSync(path.join(clientDir, "index.html"), "utf8");
const ssrManifest = JSON.parse(readFileSync(path.join(clientDir, ".vite", "ssr-manifest.json"), "utf8")) as Record<string, string[]>;
const unavailablePage = readFileSync(path.join(here, "..", "unavailable.html"), "utf8");
const errorPage = unavailablePage
  .replace(/<title>[^<]*<\/title>/, "<title>Something went wrong · Oriel</title>")
  .replace(/<h1>[^<]*<\/h1>/, "<h1>Something went wrong.</h1>")
  .replace(/<p>[^<]*<\/p>/, "<p>Something went wrong on our side. Please try again in a moment.</p>");
const { render } = (await import(path.join(here, "ssr", "entry-server.js"))) as { render: (input: RenderInput) => Promise<RenderOutput> };

// The files the browser loads: fingerprinted ones under /assets, and the few
// with fixed names, such as the icons. Read once, when the server starts.
const files = new Map<string, string>();
(function walk(dir: string) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name !== ".vite") walk(full);
    } else if (full !== path.join(clientDir, "index.html")) {
      files.set("/" + path.relative(clientDir, full).split(path.sep).join("/"), full);
    }
  }
})(clientDir);

function sendFile(req: http.IncomingMessage, res: http.ServerResponse, urlPath: string): boolean {
  const file = files.get(urlPath);
  if (!file) return false;
  const body = readFileSync(file);
  res.writeHead(200, { ...staticHeaders(urlPath), "Content-Length": statSync(file).size });
  res.end(req.method === "HEAD" ? undefined : body);
  return true;
}

function sendText(req: http.IncomingMessage, res: http.ServerResponse, status: number, type: string, body: string, extra: Record<string, string> = {}) {
  res.writeHead(status, { ...pageHeaders(), "Content-Type": type, ...extra });
  res.end(req.method === "HEAD" ? undefined : body);
}

/** Calls the API from the server, with a time limit. */
async function apiGet(apiPath: string, signal?: AbortSignal): Promise<Response> {
  const limit = AbortSignal.timeout(API_TIMEOUT_MS);
  return fetch(apiBase + apiPath, { signal: signal ? AbortSignal.any([signal, limit]) : limit });
}

async function healthz(req: http.IncomingMessage, res: http.ServerResponse) {
  // The release scripts only believe a web server that can reach its own API.
  try {
    const r = await apiGet("/api/healthz");
    if (!r.ok) throw new Error(String(r.status));
    sendText(req, res, 200, "application/json", JSON.stringify({ release, api: r.headers.get("x-release") }), { "X-Release": release });
  } catch {
    sendText(req, res, 503, "application/json", JSON.stringify({ release, api: null }));
  }
}

function robots(req: http.IncomingMessage, res: http.ServerResponse) {
  const lines = ["# Every page also carries noindex.", "User-agent: *"];
  for (const p of ["/desk", "/api/", "/viewings/", "/reservations/", "/reserve/", "/compare", "/shortlist"]) lines.push("Disallow: " + p);
  lines.push("", `Sitemap: ${siteUrl}/sitemap.xml`, "");
  sendText(req, res, 200, "text/plain; charset=utf-8", lines.join("\n"), { "Cache-Control": "public, max-age=3600" });
}

async function sitemap(req: http.IncomingMessage, res: http.ServerResponse) {
  try {
    const r = await apiGet("/api/homes?show=all");
    if (!r.ok) throw new Error(String(r.status));
    const list = (await r.json()) as { homes: { id: string }[] };
    const today = nairobiToday();
    const paths = ["/", "/residences", "/explore", "/building", "/location", "/progress", "/buying", "/viewings/new", "/enquire", "/privacy", "/terms"];
    for (const code of ["A", "B", "C", "D", "P"]) paths.push("/residences/types/" + code);
    for (const h of list.homes) paths.push("/residences/" + h.id);
    const urls = paths.map((p) => `  <url>\n    <loc>${siteUrl}${p}</loc>\n    <lastmod>${today}</lastmod>\n  </url>`).join("\n");
    const body = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`;
    sendText(req, res, 200, "application/xml; charset=utf-8", body, { "Cache-Control": "public, max-age=3600" });
  } catch {
    sendText(req, res, 503, "text/html; charset=utf-8", unavailablePage, { "Retry-After": "60" });
  }
}

function manifest(req: http.IncomingMessage, res: http.ServerResponse) {
  const body = {
    name: "Oriel, Riverside",
    short_name: "Oriel",
    description: "Eighty-six homes designed around the light in Riverside, Nairobi.",
    start_url: "/",
    scope: "/",
    display: "standalone",
    background_color: "#ecedea",
    theme_color: "#ecedea",
    lang: "en-KE",
    icons: [
      { src: "/icons/icon-192.png", sizes: "192x192", type: "image/png" },
      { src: "/icons/icon-512.png", sizes: "512x512", type: "image/png" },
      { src: "/icons/icon-maskable-512.png", sizes: "512x512", type: "image/png", purpose: "maskable" },
    ],
  };
  sendText(req, res, 200, "application/manifest+json", JSON.stringify(body), { "Cache-Control": "public, max-age=3600" });
}

/** Passes a request on to the API unchanged, streaming both ways. Only in CI. */
function proxy(req: http.IncomingMessage, res: http.ServerResponse) {
  const upstream = http.request(
    { host: "127.0.0.1", port: apiPort, method: req.method, path: req.url, headers: { ...req.headers, "x-real-ip": clientAddress(req) } },
    (answer) => {
      res.writeHead(answer.statusCode ?? 502, answer.headers);
      answer.pipe(res);
    },
  );
  upstream.on("error", () => {
    if (!res.headersSent) sendText(req, res, 503, "text/html; charset=utf-8", unavailablePage);
    else res.destroy();
  });
  req.pipe(upstream);
}

async function page(req: http.IncomingMessage, res: http.ServerResponse, url: string) {
  const cookies = allowedCookies(req.headers.cookie);
  const address = clientAddress(req);
  const setCookies: string[] = [];
  const deadline = AbortSignal.timeout(RENDER_TIMEOUT_MS);

  // Every read the page makes goes to this slot's API as this visitor: their
  // copy's cookie and the desk's session, nothing else they sent, and their
  // address, so the API's limits apply to them and not to this server.
  const visitorFetch = async (request: Request): Promise<Response> => {
    const headers = new Headers({ accept: "application/json", "x-real-ip": address });
    if (cookies) headers.set("cookie", cookies);
    const response = await fetch(request.url, {
      method: request.method,
      headers,
      signal: AbortSignal.any([deadline, AbortSignal.timeout(API_TIMEOUT_MS)]),
    });
    setCookies.push(...response.headers.getSetCookie());
    return response;
  };

  let out: RenderOutput;
  try {
    out = await Promise.race([
      render({ url, siteUrl, today: nairobiToday(), apiBase, fetch: visitorFetch }),
      new Promise<never>((_, reject) => deadline.addEventListener("abort", () => reject(new Error("the render took too long")), { once: true })),
    ]);
  } catch (e) {
    console.error("render failed", url, e instanceof Error ? e.message : e);
    sendText(req, res, 500, "text/html; charset=utf-8", errorPage);
    return;
  }
  if (out.unreachable) {
    sendText(req, res, 503, "text/html; charset=utf-8", unavailablePage, { "Retry-After": "60" });
    return;
  }
  if (out.redirect) {
    res.writeHead(out.status, { ...pageHeaders(), Location: out.redirect });
    res.end();
    return;
  }
  const html = template
    .replace("<!--html-attrs-->", out.head.htmlAttrs ? " " + out.head.htmlAttrs : "")
    .replace("<!--head-->", preloadLinks(out.modules, ssrManifest) + out.head.headTags)
    .replace("<!--body-attrs-->", out.head.bodyAttrs ? " " + out.head.bodyAttrs : "")
    .replace("<!--app-->", out.html)
    .replace("<!--state-->", `<script type="application/json" id="oriel-state">${out.state}</script>${out.head.bodyTags}`);
  const headers: http.OutgoingHttpHeaders = { ...pageHeaders(), "Content-Type": "text/html; charset=utf-8" };
  if (setCookies.length) headers["Set-Cookie"] = setCookies;
  res.writeHead(out.status, headers);
  res.end(req.method === "HEAD" ? undefined : html);
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url ?? "/", "http://localhost");
  const p = url.pathname;
  if (proxyApi && p.startsWith("/api/")) return proxy(req, res);
  if (req.method !== "GET" && req.method !== "HEAD") {
    sendText(req, res, 405, "text/plain; charset=utf-8", "Method not allowed", { Allow: "GET, HEAD" });
    return;
  }
  if (p === "/healthz") return void healthz(req, res);
  if (p === "/robots.txt") return robots(req, res);
  if (p === "/sitemap.xml") return void sitemap(req, res);
  if (p === "/site.webmanifest") return manifest(req, res);
  if (sendFile(req, res, p)) return;
  if (p.startsWith("/assets/") || p.startsWith("/api/")) {
    sendText(req, res, 404, "text/plain; charset=utf-8", "Not found");
    return;
  }
  void page(req, res, url.pathname + url.search);
});

server.headersTimeout = 10_000;
server.requestTimeout = 30_000;
// Longer than nginx keeps its idle connections to us, so nginx closes first.
server.keepAliveTimeout = 65_000;

server.listen(port, "127.0.0.1", () => {
  console.log(`serving on 127.0.0.1:${port}, release ${release}, API ${apiBase}`);
});

// On SIGTERM, finish the requests in hand and stop.
for (const signal of ["SIGTERM", "SIGINT"] as const) {
  process.on(signal, () => {
    server.close(() => process.exit(0));
    server.closeIdleConnections();
    setTimeout(() => process.exit(0), 15_000).unref();
  });
}
