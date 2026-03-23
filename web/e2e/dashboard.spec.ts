import { test, expect, type Page } from "@playwright/test";
import { createAdminContext } from "./helpers";

let page: Page;

test.describe.configure({ mode: "serial" });

test.beforeAll(async ({ browser }) => {
  ({ page } = await createAdminContext(browser));
});

test.afterAll(async () => {
  await page.context().close();
});

test.describe("Dashboard", () => {
  test.beforeEach(async () => {
    // Client-side navigation to dashboard — preserves Zustand token
    await page.getByRole("link", { name: "Dashboard" }).click();
    await page.waitForURL("/");
  });

  test("shows welcome message with username", async () => {
    await expect(page.getByText(/Bun venit/)).toBeVisible();
  });

  test("displays four summary cards", async () => {
    await expect(page.getByText("Etichete totale")).toBeVisible();
    await expect(page.getByText("Utilizate / Limită")).toBeVisible();
    await expect(page.getByText("Plan")).toBeVisible();
    await expect(page.getByText("Rol")).toBeVisible();
  });

  test("sidebar navigation is present", async () => {
    await expect(page.getByRole("link", { name: "Dashboard" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Etichete" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Produse" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Workspace" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Abonament" })).toBeVisible();
  });

  test("admin link is visible for admin user", async () => {
    await expect(page.getByRole("link", { name: "Admin" })).toBeVisible();
  });

  test("topbar shows user email", async () => {
    await expect(page.getByText("admin@demo.com")).toBeVisible();
  });

  test("clicking Etichete navigates to labels page", async () => {
    await page.getByRole("link", { name: "Etichete" }).click();
    await page.waitForURL("/labels");
    await expect(page.getByRole("heading", { name: "Etichete", level: 1 })).toBeVisible();
  });
});
