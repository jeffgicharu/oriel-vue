import { defineConfig, devices } from "@playwright/test";

// The journeys run against the built release: its API and its web server,
// started in CI exactly as they run on the server (see .github/workflows).
export default defineConfig({
  testDir: "e2e",
  timeout: 45_000,
  expect: { timeout: 8_000 },
  fullyParallel: true,
  workers: process.env.CI ? 3 : 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://127.0.0.1:8700",
    // In CI the tests carry the release check's token, as the gate does.
    extraHTTPHeaders: process.env.E2E_CHECK_TOKEN ? { "X-Oriel-Check": process.env.E2E_CHECK_TOKEN } : {},
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
    { name: "phone", use: { ...devices["Pixel 7"] } },
  ],
});
