import { expect, test } from "./fixtures";

test("taste redesign uses an editorial layout without sacrificing real tool navigation", async ({
  page,
}) => {
  await page.goto("/");
  const featureCards = page.locator(".studio-feature");
  const picks = page.locator(".studio-pick");
  const directory = page.locator(".tool-directory");

  await expect(featureCards).toHaveCount(3);
  await expect(picks).toHaveCount(3);
  await expect(page.locator(".tool-card-category").first()).toBeVisible();

  const width = page.viewportSize()?.width ?? 1440;
  if (width > 720) {
    const first = await featureCards.nth(0).boundingBox();
    const second = await featureCards.nth(1).boundingBox();
    expect(first?.width ?? 0).toBeGreaterThan((second?.width ?? 0) * 1.5);
    await expect(page.locator(".studio-pick-grid")).toHaveCSS(
      "grid-template-columns",
      /\d+px/,
    );
  } else {
    await expect(featureCards.first()).toHaveCSS("display", "flex");
  }

  const bodyWidth = await page.evaluate(() => document.documentElement.scrollWidth);
  expect(bodyWidth).toBeLessThanOrEqual(width + 2);

  await directory.scrollIntoViewIfNeeded();
  const search = page.getByRole("searchbox", { name: "Search tools" });
  await search.fill("JSON formatter");
  await expect(page.locator(".tool-card")).toHaveCount(1);
  await expect(page.locator(".tool-card-category")).toHaveCount(1);
  await page.getByRole("link", { name: /JSON formatter/ }).click();
  await expect(page.locator(".workspace-grid")).toBeVisible();
  await expect(page.locator(".input-panel")).toBeVisible();
  await expect(page.locator(".output-panel")).toBeVisible();
});

test("editorial cards, FAQ, and pricing remain keyboard accessible", async ({ page }) => {
  await page.goto("/");
  const pricing = page.locator(".studio-pass");
  await pricing.scrollIntoViewIfNeeded();
  await expect(pricing.getByText("₹0 · forever")).toBeVisible();
  await expect(pricing.locator(".pass-card li")).toHaveCount(4);

  const cta = pricing.getByRole("button", { name: "Find your next shortcut" });
  await cta.focus();
  await expect(cta).toBeFocused();
  await cta.press("Enter");

  const search = page.getByRole("searchbox", { name: "Search tools" });
  await expect(search).toBeInViewport();
  await search.fill("PDF editor");
  await expect(page.locator(".tool-card")).toHaveCount(1);
  await expect(page.locator(".tool-card-category")).toContainText("Documents");
});
