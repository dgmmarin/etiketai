/**
 * Auth flows: login, logout, bad credentials, register.
 * These tests run WITHOUT the saved admin session (they manage their own state).
 */
import { test, expect } from "@playwright/test";

// Override storageState for this file — start unauthenticated
test.use({ storageState: { cookies: [], origins: [] } });

test.describe("Auth / Login", () => {
  test("redirects unauthenticated users to /login", async ({ page }) => {
    await page.goto("/");
    await page.waitForURL(/\/login/);
    await expect(page.getByRole("heading", { name: "Autentificare" })).toBeVisible();
  });

  test("shows error on bad credentials", async ({ page }) => {
    await page.goto("/login");
    await page.locator("#email").fill("bad@example.com");
    await page.locator("#password").fill("wrongpassword");
    await page.getByRole("button", { name: "Intră în cont" }).click();
    await expect(page.locator(".text-destructive")).toBeVisible({ timeout: 8_000 });
  });

  test("logs in successfully and lands on dashboard", async ({ page }) => {
    await page.goto("/login");
    await page.locator("#email").fill("admin@demo.com");
    await page.locator("#password").fill("DevPass123!");
    await page.getByRole("button", { name: "Intră în cont" }).click();
    await page.waitForURL("/", { timeout: 10_000 });
    await expect(page.getByText("Bun venit")).toBeVisible();
  });

  test("logs out via topbar icon and returns to login page", async ({ page }) => {
    await page.goto("/login");
    await page.locator("#email").fill("admin@demo.com");
    await page.locator("#password").fill("DevPass123!");
    await page.getByRole("button", { name: "Intră în cont" }).click();
    await page.waitForURL("/", { timeout: 10_000 });

    await page.getByTitle("Deconectare").click();
    await page.waitForURL(/\/login/, { timeout: 5_000 });
    await expect(page.getByRole("heading", { name: "Autentificare" })).toBeVisible();
  });
});

test.describe("Auth / Register", () => {
  test("register page renders and links back to login", async ({ page }) => {
    await page.goto("/register");
    await expect(page.getByRole("heading", { name: "Creare cont" })).toBeVisible();
    await page.getByRole("link", { name: "Autentificare" }).click();
    await page.waitForURL(/\/login/);
  });

  test("shows validation error for missing fields", async ({ page }) => {
    await page.goto("/register");
    await page.getByRole("button", { name: "Creează cont" }).click();
    // Zod validation — at least one error message should appear
    await expect(page.locator(".text-destructive").first()).toBeVisible();
  });

  test("registers a new account and shows email confirmation", async ({ page }) => {
    const ts = Date.now();
    await page.goto("/register");
    await page.locator("#email").fill(`testuser-${ts}@example.com`);
    await page.locator("#password").fill("DevPass123!");
    await page.locator("#workspace_name").fill(`Test Co ${ts}`);
    await page.getByRole("button", { name: "Creează cont" }).click();
    await expect(page.getByRole("heading", { name: "Verifică emailul" })).toBeVisible({ timeout: 8_000 });
  });

  test("shows error for duplicate email", async ({ page }) => {
    await page.goto("/register");
    await page.locator("#email").fill("admin@demo.com"); // seed user
    await page.locator("#password").fill("DevPass123!");
    await page.locator("#workspace_name").fill("Duplicate Test");
    await page.getByRole("button", { name: "Creează cont" }).click();
    await expect(page.locator(".text-destructive")).toBeVisible({ timeout: 8_000 });
  });
});
