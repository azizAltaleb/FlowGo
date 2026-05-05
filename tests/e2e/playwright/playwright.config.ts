import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./specs",
  timeout: 60_000,
  retries: 1,
  reporter: [["list"], ["json", { outputFile: "../../../reports/playwright-results.json" }]],
  use: {
    baseURL: process.env.FRONTEND_URL || "http://localhost:9100",
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    headless: true,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
