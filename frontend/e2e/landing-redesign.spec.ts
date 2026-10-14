import { expect, test } from "./fixtures";

test("redesigned WeMe marketing panels load approved art and preserve tool navigation", async ({
  page,
}) => {
  await page.goto("/");
  const hero = page.locator(".hero-landscape");
  await expect(hero).toHaveAttribute(
    "src",
    "/images/editorial/hero-sea-wide.webp",
  );
  await expect
    .poll(() =>
      hero.evaluate(
        (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
      ),
    )
    .toBe(true);

  const artwork = page.locator(".studio-pick .toolkit-art-scene img");
  await expect(artwork).toHaveCount(3);
  await expect(artwork.nth(0)).toHaveAttribute(
    "src",
    "/images/editorial/tool-email.webp",
  );
  await expect(artwork.nth(1)).toHaveAttribute(
    "src",
    "/images/editorial/tool-code.webp",
  );
  await expect(artwork.nth(2)).toHaveAttribute(
    "src",
    "/images/editorial/tool-qr.webp",
  );

  await expect(page.locator(".human-connection-panorama img")).toHaveAttribute(
    "src",
    "/images/editorial/hands-wide.webp",
  );
  await expect(page.locator(".tour-editorial-detail")).toHaveCount(0);
  await expect(page.locator(".tour-editorial-symbol")).toHaveCount(1);

  const pricing = page.locator(".studio-pass");
  await pricing.scrollIntoViewIfNeeded();
  await expect(pricing).toHaveCSS("background-image", /pricing-bg-wide\.webp/);
  await expect(pricing.getByText("₹0 · forever")).toBeVisible();
  await expect(pricing.locator(".pass-card li")).toHaveCount(4);
  await pricing
    .getByRole("button", { name: "Find your next shortcut" })
    .click();
  await expect(
    page.getByRole("searchbox", { name: "Search tools" }),
  ).toBeInViewport();
});
