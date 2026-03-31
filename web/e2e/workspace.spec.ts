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

async function goToWorkspace() {
  await page.getByRole("link", { name: "Workspace" }).click();
  await page.waitForURL("/workspace");
  await expect(page.getByRole("heading", { name: "Workspace", level: 1 })).toBeVisible();
}

test.describe("Workspace / Profile", () => {
  test.beforeEach(goToWorkspace);

  test("shows company profile card with details", async () => {
    await expect(page.getByText("Profil companie")).toBeVisible();
    await expect(page.getByText("Denumire")).toBeVisible();
    await expect(page.getByText("Plan")).toBeVisible();
    await expect(page.getByText("Cotă etichete")).toBeVisible();
  });

  test("Editează button opens inline edit form", async () => {
    await expect(page.getByRole("button", { name: "Editează" })).toBeVisible({ timeout: 8_000 });
    await page.getByRole("button", { name: "Editează" }).click();
    await expect(page.getByRole("button", { name: "Salvează" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Anulează" })).toBeVisible();
    await page.getByRole("button", { name: "Anulează" }).click();
    await expect(page.getByRole("button", { name: "Editează" })).toBeVisible();
  });

  test("saves updated workspace name", async () => {
    await expect(page.getByRole("button", { name: "Editează" })).toBeVisible({ timeout: 8_000 });
    await page.getByRole("button", { name: "Editează" }).click();

    const nameInput = page.locator("input[name='name']");
    await nameInput.clear();
    await nameInput.fill("Demo Workspace SRL");
    await page.getByRole("button", { name: "Salvează" }).click();

    await expect(page.getByText("Demo Workspace SRL")).toBeVisible({ timeout: 5_000 });
  });
});

test.describe("Workspace / Members", () => {
  test.beforeEach(goToWorkspace);

  test("shows members table with correct columns", async () => {
    await expect(page.getByText("Membri")).toBeVisible();
    await expect(page.getByText("Email")).toBeVisible({ timeout: 8_000 });
    await expect(page.getByText("Rol")).toBeVisible();
    await expect(page.getByText("Alăturat")).toBeVisible();
  });

  test("seed admin appears in members list", async () => {
    await expect(page.getByRole("cell", { name: "admin@demo.com" })).toBeVisible({ timeout: 8_000 });
  });

  test("Invită button opens invite dialog and cancel closes it", async () => {
    await page.getByRole("button", { name: "Invită" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await expect(page.getByRole("heading", { name: "Invită membru" })).toBeVisible();
    await page.getByRole("button", { name: "Anulează" }).click();
    await expect(page.getByRole("dialog")).not.toBeVisible();
  });

  test("invite dialog submits with valid email and closes", async () => {
    const ts = Date.now();
    await page.getByRole("button", { name: "Invită" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.locator("#inv-email").fill(`invited-${ts}@example.com`);
    await page.getByRole("button", { name: "Trimite invitație" }).click();
    await expect(page.getByRole("dialog")).not.toBeVisible({ timeout: 8_000 });
  });

  test("invite dialog shows validation error for invalid email", async () => {
    await page.getByRole("button", { name: "Invită" }).click();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.locator("#inv-email").fill("notanemail");
    await page.getByRole("button", { name: "Trimite invitație" }).click();
    await expect(page.getByText("Email invalid")).toBeVisible();
    await page.getByRole("button", { name: "Anulează" }).click();
  });
});
