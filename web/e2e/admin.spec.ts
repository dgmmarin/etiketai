import { test, expect, type Page } from "@playwright/test";
import { createAdminContext, loginAs, VIEWER_EMAIL, PASSWORD } from "./helpers";

let adminPage: Page;

test.describe.configure({ mode: "serial" });

test.beforeAll(async ({ browser }) => {
  ({ page: adminPage } = await createAdminContext(browser));
});

test.afterAll(async () => {
  await adminPage.context().close();
});

async function goToAdmin() {
  await adminPage.getByRole("link", { name: "Admin", exact: true }).click();
  await adminPage.waitForURL("/admin");
  await expect(adminPage.getByRole("tab", { name: /Agent AI/i })).toBeVisible({ timeout: 8_000 });
}

test.describe("Admin / Access", () => {
  test("admin link in sidebar navigates to /admin", async () => {
    await adminPage.getByRole("link", { name: "Dashboard" }).click();
    await adminPage.waitForURL("/");
    await adminPage.getByRole("link", { name: "Admin", exact: true }).click();
    await adminPage.waitForURL("/admin");
    await expect(adminPage.url()).toContain("/admin");
  });

  test("admin page renders tab navigation", async () => {
    await expect(adminPage.getByRole("tab", { name: /Agent AI/i })).toBeVisible({ timeout: 8_000 });
    await expect(adminPage.getByRole("tab", { name: /Loguri/i })).toBeVisible();
    await expect(adminPage.getByRole("tab", { name: /Rate Limits/i })).toBeVisible();
  });
});

test.describe("Admin / Agent Config", () => {
  test.beforeEach(goToAdmin);

  test("shows vision, translation and validation provider rows", async () => {
    await expect(adminPage.getByText("Vision", { exact: true })).toBeVisible({ timeout: 8_000 });
    await expect(adminPage.getByText("Translation", { exact: true })).toBeVisible();
    await expect(adminPage.getByText("Validation", { exact: true })).toBeVisible();
  });

  test("metrics card shows label statistics", async () => {
    await expect(adminPage.getByText("Etichete totale")).toBeVisible({ timeout: 8_000 });
  });

  test("shows cost estimate per 1000 labels", async () => {
    await expect(adminPage.getByText(/Cost estimat \/ 1 000 etichete/i)).toBeVisible({ timeout: 8_000 });
  });
});

test.describe("Admin / Logs", () => {
  test("switching to logs tab shows agent call history", async () => {
    await goToAdmin();
    await adminPage.getByRole("tab", { name: /Loguri/i }).click();
    await expect(
      adminPage.getByText(/Method|Metodă|log|Niciun/i)
    ).toBeVisible({ timeout: 8_000 });
  });
});

test.describe("Admin / Rate Limits", () => {
  test("switching to rate limits tab shows limit inputs", async () => {
    await goToAdmin();
    await adminPage.getByRole("tab", { name: /Rate Limits/i }).click();
    await expect(adminPage.getByText(/Încărcări/i)).toBeVisible({ timeout: 8_000 });
    await expect(adminPage.getByText(/Tipăriri/i)).toBeVisible();
  });
});

test.describe("Admin / RBAC — viewer access denied", () => {
  let viewerPage: Page;

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext();
    viewerPage = await context.newPage();
    await loginAs(viewerPage, VIEWER_EMAIL, PASSWORD);
  });

  test.afterAll(async () => {
    await viewerPage.context().close();
  });

  test("viewer sees no Admin link in sidebar", async () => {
    await expect(viewerPage.getByRole("link", { name: "Admin" })).not.toBeVisible();
  });

  test("viewer navigating to /admin via URL sees access denied message", async () => {
    // Client-side navigate to /admin
    await viewerPage.evaluate(() => window.history.pushState({}, "", "/admin"));
    await viewerPage.reload();  // force Next.js to re-render the /admin route
    // After reload, silent refresh runs → renders admin page which checks role client-side
    await viewerPage.waitForLoadState("networkidle");
    await expect(viewerPage.getByText(/restricționat|administrator/i)).toBeVisible({ timeout: 8_000 });
  });
});
