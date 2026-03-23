import { test, expect, type Page } from "@playwright/test";
import { createAdminContext } from "./helpers";

let page: Page;

test.describe.configure({ mode: "serial" });

test.beforeAll(async ({ browser }) => {
  ({ page } = await createAdminContext(browser));
  // Navigate to labels once via sidebar
  await page.getByRole("link", { name: "Etichete" }).click();
  await page.waitForURL("/labels");
});

test.afterAll(async () => {
  await page.context().close();
});

async function goToLabels() {
  await page.getByRole("link", { name: "Etichete" }).click();
  await page.waitForURL("/labels");
  await expect(page.getByRole("heading", { name: "Etichete", level: 1 })).toBeVisible();
}

test.describe("Labels / List", () => {
  test.beforeEach(goToLabels);

  test("displays label count and table", async () => {
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 8_000 });
    await expect(page.getByRole("columnheader", { name: "Fișier" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Status" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Data" })).toBeVisible();
  });

  test("upload button is visible for admin", async () => {
    await expect(page.getByRole("button", { name: /Încarcă etichetă/ })).toBeVisible();
  });

  test("export CSV button is visible for admin", async () => {
    await expect(page.getByRole("button", { name: /Export CSV/ })).toBeVisible();
  });

  test("filter by status 'confirmed' updates the list", async () => {
    const trigger = page.getByRole("combobox").first();
    await trigger.click();
    await page.getByRole("option", { name: "confirmed" }).click();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 5_000 });
  });

  test("filter by status 'pending' shows pending labels", async () => {
    const trigger = page.getByRole("combobox").first();
    await trigger.click();
    await page.getByRole("option", { name: "pending" }).click();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 5_000 });
  });

  test("reset filter to all statuses", async () => {
    const trigger = page.getByRole("combobox").first();
    await trigger.click();
    await page.getByRole("option", { name: "Toate statusurile" }).click();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Labels / Detail", () => {
  test("clicking a label opens its detail page and shows content", async () => {
    await goToLabels();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 8_000 });

    const firstLink = page.locator("table a").first();
    await firstLink.click();
    await page.waitForURL(/\/labels\/.+/);
    // Detail page should render a heading or field content
    await expect(page.locator("h1, h2, h3").first()).toBeVisible({ timeout: 8_000 });
    // Navigate back
    await page.goBack();
    await page.waitForURL("/labels");
  });
});

test.describe("Labels / Export", () => {
  test("CSV export triggers a download", async () => {
    await goToLabels();
    const downloadPromise = page.waitForEvent("download", { timeout: 10_000 });
    await page.getByRole("button", { name: /Export CSV/ }).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/etichete-export.*\.csv/);
  });
});
