/**
 * Auth setup: logs in as admin once and saves storage state.
 * All other tests reuse this state via storageState in playwright.config.ts.
 */
import { test as setup, expect } from "@playwright/test";
import path from "path";

const ADMIN_EMAIL = "admin@demo.com";
const ADMIN_PASSWORD = "DevPass123!";
const AUTH_FILE = path.join(__dirname, ".auth/admin.json");

setup("authenticate as admin", async ({ page }) => {
  await page.goto("/login");
  await expect(page.getByRole("heading", { name: "Autentificare" })).toBeVisible();

  await page.locator("#email").fill(ADMIN_EMAIL);
  await page.locator("#password").fill(ADMIN_PASSWORD);
  await page.getByRole("button", { name: "Intră în cont" }).click();

  // Wait for redirect to dashboard
  await page.waitForURL("/", { timeout: 10_000 });
  await expect(page.getByText("Bun venit")).toBeVisible();

  await page.context().storageState({ path: AUTH_FILE });
});
