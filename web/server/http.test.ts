import { describe, expect, it } from "vitest";
import { allowedCookies, canonicalPath, clientAddress, nairobiToday, pageHeaders, preloadLinks, staticHeaders } from "./http";

describe("what the web server passes on", () => {
  it("forwards only the visitor's copy and the desk's session to the API", () => {
    expect(allowedCookies("theme=dark; oriel_copy=abc.sig; _ga=GA1.2; oriel_desk=tok")).toBe("oriel_copy=abc.sig; oriel_desk=tok");
    expect(allowedCookies("oriel_copy_x=nope; xoriel_copy=nope")).toBe("");
    expect(allowedCookies(undefined)).toBe("");
  });

  it("takes the visitor's address from nginx", () => {
    expect(clientAddress({ headers: { "x-real-ip": "198.51.100.7" }, socket: { remoteAddress: "127.0.0.1" } } as never)).toBe("198.51.100.7");
    expect(clientAddress({ headers: {}, socket: { remoteAddress: "127.0.0.1" } } as never)).toBe("127.0.0.1");
  });

  it("never lets a page be kept by a shared cache, and asks search engines not to list it", () => {
    const h = pageHeaders();
    expect(h["Cache-Control"]).toBe("private, no-store");
    expect(h["X-Robots-Tag"]).toBe("noindex, nofollow");
    expect(h["Content-Security-Policy"]).toContain("frame-ancestors 'none'");
    expect(h["Content-Security-Policy"]).not.toContain("unsafe-eval");
    expect(h["Content-Security-Policy"]).toMatch(/script-src 'self' https:\/\/static\.cloudflareinsights\.com;/);
  });

  it("keeps fingerprinted files for a year and the rest for an hour", () => {
    expect(staticHeaders("/assets/index-abc123.js")["Cache-Control"]).toBe("public, max-age=31536000, immutable");
    expect(staticHeaders("/favicon.svg")["Cache-Control"]).toBe("public, max-age=3600");
    expect(staticHeaders("/assets/x.woff2")["Content-Type"]).toBe("font/woff2");
  });

  it("sends a page's other spellings to its one address, and never off the site", () => {
    expect(canonicalPath("/residences")).toBeNull();
    expect(canonicalPath("/")).toBeNull();
    expect(canonicalPath("/residences/")).toBe("/residences");
    expect(canonicalPath("/residences//0504/")).toBe("/residences/0504");
    expect(canonicalPath("//evil.example/")).toBe("/evil.example");
    expect(canonicalPath("///")).toBe("/");
  });

  it("knows the day in Nairobi", () => {
    expect(nairobiToday(new Date("2026-10-28T20:59:00Z"))).toBe("2026-10-28");
    expect(nairobiToday(new Date("2026-10-28T21:01:00Z"))).toBe("2026-10-29");
  });

  it("asks early for each script and style a page used, once", () => {
    const links = preloadLinks(new Set(["src/pages/A.vue", "src/pages/B.vue"]), {
      "src/pages/A.vue": ["/assets/A.js", "/assets/shared.js", "/assets/A.css"],
      "src/pages/B.vue": ["/assets/shared.js"],
    });
    expect(links).toBe('<link rel="modulepreload" crossorigin href="/assets/A.js"><link rel="modulepreload" crossorigin href="/assets/shared.js"><link rel="stylesheet" href="/assets/A.css">');
  });
});
