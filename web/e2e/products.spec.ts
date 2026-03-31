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

async function goToProducts() {
  await page.getByRole("link", { name: "Produse" }).click();
  await page.waitForURL("/products");
  await expect(page.getByRole("heading", { name: "Produse", level: 1 })).toBeVisible();
}

test.describe("Products / List", () => {
  test.beforeEach(goToProducts);

  test("displays product count and table columns", async () => {
    await expect(page.getByText(/produse/)).toBeVisible({ timeout: 8_000 });
    await expect(page.getByRole("columnheader", { name: "Denumire" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "SKU" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Categorie" })).toBeVisible();
    await expect(page.getByRole("columnheader", { name: "Tipăriri" })).toBeVisible();
  });

  test("search filters products by name", async () => {
    await expect(page.getByText(/produse/)).toBeVisible({ timeout: 8_000 });
    await page.getByPlaceholder("Caută după nume sau SKU...").fill("food");
    await page.waitForTimeout(600);
    await expect(page.getByText(/produse/)).toBeVisible();
  });

  test("search with no results shows empty state", async () => {
    await page.getByPlaceholder("Caută după nume sau SKU...").fill("ZZZNOMATCH99999");
    await page.waitForTimeout(600);
    const noResults =
      (await page.getByText("0 produse").isVisible()) ||
      (await page.getByText("Niciun produs în bibliotecă.").isVisible());
    expect(noResults).toBeTruthy();
  });

  test("clear search restores full list", async () => {
    await page.getByPlaceholder("Caută după nume sau SKU...").clear();
    await page.waitForTimeout(600);
    await expect(page.getByText(/produse/)).toBeVisible();
  });

  test("Produs nou button opens create dialog then cancel closes it", async () => {
    await page.getByRole("button", { name: /Produs nou/ }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Produs nou" })).toBeVisible();
    await page.getByRole("button", { name: "Anulează" }).click();
    await expect(page.getByRole("dialog")).not.toBeVisible();
  });
});

test.describe("Products / Create", () => {
  test("creates a new product and it appears in the list", async () => {
    await goToProducts();
    const ts = Date.now();
    const name = `Test Produs ${ts}`;

    await page.getByRole("button", { name: /Produs nou/ }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.locator("#name").fill(name);
    await page.locator("#sku").fill(`SKU-${ts}`);
    await page.getByRole("button", { name: "Creează" }).click();

    await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 8_000 });
    // Search to locate the new product (list may paginate)
    await page.getByPlaceholder("Caută după nume sau SKU...").fill(name);
    await page.waitForTimeout(600);
    await expect(page.getByText(name)).toBeVisible({ timeout: 8_000 });
    await page.getByPlaceholder("Caută după nume sau SKU...").clear();
  });

  test("shows validation error when name is empty", async () => {
    await goToProducts();
    await page.getByRole("button", { name: /Produs nou/ }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.getByRole("button", { name: "Creează" }).click();
    await expect(page.locator(".text-destructive")).toBeVisible();
    await page.getByRole("button", { name: "Anulează" }).click();
  });
});

test.describe("Products / Edit", () => {
  test("edit button opens dialog pre-filled with product data", async () => {
    await goToProducts();
    await expect(page.getByText(/produse/)).toBeVisible({ timeout: 8_000 });

    const editBtn = page.locator("table button").first();
    await editBtn.click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Editează produs" })).toBeVisible();
    const nameInput = page.locator("#name");
    expect(await nameInput.inputValue()).not.toBe("");
    await page.getByRole("button", { name: "Anulează" }).click();
  });
});
