import { expect, test } from "@playwright/test";
import { availableHome, buyer, hydrated } from "./helpers";

test.describe("finding a home", () => {
  test("the residences filters live in the address and survive a reload", async ({ page }) => {
    await page.goto("/residences");
    await hydrated(page);
    await expect(page.getByRole("heading", { level: 1, name: "Find your home." })).toBeVisible();
    await page.locator("label.chip", { hasText: /^2$/ }).click();
    await expect(page).toHaveURL(/\?beds=2$/);
    const rows = page.locator("tbody tr");
    await expect(rows.first()).toBeVisible();
    for (const type of await rows.locator("td[data-label=Type]").allTextContents()) expect(type).toContain("2 bed");

    await page.getByLabel("Side").selectOption("river");
    await expect(page).toHaveURL(/beds=2/);
    await expect(page).toHaveURL(/side=river/);
    await page.reload();
    await expect(page.getByLabel("Side")).toHaveValue("river");
    await expect(page.locator("label.chip", { hasText: /^2$/ }).locator("input")).toBeChecked();

    await page.getByRole("button", { name: "Clear filters" }).click();
    await expect(page).toHaveURL(/\/residences$/);
  });

  test("a home's page shows its plan, light and what each plan costs", async ({ page, request }) => {
    const home = await availableHome(request, 2);
    await page.goto(`/residences/${home.id}`);
    await hydrated(page);
    await expect(page.getByRole("heading", { level: 1 })).toContainText(home.title);
    await expect(page.getByRole("img", { name: `Floor plan of home ${home.id}, Two bedrooms` })).toBeVisible();

    await page.getByLabel("Furniture").uncheck();
    await expect(page.locator(".plan-figure")).toHaveClass(/hide-furniture/);

    const total = page.locator(".schedule-card__total");
    const staged = await total.textContent();
    await page.locator("label.payplan", { hasText: "Cash" }).click();
    await expect(page).toHaveURL(/plan=cash/);
    await expect(total).not.toHaveText(staged ?? "");

    await page.locator("label.payplan", { hasText: "Mortgage" }).click();
    const monthly = page.locator(".mortgage__monthly");
    const before = await monthly.textContent();
    await page.getByLabel("Term").selectOption("10");
    await expect(monthly).not.toHaveText(before ?? "");
  });

  test("a home that does not exist is a real 404", async ({ page }) => {
    const res = await page.goto("/residences/9999");
    expect(res?.status()).toBe(404);
    await expect(page.getByRole("heading", { level: 1, name: "This page isn’t on the plans." })).toBeVisible();
  });
});

test.describe("getting in touch", () => {
  test("booking a viewing, adding it to a calendar and cancelling it", async ({ page, request }) => {
    const home = await availableHome(request);
    await page.goto(`/viewings/new?home=${home.id}`);
    await hydrated(page);
    await expect(page.getByText("You’re coming to see")).toBeVisible();

    // Sent empty, the form says what is missing, at the top and beside each field.
    await page.getByRole("button", { name: "Book this viewing" }).click();
    const summary = page.locator("#viewing-errors");
    await expect(summary).toBeFocused();
    await expect(summary).toContainText("Please choose a time.");
    await expect(summary).toContainText("Please tell us your name.");

    await page.locator(".slot:not(.is-full) input").first().check();
    await page.getByLabel("Full name").fill(buyer.name);
    await page.getByLabel("Phone").fill(buyer.phone);
    await page.getByLabel("Email").fill(buyer.email);
    await page.getByRole("button", { name: "Book this viewing" }).click();

    await expect(page).toHaveURL(/\/viewings\/V-[A-Z0-9]+\?booked=1$/);
    await expect(page.getByRole("heading", { level: 1, name: "You’re booked, Test." })).toBeVisible();
    await expect(page.getByRole("link", { name: "Add to calendar" })).toHaveAttribute("href", /\/api\/viewings\/V-[A-Z0-9]+\/calendar\.ics$/);

    // The booking is this browser's own: a page loaded afresh still shows it.
    await page.reload();
    await expect(page.getByRole("heading", { level: 1, name: "You’re booked, Test." })).toBeVisible();

    page.once("dialog", (d) => void d.accept());
    await page.getByRole("button", { name: "Cancel this viewing" }).click();
    await expect(page.getByRole("heading", { level: 1, name: "Viewing cancelled." })).toBeVisible();
  });

  test("asking a question, with WhatsApp offered and naming the home", async ({ page, request }) => {
    const home = await availableHome(request);
    await page.goto(`/enquire?home=${home.id}`);
    await hydrated(page);
    const whatsapp = page.getByRole("link", { name: /Message us on WhatsApp/ });
    await expect(whatsapp).toHaveAttribute("href", new RegExp(`^https://wa\\.me/\\?text=.*${home.id}`));

    await page.getByLabel("Full name").fill(buyer.name);
    await page.getByLabel("Phone").fill(buyer.phone);
    await page.getByLabel("Email").fill("not-an-address");
    await page.getByLabel("Your question").fill("Is there parking for two cars?");
    await page.getByRole("button", { name: "Send the question" }).click();
    await expect(page.locator("#enquiry-errors")).toContainText("That email address doesn’t look right.");
    await expect(page.getByLabel("Email")).toHaveAttribute("aria-invalid", "true");

    await page.getByLabel("Email").fill(buyer.email);
    await page.getByRole("button", { name: "Send the question" }).click();
    await expect(page.getByRole("heading", { name: "Thank you, Test." })).toBeVisible();
  });

  test("the API's own refusals land beside the fields they are about", async ({ page }) => {
    // The browser's checks are the API's checks, so to see the API's answer
    // the request is sent with the browser's checks passed and the API's
    // refusal returned as it would be.
    await page.route("**/api/enquiries", (route) =>
      route.fulfill({
        status: 422,
        contentType: "application/problem+json",
        body: JSON.stringify({ type: "https://oriel-vue.jeffgicharu.com/problems/validation", title: "Some answers need another look", status: 422, errors: { phone: "That number is not in service." } }),
      }),
    );
    await page.goto("/enquire");
    await hydrated(page);
    await page.getByLabel("Full name").fill(buyer.name);
    await page.getByLabel("Phone").fill(buyer.phone);
    await page.getByLabel("Email").fill(buyer.email);
    await page.getByLabel("Your question").fill("When are the show apartments open?");
    await page.getByRole("button", { name: "Send the question" }).click();
    await expect(page.locator("#enquiry-errors")).toContainText("That number is not in service.");
    await expect(page.locator("#enquiry-phone-note")).toHaveText("That number is not in service.");
  });
});

test.describe("reserving", () => {
  test("a reservation holds the home, and its page is the buyer's", async ({ page, request }) => {
    const home = await availableHome(request);
    await page.goto(`/residences/${home.id}`);
    await hydrated(page);
    await page.locator("label.payplan", { hasText: "Cash" }).click();
    await page.getByRole("link", { name: "Request a reservation" }).click();
    await expect(page).toHaveURL(new RegExp(`/reserve/${home.id}\\?plan=cash$`));
    await expect(page.locator("label.payplan", { hasText: "Cash" }).locator("input")).toBeChecked();

    await page.getByRole("button", { name: "Request the reservation" }).click();
    await expect(page.locator("#reserve-errors")).toContainText("Please confirm you understand how a reservation works.");

    await page.getByLabel("Full name, as it will appear on the letter of offer").fill(buyer.name);
    await page.getByLabel("Phone").fill(buyer.phone);
    await page.getByLabel("Email").fill(buyer.email);
    await page.getByLabel(/I understand this is a request/).check();
    await page.getByRole("button", { name: "Request the reservation" }).click();

    await expect(page).toHaveURL(/\/reservations\/R-[A-Z0-9]+\?new=1$/);
    await expect(page.getByRole("heading", { level: 1, name: `Home ${home.id} is held for you.` })).toBeVisible();
    await expect(page.locator(".steps__item.is-current")).toContainText("Letter of offer");

    // The home is held: its own page says so, and offers no reservation.
    await page.getByRole("link", { name: "See the home" }).click();
    await expect(page.locator(".unit-head__deal .status")).toHaveText("On hold");
    await expect(page.getByRole("link", { name: "Request a reservation" })).toHaveCount(0);
  });
});

test.describe("keeping a shortlist", () => {
  test("shortlisted homes come together to compare, drawn to one scale", async ({ page, request }) => {
    const res = await request.get("/api/homes?beds=2");
    const homes = ((await res.json()) as { homes: { id: string }[] }).homes.slice(0, 2);
    for (const h of homes) {
      await page.goto(`/residences/${h.id}`);
      await hydrated(page);
      await page.getByRole("button", { name: "Shortlist" }).click();
      await expect(page.getByRole("button", { name: "Shortlisted" })).toHaveAttribute("aria-pressed", "true");
    }
    await page.goto("/shortlist");
    await hydrated(page);
    await expect(page.locator(".shortlist__item")).toHaveCount(2);
    await page.getByRole("button", { name: "Compare the chosen homes" }).click();
    await expect(page).toHaveURL(/\/compare\?homes=/);
    await expect(page.getByRole("heading", { level: 1, name: "Side by side." })).toBeVisible();
    await expect(page.locator(".compare__plan svg")).toHaveCount(2);
  });
});

test.describe("rendered on the server", () => {
  test.use({ javaScriptEnabled: false });

  test("pages arrive whole before any script runs", async ({ page, request }) => {
    const home = await availableHome(request);
    await page.goto(`/residences/${home.id}`);
    await expect(page.getByRole("heading", { level: 1 })).toContainText(home.title);
    await expect(page.locator(".unit-head__price")).toContainText("KES");
    await page.goto("/residences?beds=3");
    await expect(page.locator("tbody tr").first()).toBeVisible();
    await expect(page).toHaveTitle("Residences · Oriel, Riverside");
  });
});
