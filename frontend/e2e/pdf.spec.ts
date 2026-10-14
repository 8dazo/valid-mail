import { expect, test } from "./fixtures";

test("PDF Studio uploads, previews page metadata, edits text, and offers a download", async ({ page }) => {
  const inspected = {
    success: true,
    page_count: 2,
    pages: [
      { number: 1, width: 612, height: 792 },
      { number: 2, width: 612, height: 792 },
    ],
    runs: [{ text: "DRAFT agreement", x: 72, y: 110, size: 16, font: "Helvetica" }],
    fields: [],
    encrypted: false,
    has_signatures: false,
  };
  await page.route("**/api/pdf/inspect", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(inspected) });
  });
  await page.route("**/api/pdf/preview", async (route) => {
    // The editor should gracefully use the browser's native PDF viewer when
    // a server-side renderer cannot handle a particular document.
    await route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ error: "Renderer unavailable" }) });
  });
  await page.route("**/api/pdf/edit", async (route) => {
    const raw = route.request().postDataBuffer()?.toString("utf8") ?? "";
    expect(raw).toContain('name="operation"');
    expect(raw).toContain("replace");
    expect(raw).toContain("DRAFT");
    expect(raw).toContain("FINAL");
    await route.fulfill({ status: 200, contentType: "application/pdf", body: Buffer.from("%PDF-1.7\n% TEST OUTPUT\n") });
  });
  await page.goto("/tools/pdf");
  await expect(page.getByRole("heading", { name: /Make your PDF/ })).toBeVisible();
  await expect(page.getByText("Drop a PDF to start creating")).toBeVisible();
  await page.locator('input[type="file"][accept=".pdf,application/pdf"]').first().setInputFiles({
    name: "agreement.pdf",
    mimeType: "application/pdf",
    buffer: Buffer.from("%PDF-1.7\n% TEST INPUT\n"),
  });
  await expect(page.getByText("2 pages")).toBeVisible();
  await expect(page.getByText("DRAFT agreement")).not.toBeVisible();
  await page.getByLabel("Find existing text").fill("DRAFT");
  await expect(page.getByText("DRAFT agreement")).toBeVisible();
  await page.getByLabel("Replace with").fill("FINAL");
  await page.getByRole("button", { name: /Apply & create PDF/ }).click();
  await expect(page.getByRole("link", { name: /Download/ })).toHaveAttribute("download", "weme-replace.pdf");
  await expect(page.getByRole("heading", { name: /Make your PDF/ })).toBeVisible();
});
