import { expect, test } from "./fixtures";

test("landing, local search, favorites, and keyboard navigation work", async ({
  page,
}) => {
  const failures: string[] = [];
  page.on("pageerror", (error) => failures.push(error.message));
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Less friction. More creating." }),
  ).toBeVisible();
  const hero = page.locator(".hero-landscape");
  await expect(hero).toBeVisible();
  await expect
    .poll(
      () =>
        hero.evaluate(
          (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
        ),
      { timeout: 15_000 },
    )
    .toBe(true);
  await page.getByRole("button", { name: "Meet your toolkit" }).click();
  await page
    .getByRole("searchbox", { name: "Search tools" })
    .fill("JSON formatter");
  await expect(page.locator(".tool-card")).toHaveCount(1);
  await page
    .getByRole("button", { name: "Add JSON formatter to favorites" })
    .click();
  await page.getByRole("searchbox").fill("");
  await page.getByRole("button", { name: /Favorites/ }).click();
  await expect(page.locator(".tool-card")).toHaveCount(1);
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Remove JSON formatter from favorites" }),
  ).toBeVisible();
  expect(failures).toEqual([]);
});

test("JSON workspace repairs input, reports errors, and resets without stale results", async ({
  page,
  context,
}) => {
  await page.goto("/tools/json");
  await expect(
    page.getByRole("heading", { name: "JSON formatter." }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Try example" }).click();
  await page.getByRole("button", { name: "Format JSON", exact: true }).click();
  await expect(page.locator(".output-code")).toContainText('"name": "WeMe"');
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.getByRole("button", { name: "Copy result" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toContain(
    '"name": "WeMe"',
  );
  await page.evaluate(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: async () => {
          throw new Error("Permission denied");
        },
      },
    });
  });
  await page.getByRole("button", { name: "Copied result" }).click();
  await expect(page.getByRole("status")).toContainText("Select the output");
  await expect(page.locator(".output-code")).toBeVisible();
  await page.getByLabel("Operation").selectOption("format");
  await page.getByLabel("JSON input").fill("{broken");
  await page.getByRole("button", { name: "Format JSON", exact: true }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await page.getByRole("button", { name: "Reset input and output" }).click();
  await expect(
    page.getByRole("heading", { name: "Good things start here." }),
  ).toBeVisible();
  await expect(page.getByLabel("JSON input")).toHaveValue("");
});

test("QR downloads, sanitized Markdown, and browser-local execution work", async ({
  page,
}) => {
  const requests: string[] = [];
  page.on("request", (request) => {
    if (request.url().includes("/api/")) requests.push(request.url());
  });
  await page.goto("/tools/qr");
  await page.getByRole("button", { name: "Try example" }).click();
  await page
    .getByRole("button", { name: "Generate QR code", exact: true })
    .click();
  await expect(
    page.getByRole("img", { name: "Generated QR code" }),
  ).toBeVisible();
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download result" }).click();
  expect((await download).suggestedFilename()).toBe("weme-qr.svg");
  await page.goto("/tools/markdown");
  await page
    .getByLabel("Markdown", { exact: true })
    .fill(
      '# Safe\n<img src=x onerror="window.__attacked=true"><script>window.__attacked=true</script>',
    );
  await page
    .getByRole("button", { name: "Preview Markdown", exact: true })
    .click();
  await expect(page.locator(".markdown-preview h1")).toHaveText("Safe");
  await expect(
    page.locator(".markdown-preview img,.markdown-preview script"),
  ).toHaveCount(0);
  expect(
    await page.evaluate(
      () => (window as unknown as { __attacked?: boolean }).__attacked,
    ),
  ).toBeUndefined();
  expect(requests).toEqual([]);
});

test("email results distinguish syntax errors and inconclusive DNS failures", async ({
  page,
}) => {
  await page.goto("/tools/email");
  await page.getByLabel("Email address").fill("not-an-email");
  await page.getByRole("button", { name: "Verify email", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Email invalid" }),
  ).toBeVisible();
  await page.route("**/api/verify", (route) =>
    route.fulfill({
      json: {
        success: true,
        verification: { syntax: { valid: true }, has_mx_records: false },
        validity: { status: "unknown" },
        warning: "DNS unavailable",
        reachability: { status: "unknown" },
      },
    }),
  );
  await page.getByLabel("Email address").fill("hello@gmail.com");
  await page.getByRole("button", { name: "Verify email", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Verification inconclusive" }),
  ).toBeVisible();
  await expect(page.locator(".result-notice")).toContainText("DNS unavailable");
});

test("every local tool example produces a result without a runtime error", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  for (const id of [
    "phone",
    "url",
    "ip",
    "json",
    "yaml",
    "xml",
    "csv",
    "jwt",
    "cron",
    "qr",
    "markdown",
    "diff",
    "id",
    "password",
    "hash",
    "base64",
    "timestamp",
    "case",
    "url-codec",
  ]) {
    await page.goto(`/tools/${id}`);
    await page.getByRole("button", { name: "Try example" }).click();
    await page.locator('.input-actions button[type="submit"]').click();
    await expect(page.locator(".result-body"), id).toBeVisible();
    await expect(page.getByRole("alert"), id).toHaveCount(0);
  }
  expect(errors).toEqual([]);
});

test("layout fits the viewport and reduced motion stops the marquee", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  expect(
    await page
      .locator(".marquee-track")
      .evaluate((element) => getComputedStyle(element).animationName),
  ).toBe("none");
  await page.goto("/tools/diff");
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});

test("SMTP evidence displays confidence and leaves blocked mailboxes unconfirmed", async ({
  page,
}) => {
  let accepted = true;
  await page.route("**/api/verify", (route) =>
    route.fulfill({
      json: {
        success: true,
        smtp_enabled: true,
        smtp_attempted: true,
        smtp_via_proxy: false,
        verification: { syntax: { valid: true }, has_mx_records: true },
        validity: { status: "valid" },
        reachability: accepted
          ? {
              status: "valid",
              confidence: 96,
              evidence: "Target accepted; random recipients rejected.",
              provider: "fixture",
              mx_host: "mx.fixture.test",
              catch_all: false,
              attempts: 1,
              target: {
                code: 250,
                enhanced_code: "2.1.5",
                message: "Recipient accepted",
              },
            }
          : {
              status: "unknown",
              confidence: 0,
              evidence: "Route blocked by policy.",
              attempts: 1,
            },
      },
    }),
  );
  await page.goto("/tools/email");
  await page.getByLabel("Email address").fill("target@fixture.test");
  await page.getByRole("button", { name: "Verify email", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Mailbox reachable", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("progressbar", { name: "Verdict confidence" }),
  ).toHaveAttribute("value", "96");
  await expect(
    page.getByRole("region", { name: "SMTP verification evidence" }),
  ).toContainText("mx.fixture.test");
  accepted = false;
  await page.getByRole("button", { name: "Verify email", exact: true }).click();
  await expect(
    page.getByRole("heading", {
      name: "Address valid; mailbox unconfirmed",
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.locator(".smtp-confidence")).toContainText("Not measured");
  await expect(page.getByRole("progressbar")).toHaveCount(0);
});

test("work email finder renders candidate data and public-page failures", async ({
  page,
}) => {
  let readable = true;
  await page.route("**/api/find-work-email", (route) =>
    route.fulfill({
      status: readable ? 200 : 502,
      json: readable
        ? {
            success: true,
            domain: "fixture.test",
            pages_fetched: 2,
            observed_email_count: 1,
            candidates: [{ email: "jane@fixture.test" }],
          }
        : { success: false, error: "Public pages could not be read" },
    }),
  );
  await page.goto("/tools/finder");
  await page.getByRole("button", { name: "Try example" }).click();
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".output-code")).toContainText("jane@fixture.test");
  await expect(page.locator(".facts")).toContainText("fixture.test");
  readable = false;
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.getByRole("alert")).toContainText(
    "Public pages could not be read",
  );
  await expect(page.locator(".result-body")).toHaveCount(0);
});
