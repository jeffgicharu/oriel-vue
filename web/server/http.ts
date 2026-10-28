// The web server's small decisions, kept apart so they can be tested: which
// cookies reach the API, which address is the visitor's, which headers go
// with each answer, and which files a page asks for early.

import type http from "node:http";

/** The only cookies passed on to the API: the visitor's copy and the desk's session. */
export const FORWARDED_COOKIES = ["oriel_copy", "oriel_desk"];

/** Keeps only the allowed cookies from a Cookie header. */
export function allowedCookies(header: string | undefined): string {
  if (!header) return "";
  return header
    .split(";")
    .map((part) => part.trim())
    .filter((part) => FORWARDED_COOKIES.includes(part.slice(0, part.indexOf("=")).trim()))
    .join("; ");
}

/**
 * The visitor's address. nginx sets X-Real-IP from Cloudflare's header; the
 * server listens on 127.0.0.1, so nothing but nginx can set it.
 */
export function clientAddress(req: Pick<http.IncomingMessage, "headers" | "socket">): string {
  const real = req.headers["x-real-ip"];
  const value = Array.isArray(real) ? real[0] : real;
  return value || req.socket.remoteAddress || "";
}

/**
 * The policy for pages: only Oriel's own scripts, styles, fonts and pictures,
 * plus Cloudflare's page-speed beacon. The stores' state travels in a JSON
 * data block, which runs nothing, so no inline script is allowed. Styles may
 * be inline because the drawings and the 3D view set element styles.
 */
export const CONTENT_SECURITY_POLICY =
  "default-src 'self'; " +
  "script-src 'self' https://static.cloudflareinsights.com; " +
  "style-src 'self' 'unsafe-inline'; " +
  "img-src 'self' data: blob:; " +
  "font-src 'self'; " +
  "connect-src 'self' https://cloudflareinsights.com; " +
  "worker-src 'self' blob:; " +
  "object-src 'none'; " +
  "frame-ancestors 'none'; " +
  "base-uri 'self'; " +
  "form-action 'self'";

const security = {
  "Content-Security-Policy": CONTENT_SECURITY_POLICY,
  "X-Content-Type-Options": "nosniff",
  "X-Frame-Options": "DENY",
  "Referrer-Policy": "strict-origin-when-cross-origin",
  "Strict-Transport-Security": "max-age=31536000",
  "Permissions-Policy": "camera=(), microphone=(), geolocation=(), payment=()",
  "Cross-Origin-Opener-Policy": "same-origin",
  "X-Robots-Tag": "noindex, nofollow",
};

/**
 * Headers for a page. Every page may hold a visitor's own bookings or the
 * desk's records, so none may ever be kept by Cloudflare, nginx or the
 * browser's cache and handed to anyone else.
 */
export function pageHeaders(): Record<string, string> {
  return { ...security, "Cache-Control": "private, no-store" };
}

const TYPES: Record<string, string> = {
  ".js": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".woff2": "font/woff2",
  ".avif": "image/avif",
  ".webp": "image/webp",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".txt": "text/plain; charset=utf-8",
  ".json": "application/json",
};

/** Headers for a file: fingerprinted files never change; the rest are kept an hour. */
export function staticHeaders(urlPath: string): Record<string, string> {
  const ext = urlPath.slice(urlPath.lastIndexOf("."));
  return {
    ...security,
    "Content-Type": TYPES[ext] ?? "application/octet-stream",
    "Cache-Control": urlPath.startsWith("/assets/") ? "public, max-age=31536000, immutable" : "public, max-age=3600",
  };
}

/**
 * The one address a page answers at, when the request used another: no
 * trailing slash and no doubled slashes. Null when the path is already it.
 * The result always starts with a single slash, so it can never point at
 * another site.
 */
export function canonicalPath(urlPath: string): string | null {
  const clean = "/" + urlPath.split("/").filter(Boolean).join("/");
  return clean === urlPath ? null : clean;
}

/** Today's date in Nairobi, three hours ahead of UTC all year. */
export function nairobiToday(now = new Date()): string {
  return new Date(now.getTime() + 3 * 3_600_000).toISOString().slice(0, 10);
}

/**
 * Links asking the browser for the scripts and styles of the components a
 * page rendered, so they arrive alongside the page instead of after it.
 */
export function preloadLinks(modules: Set<string>, manifest: Record<string, string[]>): string {
  const seen = new Set<string>();
  let out = "";
  for (const id of modules) {
    for (const file of manifest[id] ?? []) {
      if (seen.has(file)) continue;
      seen.add(file);
      if (file.endsWith(".js")) out += `<link rel="modulepreload" crossorigin href="${file}">`;
      else if (file.endsWith(".css")) out += `<link rel="stylesheet" href="${file}">`;
    }
  }
  return out;
}
