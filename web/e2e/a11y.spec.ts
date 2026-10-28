import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";
import { hydrated } from "./helpers";

const pages = ["/", "/explore", "/residences", "/residences/0704", "/residences/1601", "/residences/types/B", "/compare?homes=0704,0806", "/building", "/location", "/progress", "/buying", "/viewings/new", "/enquire", "/privacy", "/terms", "/shortlist", "/desk/sign-in", "/no-such-page"];

// The sales desk's pages, as the sales manager sees them, with every part shown.
const deskPages = [
  "/desk",
  "/desk/stack",
  "/desk/stack?home=0704",
  "/desk/leads",
  "/desk/leads?view=table",
  "/desk/leads/new",
  "/desk/viewings",
  "/desk/reservations",
  "/desk/reservations?tab=confirmed",
  "/desk/pricing",
  "/desk/activity",
];

/** Opens a page, checks the browser took it over as it arrived, and runs axe on it. */
async function check(page: Page, path: string) {
  const mismatches: string[] = [];
  page.on("console", (m) => {
    if ((m.type() === "error" || m.type() === "warning") && /hydrat|mismatch/i.test(m.text())) mismatches.push(m.text().slice(0, 600));
  });
  await page.goto(path);
  await hydrated(page);
  expect(mismatches, "the page the server sent matches the one the browser renders").toEqual([]);
  // Nothing on the page makes it scroll sideways, on a phone or a laptop.
  for (const width of [390, 1280]) {
    await page.setViewportSize({ width, height: 900 });
    // The elements past the right edge that are not inside something that scrolls on its own.
    const past = await page.evaluate(() => {
      const edge = document.documentElement.clientWidth;
      if (document.documentElement.scrollWidth <= edge) return [];
      // The body clips sideways overflow for the page as a whole, so it is not counted.
      const free = (e: Element) => {
        for (let p = e.parentElement; p && p !== document.body; p = p.parentElement) if (getComputedStyle(p).overflowX !== "visible") return false;
        return true;
      };
      const found = [...document.querySelectorAll("body *")]
        .filter((e) => e.getBoundingClientRect().right > edge && free(e))
        .slice(0, 4)
        .map((e) => `${e.tagName.toLowerCase()}.${[...e.classList].join(".")} to ${e.getBoundingClientRect().right.toFixed(1)}`);
      return found.length > 0 ? found : [`the page, ${document.documentElement.scrollWidth - edge} px too wide`];
    });
    expect(past, `no sideways scroll at ${width} px`).toEqual([]);
  }
  // Let the drawings finish tracing themselves, so colours are checked as seen.
  await page.emulateMedia({ reducedMotion: "reduce" });
  const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"]).analyze();
  // Each failure names the first few elements it was found on.
  const where = (v: (typeof results.violations)[number]) => v.nodes.slice(0, 3).map((n) => n.target.join(" ")).join(", ");
  expect(results.violations.map((v) => `${v.id}: ${v.nodes.length} × ${v.help} (${where(v)})`)).toEqual([]);
}

test.describe("accessibility", () => {
  // One engine is enough for the rules axe checks; the journeys run in all of them.
  test.skip(({ browserName }) => browserName !== "chromium", "checked in Chromium");

  for (const path of pages) {
    test(`${path} has no accessibility failures, and the browser takes it over as it arrived`, async ({ page }) => {
      await check(page, path);
    });
  }
});

test.describe("accessibility of the sales desk", () => {
  test.skip(({ browserName }) => browserName !== "chromium", "checked in Chromium");

  // Signed in as the sales manager through the API, in the page's own browser.
  test.beforeEach(async ({ page }) => {
    const res = await page.request.post("/api/desk/session", { data: { username: "grace.wambui", password: "Riverside2028" } });
    expect(res.ok(), "the sales manager signs in").toBeTruthy();
  });

  for (const path of deskPages) {
    test(`${path} has no accessibility failures, and the browser takes it over as it arrived`, async ({ page }) => {
      await check(page, path);
    });
  }

  test("a lead's page has no accessibility failures, and the browser takes it over as it arrived", async ({ page }) => {
    await page.goto("/desk/leads?view=table");
    const href = await page.locator("tbody th a").first().getAttribute("href");
    expect(href).toMatch(/^\/desk\/leads\/\d+$/);
    await check(page, href!);
  });
});
