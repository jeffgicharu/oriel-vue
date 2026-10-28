import { expect, type APIRequestContext, type Page } from "@playwright/test";

/** A home still for sale, read from the API as the page would. */
export async function availableHome(request: APIRequestContext, beds?: number): Promise<{ id: string; title: string; price: number }> {
  const res = await request.get(`/api/homes${beds ? `?beds=${beds}` : ""}`);
  expect(res.ok()).toBeTruthy();
  const list = (await res.json()) as { homes: { id: string; title: string; price: number }[] };
  const home = list.homes[0];
  expect(home, "a home is available").toBeTruthy();
  return home!;
}

/** Waits until the page running in the browser has taken over from the server's HTML. */
export async function hydrated(page: Page) {
  await page.waitForFunction(() => (document.querySelector("#app") as HTMLElement & { __vue_app__?: unknown })?.__vue_app__ !== undefined);
}

export const buyer = { name: "Test Buyer", phone: "0712 345 678", email: "test-buyer@example.com" };
