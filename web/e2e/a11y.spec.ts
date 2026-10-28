import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { hydrated } from "./helpers";

const pages = ["/", "/residences", "/residences/0704", "/residences/1601", "/residences/types/B", "/compare?homes=0704,0806", "/building", "/location", "/progress", "/buying", "/viewings/new", "/enquire", "/privacy", "/terms", "/shortlist", "/no-such-page"];

test.describe("accessibility", () => {
  // One engine is enough for the rules axe checks; the journeys run in all of them.
  test.skip(({ browserName }) => browserName !== "chromium", "checked in Chromium");

  for (const path of pages) {
    test(`${path} has no accessibility failures`, async ({ page }) => {
      await page.goto(path);
      await hydrated(page);
      // Let the drawings finish tracing themselves, so colours are checked as seen.
      await page.emulateMedia({ reducedMotion: "reduce" });
      const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"]).analyze();
      expect(results.violations.map((v) => `${v.id}: ${v.nodes.length} × ${v.help}`)).toEqual([]);
    });
  }
});
