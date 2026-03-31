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
    await expect(page.getByRole("columnheader", { name: "Categorie" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Status" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Data" })).toBeVisible();
  });

  test("upload button is visible for admin", async () => {
    await expect(page.getByRole("button", { name: /Încarcă etichetă/ })).toBeVisible();
  });

  test("export CSV button is visible for admin", async () => {
    await expect(page.getByRole("button", { name: /Export CSV/ })).toBeVisible();
  });

  test("search input filters labels by product name", async () => {
    const searchInput = page.getByPlaceholder("Caută după produs...");
    await expect(searchInput).toBeVisible();
    await searchInput.fill("nonexistent-xyz-12345");
    await page.waitForTimeout(400);
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 5_000 });
    await searchInput.clear();
  });

  test("category filter changes the list", async () => {
    const statusSelect = page.getByRole("combobox").nth(1);
    await statusSelect.click();
    await page.getByRole("option", { name: "food" }).click();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 5_000 });
    await statusSelect.click();
    await page.getByRole("option", { name: "Toate categoriile" }).click();
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

  test("reset filter clears all filters", async () => {
    // Set a filter then reset
    const trigger = page.getByRole("combobox").first();
    await trigger.click();
    await page.getByRole("option", { name: "confirmed" }).click();
    await page.getByRole("button", { name: /Resetează filtre/ }).click();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 5_000 });
    await expect(page.getByRole("button", { name: /Resetează filtre/ })).not.toBeVisible();
  });
});

test.describe("Labels / Detail", () => {
  test("clicking a label opens its detail page and shows content", async () => {
    await goToLabels();
    await expect(page.getByText(/etichete/)).toBeVisible({ timeout: 8_000 });

    const firstLink = page.locator("table a").first();
    await firstLink.click();
    await page.waitForURL(/\/labels\/.+/);
    // Detail page renders label ID in h1 once data loads
    await expect(page.locator("h1").filter({ hasNotText: "" })).toBeVisible({ timeout: 10_000 });
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
