import { expect, test, type Locator, type Page } from "@playwright/test";
import { availableHome, buyer, hydrated } from "./helpers";

// The sample's sales team, who share one password: the sales manager and one of the agents.
const PASSWORD = "Riverside2028";
const manager = { username: "grace.wambui", first: "Grace" };
const agent = { username: "daniel.otieno", first: "Daniel" };

/** Signs in through the desk's own form, and waits for the page it goes on to. */
async function signIn(page: Page, who: { username: string }, next = "/desk") {
  await page.goto(`/desk/sign-in?next=${encodeURIComponent(next)}`);
  await hydrated(page);
  await page.getByLabel("Username").fill(who.username);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Sign in" }).click();
  const path = next.split("?")[0];
  await page.waitForURL((url) => url.pathname === path);
  await hydrated(page);
}

/** Drags with the pointer, in small steps, as a person does. */
async function drag(page: Page, from: Locator, to: Locator) {
  await from.scrollIntoViewIfNeeded();
  const a = await from.boundingBox();
  const b = await to.boundingBox();
  expect(a && b, "both ends of the drag are on the page").toBeTruthy();
  const x = a!.x + 24;
  const y = a!.y + a!.height - 8;
  await page.mouse.move(x, y);
  await page.mouse.down();
  await page.mouse.move(x + 12, y + 4, { steps: 4 });
  await page.mouse.move(b!.x + Math.min(b!.width / 2, 60), b!.y + 24, { steps: 24 });
  await page.mouse.move(b!.x + Math.min(b!.width / 2, 60) + 2, b!.y + 26, { steps: 2 });
  await page.mouse.up();
}

test.describe("signing in to the desk", () => {
  test("a visitor who is not signed in is sent to sign in, and back to the page after", async ({ page }) => {
    const res = await page.goto("/desk/leads?view=table");
    expect((await res?.request().redirectedFrom()?.response())?.status(), "a temporary redirect, not a permanent one").toBe(302);
    await expect(page).toHaveURL(/\/desk\/sign-in\?next=/);
    await expect(page).toHaveTitle("Sign in · Oriel sales desk");
    await hydrated(page);

    await page.getByLabel("Username").fill(manager.username);
    await page.getByLabel("Password").fill("not the password");
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page.getByRole("alert")).toContainText("That username and password don’t match.");

    await page.getByLabel("Password").fill(PASSWORD);
    await page.getByRole("button", { name: "Sign in" }).click();
    await expect(page).toHaveURL(/\/desk\/leads\?view=table$/);
    await expect(page.getByRole("heading", { level: 1, name: "Leads" })).toBeVisible();
    await expect(page.locator("table.desk-table thead")).toContainText("Source");
    await expect(page).toHaveTitle("Leads · Oriel sales desk");
  });
});

test.describe("the sales manager's day", () => {
  test("the dashboard says where leads come from", async ({ page }) => {
    await signIn(page, manager);
    await expect(page.getByRole("heading", { level: 1, name: new RegExp(`^Good (morning|afternoon|evening), ${manager.first}\\.$`) })).toBeVisible();
    await expect(page.locator(".desk-kpi").first()).toContainText("Available");
    await expect(page.getByRole("img", { name: /homes:/ })).toBeVisible();

    const channels = page.locator("section", { has: page.getByRole("heading", { name: "Where leads come from" }) });
    await expect(channels.getByRole("img", { name: /Leads from the website by channel/ })).toBeVisible();
    await expect(channels.getByRole("table")).toContainText("Paid social");
    await expect(page.getByRole("heading", { name: "Taps on WhatsApp" })).toBeVisible();
    // On a phone the menu is folded away, so it is read rather than looked at.
    await expect(page.locator(".desk-nav a", { hasText: "Pricing" })).toHaveCount(1);
  });

  test("a home on the stacking plan opens in its drawer, and closes", async ({ page }) => {
    await signIn(page, manager, "/desk/stack");
    const cell = page.locator(".stack-cell.is-available").first();
    const id = await cell.getAttribute("data-unit");
    await cell.click();
    await expect(page).toHaveURL(new RegExp(`/desk/stack\\?home=${id}$`));
    const drawer = page.locator("#drawer");
    await expect(drawer.getByRole("heading", { level: 2, name: `Home ${id}` })).toBeVisible();
    await expect(drawer.getByRole("heading", { name: "Withhold from sale" })).toBeVisible();
    await expect(drawer.getByRole("heading", { name: "History" })).toBeVisible();
    await expect(cell).toHaveAttribute("aria-current", "true");

    // The address opens it again, drawn on the server.
    await page.reload();
    await expect(drawer.getByRole("heading", { level: 2, name: `Home ${id}` })).toBeVisible();
    await hydrated(page);
    await drawer.getByRole("button", { name: "Close" }).click();
    await expect(page).toHaveURL(/\/desk\/stack$/);
    await expect(drawer.locator(".desk-home")).toHaveCount(0);
  });

  test("a lead dragged on the board moves to its new stage", async ({ page }) => {
    await signIn(page, manager, "/desk/leads");
    const stages = ["new", "contacted", "viewing", "reservation"];
    let from = "";
    for (const s of stages) {
      if ((await page.locator(`[data-stage="${s}"] .lead-card`).count()) > 0) {
        from = s;
        break;
      }
    }
    expect(from, "a lead to move").not.toBe("");
    const to = stages[stages.indexOf(from) + 1]!;
    const card = page.locator(`[data-stage="${from}"] .lead-card`).first();
    const id = await card.getAttribute("data-id");

    // The board scrolls sideways on a narrow screen: the lead's column goes to
    // the left edge, so the next one is in view beside it.
    await page.locator(".board").evaluate((board, stage) => {
      const col = board.querySelector(`[data-stage="${stage}"]`)?.closest(".board__col");
      if (col) board.scrollLeft += col.getBoundingClientRect().left - board.getBoundingClientRect().left;
    }, from);

    const saved = page.waitForResponse((r) => r.url().includes(`/api/desk/leads/${id}/move`) && r.request().method() === "POST");
    await drag(page, card, page.locator(`ol[data-stage="${to}"]`));
    expect((await saved).status()).toBe(204);
    await expect(page.locator(`[data-stage="${to}"] [data-id="${id}"]`)).toBeVisible();

    await page.reload();
    await expect(page.locator(`[data-stage="${to}"] [data-id="${id}"]`)).toBeVisible();
  });

  test("a lead moves on the board with the keyboard alone", async ({ page }) => {
    await signIn(page, manager, "/desk/leads");
    const stages = ["new", "contacted", "viewing", "reservation"];
    let from = "";
    for (const s of stages) {
      if ((await page.locator(`[data-stage="${s}"] .lead-card`).count()) > 1) {
        from = s;
        break;
      }
    }
    expect(from, "a stage with two leads").not.toBe("");
    const to = stages[stages.indexOf(from) + 1]!;
    const toTitle = (await page.locator(`.board__col:has(ol[data-stage="${to}"]) .board__title`).textContent())!.trim();
    const card = page.locator(`[data-stage="${from}"] .lead-card`).first();
    const id = await card.getAttribute("data-id");
    const name = (await card.locator(".lead-card__name").textContent())!.trim();
    const moved = page.locator(`.lead-card[data-id="${id}"]`);

    const move = moved.getByRole("button", { name: `Move ${name}` });
    await move.focus();
    await page.keyboard.press("Enter");
    await expect(move).toHaveAttribute("aria-expanded", "true");
    const steps = page.getByRole("group", { name: `Move ${name}` });

    let saved = page.waitForResponse((r) => r.url().includes(`/api/desk/leads/${id}/move`));
    await steps.getByRole("button", { name: "Down" }).focus();
    await page.keyboard.press("Enter");
    expect((await saved).status()).toBe(204);
    await expect(page.locator(`ol[data-stage="${from}"] > li`).nth(1)).toHaveAttribute("data-id", id!);
    await expect(page.getByRole("status").filter({ hasText: `${name} moved down` })).toHaveCount(1);

    saved = page.waitForResponse((r) => r.url().includes(`/api/desk/leads/${id}/move`));
    await steps.getByRole("button", { name: `To ${toTitle}` }).focus();
    await page.keyboard.press("Enter");
    expect((await saved).status()).toBe(204);
    await expect(page.locator(`ol[data-stage="${to}"] > li`).first()).toHaveAttribute("data-id", id!);
    await expect(moved.getByRole("button", { name: `Move ${name}` })).toBeFocused();

    await page.reload();
    await expect(page.locator(`ol[data-stage="${to}"] > li`).first()).toHaveAttribute("data-id", id!);
  });

  test("a reservation made on the website is confirmed on the desk", async ({ page, request }) => {
    // The buyer's side, in the same browser: the desk works on what this browser sent.
    const home = await availableHome(request);
    await page.goto(`/reserve/${home.id}`);
    await hydrated(page);
    await page.getByLabel("Full name, as it will appear on the letter of offer").fill(buyer.name);
    await page.getByLabel("Phone").fill(buyer.phone);
    await page.getByLabel("Email").fill(buyer.email);
    await page.getByLabel(/I understand this is a request/).check();
    await page.getByRole("button", { name: "Request the reservation" }).click();
    await expect(page).toHaveURL(/\/reservations\/R-[A-Z0-9]+\?new=1$/);
    const ref = new URL(page.url()).pathname.split("/").pop()!;

    await signIn(page, manager, "/desk/reservations");
    const card = page.locator(`[id="${ref}"]`);
    await expect(card).toContainText(buyer.name);
    await expect(card).toContainText(home.id);
    await expect(page.locator(".desk-nav__badge")).toHaveText(/^[1-9]\d* waiting$/);

    page.once("dialog", (d) => void d.accept());
    await card.getByRole("button", { name: "Confirm and issue the offer" }).click();
    await expect(page).toHaveURL(new RegExp(`/desk/reservations\\?tab=confirmed#${ref}$`));
    await expect(page.locator(".desk-notice")).toContainText(`Reservation ${ref} confirmed.`);
    await expect(page.locator(`[id="${ref}"]`).getByRole("link", { name: "Letter of offer (PDF)" })).toHaveAttribute("href", `/api/reservations/${ref}/offer.pdf`);
    await expect(page.locator(`[id="${ref}"]`).getByLabel("Deposit received, in KES")).toBeVisible();
  });
});

test.describe("an agent's desk", () => {
  test("pricing is for the sales manager", async ({ page }) => {
    await signIn(page, agent);
    await expect(page.locator(".desk-nav a", { hasText: "Pricing" })).toHaveCount(0);
    const res = await page.goto("/desk/pricing");
    expect(res?.status()).toBe(403);
    await expect(page.getByRole("heading", { level: 1, name: "For the sales manager" })).toBeVisible();
    await expect(page).toHaveTitle("Managers only · Oriel sales desk");
    await expect(page.getByRole("button", { name: "Preview the change" })).toHaveCount(0);
  });
});
