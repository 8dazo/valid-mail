import { expect, test } from "./fixtures";
import { tools } from "../src/catalog";

test("pastel landing shortcuts, FAQs and footer open the real workspaces", async ({
  page,
}) => {
  await page.goto("/");
  await expect(page.locator(".studio-stats")).toContainText(
    String(tools.length),
  );
  await expect(page.locator(".studio-feature")).toHaveCount(3);
  await page
    .getByRole("button", { name: "Make a QR code", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "QR code generator." }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "README preview", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Markdown & README preview." }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Explore all tools", exact: true })
    .click();
  await expect(
    page.getByRole("searchbox", { name: "Search tools" }),
  ).toBeInViewport();
  await page
    .getByRole("button", { name: "README preview", exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Markdown & README preview." }),
  ).toBeVisible();
  await page.keyboard.press("Control+k");
  await expect(
    page.getByRole("searchbox", { name: "Search tools" }),
  ).toBeFocused();
  const question = page
    .locator(".faq-list summary")
    .filter({ hasText: "Does my data leave my browser?" });
  await question.focus();
  await question.press("Enter");
  await expect(question.locator("..")).toHaveAttribute("open", "");
  await expect(question.locator("..").locator("p")).toContainText(
    "Email verification",
  );
  await question.press("Enter");
  await expect(question.locator("..")).not.toHaveAttribute("open", "");
  await page
    .getByRole("button", { name: "Find your next shortcut", exact: true })
    .click();
  await expect(
    page.getByRole("searchbox", { name: "Search tools" }),
  ).toBeInViewport();
});

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
  test.setTimeout(180_000);
  await page.goto("/");
  for (const { id } of tools.filter((tool) => !tool.server)) {
    await page.locator(`.card-link[href="/tools/${id}"]`).click();
    await page.getByRole("button", { name: "Try example" }).click();
    await page.locator('.input-actions button[type="submit"]').click();
    await expect(page.locator(".result-body"), id).toBeVisible();
    await expect(page.getByRole("alert"), id).toHaveCount(0);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
      id,
    ).toBe(true);
    await page.getByRole("button", { name: "All tools", exact: true }).click();
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
  await expect(page.locator(".data-view")).toContainText("jane@fixture.test");
  await page.getByRole("button", { name: "Raw output", exact: true }).click();
  await expect(page.locator(".output-code")).toContainText("jane@fixture.test");
  await expect(page.locator(".facts")).toContainText("fixture.test");
  readable = false;
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.getByRole("alert")).toContainText(
    "Public pages could not be read",
  );
  await expect(page.locator(".result-body")).toHaveCount(0);
});

test("README preview preserves code, tables, details and a standalone HTML download", async ({
  page,
}) => {
  await page.goto("/tools/markdown");
  await page
    .getByLabel("Markdown", { exact: true })
    .fill(
      '# README\n\n```json\n{\n  "name": "WeMe",\n  "tools": 39\n}\n```\n\n| Name | Mode |\n| --- | --- |\n| WeMe | Local |\n\n- [x] Ready\n\n<details><summary>More information</summary><p>Details work.</p></details>\n\n![Logo](images/logo.png)',
    );
  await page
    .getByRole("button", { name: "Preview Markdown", exact: true })
    .click();
  await expect(page.locator(".markdown-preview pre code")).toContainText(
    '  "name": "WeMe"',
  );
  expect(
    await page
      .locator(".markdown-preview pre code")
      .evaluate((element) => getComputedStyle(element).fontFamily),
  ).toContain("monospace");
  expect(
    await page
      .locator(".markdown-preview pre")
      .evaluate((element) => getComputedStyle(element).whiteSpace),
  ).toBe("pre");
  await expect(page.locator(".markdown-preview table")).toContainText("WeMe");
  await expect(
    page.locator(".markdown-preview input[type=checkbox]"),
  ).toBeChecked();
  await page.locator(".markdown-preview summary").click();
  await expect(page.locator(".markdown-preview details")).toHaveAttribute(
    "open",
    "",
  );
  await expect(page.locator(".image-placeholder")).toContainText("Logo");
  await expect(page.locator(".markdown-preview img")).toHaveCount(0);
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download result" }).click();
  const file = await download;
  expect(file.suggestedFilename()).toBe("weme-preview.html");
  const { readFile } = await import("node:fs/promises");
  const content = await readFile((await file.path())!, "utf8");
  expect(content).toContain("<!doctype html>");
  expect(content).toContain("white-space:pre");
  await page.getByRole("button", { name: "HTML source", exact: true }).click();
  await expect(page.locator(".output-code")).toContainText("<!doctype html>");
});

test("imports text files, paginates CSV, switches layouts and invalidates stale output", async ({
  page,
}) => {
  await page.goto("/tools/csv");
  await page.locator('input[type="file"]').setInputFiles({
    name: "records.csv",
    mimeType: "text/csv",
    buffer: Buffer.from(
      "name,value\n" +
        Array.from({ length: 60 }, (_, index) => `row-${index},${index}`).join(
          "\n",
        ),
    ),
  });
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".data-view")).toContainText("60 rows");
  await expect(page.locator(".data-view tbody tr")).toHaveCount(25);
  await page.getByRole("button", { name: "Next rows" }).click();
  await expect(page.locator(".data-view")).toContainText("Page 2 of 3");
  await page.getByRole("button", { name: "Stack panels", exact: true }).click();
  await expect(page.locator(".workspace-grid")).toHaveClass(/stacked-panels/);
  await page.getByRole("button", { name: "More room", exact: true }).click();
  await expect(page.locator(".workspace")).toHaveClass(/workspace-expanded/);
  await page.getByLabel("Data", { exact: true }).fill("name,value\nnew,1");
  await expect(page.locator(".result-body")).toHaveCount(0);
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".data-view")).toContainText("new");
});

test("Go-backed barcode PNG and gzip downloads contain real binary data", async ({
  page,
}) => {
  const { readFile } = await import("node:fs/promises");
  const { gunzipSync } = await import("node:zlib");
  await page.goto("/tools/barcode");
  await page.getByRole("button", { name: "Try example" }).click();
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(
    page.getByRole("img", { name: "Generated barcode" }),
  ).toBeVisible();
  let waiting = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download result" }).click();
  let file = await waiting;
  expect(file.suggestedFilename()).toBe("weme-barcode.png");
  const png = await readFile((await file.path())!);
  expect(png.subarray(0, 8).toString("hex")).toBe("89504e470d0a1a0a");
  await page.goto("/tools/gzip");
  await page.getByLabel("Text or Base64 gzip").fill("Hello, 世界 👋");
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".result-body")).toBeVisible();
  waiting = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download result" }).click();
  file = await waiting;
  expect(file.suggestedFilename()).toBe("weme.txt.gz");
  expect(
    gunzipSync(await readFile((await file.path())!)).toString("utf8"),
  ).toBe("Hello, 世界 👋");
});

test("regex groups, color swatches and bcrypt mode controls fit their tool", async ({
  page,
}) => {
  await page.goto("/tools/regex");
  await page.getByRole("button", { name: "Try example" }).click();
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".regex-preview mark")).toHaveCount(2);
  await expect(page.locator(".data-view")).toContainText("order");
  await page.goto("/tools/color");
  await page.getByRole("button", { name: "Try example" }).click();
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".color-chip")).toHaveCount(3);
  await expect(page.locator(".data-view")).toContainText("rgb(7, 91, 83)");
  await page.goto("/tools/bcrypt");
  await expect(page.getByLabel("Bcrypt hash", { exact: true })).toHaveCount(0);
  await page.getByLabel("Operation").selectOption("verify");
  await expect(page.getByLabel("Bcrypt hash", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Cost (8–12)")).toHaveCount(0);
});

test("editing or cancelling an in-flight check keeps new input and clears old results", async ({
  page,
}) => {
  await page.route("**/api/verify", async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 1500));
    await route.fulfill({
      json: {
        success: true,
        verification: { syntax: { valid: true }, has_mx_records: true },
        validity: { status: "valid" },
        reachability: { status: "unknown" },
      },
    });
  });
  await page.goto("/tools/email");
  await page
    .getByLabel("Email address", { exact: true })
    .fill("first@example.com");
  await page.getByRole("button", { name: "Verify email", exact: true }).click();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(page.getByLabel("Email address", { exact: true })).toHaveValue(
    "first@example.com",
  );
  await expect(page.locator(".result-body")).toHaveCount(0);
  await page.getByRole("button", { name: "Verify email", exact: true }).click();
  await page
    .getByLabel("Email address", { exact: true })
    .fill("second@example.com");
  await expect(page.locator(".result-body")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Verify email", exact: true }),
  ).toBeEnabled();
});

test("switching tools reuses the local engine without uploading inputs", async ({
  page,
  context,
}) => {
  const wasm: string[] = [];
  const api: string[] = [];
  context.on("request", (request) => {
    if (request.url().endsWith(".wasm")) wasm.push(request.url());
    if (request.url().includes("/api/")) api.push(request.url());
  });
  await page.goto("/tools/json");
  await page.getByRole("button", { name: "Try example" }).click();
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".result-body")).toBeVisible();
  await page.getByRole("button", { name: "All tools", exact: true }).click();
  await page.locator('.card-link[href="/tools/color"]').click();
  await page.getByRole("button", { name: "Try example" }).click();
  await page.locator('.input-actions button[type="submit"]').click();
  await expect(page.locator(".color-swatches")).toBeVisible();
  expect(wasm).toHaveLength(1);
  expect(api).toEqual([]);
});
