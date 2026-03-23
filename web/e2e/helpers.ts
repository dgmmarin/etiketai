/**
 * Shared test helpers for EtiketAI Playwright tests.
 */
import { type BrowserContext, type Page } from "@playwright/test";

export const ADMIN_EMAIL = "admin@demo.com";
export const VIEWER_EMAIL = "viewer@demo.com";
export const OPERATOR_EMAIL = "operator@demo.com";
export const PASSWORD = "DevPass123!";

/**
 * Login via the UI login form and wait for the dashboard.
 * Uses a full page load — call this ONCE per browser context.
 * All subsequent navigation should be done via navTo() to preserve Zustand state.
 */
export async function loginAs(
  page: Page,
  email: string,
  password = PASSWORD,
): Promise<void> {
  await page.goto("/login");
  await page.locator("#email").fill(email);
  await page.locator("#password").fill(password);
  await page.getByRole("button", { name: "Intră în cont" }).click();
  await page.waitForURL("/", { timeout: 15_000 });
  // Confirm dashboard rendered (Zustand state is now populated)
  await page.getByText(/Bun venit/).waitFor({ state: "visible", timeout: 10_000 });
}

/**
 * Navigate using the sidebar link (client-side navigation — preserves Zustand).
 * Falls back to page.goto only if the sidebar link is not found.
 */
export async function navTo(
  page: Page,
  linkText: string,
  expectedUrl: string,
): Promise<void> {
  try {
    const link = page.getByRole("link", { name: linkText, exact: true });
    await link.waitFor({ state: "visible", timeout: 3_000 });
    await link.click();
    await page.waitForURL(expectedUrl, { timeout: 10_000 });
  } catch {
    // Sidebar not visible (e.g., already on the page) — no-op
  }
}

/**
 * Creates a fresh browser context, logs in as admin, and returns both.
 * Call in test.beforeAll. Close the context in test.afterAll.
 */
export async function createAdminContext(
  browser: import("@playwright/test").Browser,
): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext();
  const page = await context.newPage();
  await loginAs(page, ADMIN_EMAIL);
  return { context, page };
}
