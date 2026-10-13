import { test as base, expect } from "@playwright/test";

// Cloud Chromium cannot use the injected HTTP egress route directly. Forward
// same-origin requests through Playwright's TLS-verifying request client.
export const test = base.extend({
  page: async ({ page, playwright, baseURL }, use) => {
    const proxyURL =
      process.env.WEME_USE_EGRESS_PROXY === "1"
        ? process.env.HTTPS_PROXY
        : undefined;
    if (!proxyURL || !baseURL) {
      await use(page);
      return;
    }
    const proxy = new URL(proxyURL);
    const client = await playwright.request.newContext({
      proxy: {
        server: `${proxy.protocol}//${proxy.host}`,
        username: decodeURIComponent(proxy.username),
        password: decodeURIComponent(proxy.password),
      },
      timeout: 45_000,
      ignoreHTTPSErrors: false,
    });
    await page.context().route(`${baseURL}/**`, async (route) => {
      const request = route.request();
      const response = await client.fetch(request.url(), {
        method: request.method(),
        headers: request.headers(),
        data: request.postDataBuffer() ?? undefined,
      });
      try {
        await route.fulfill({ response });
      } catch (error) {
        // Navigating to another tool can cancel in-flight image/font requests.
        if (
          !(error instanceof Error) ||
          !/Route is already handled|Target page, context or browser has been closed/.test(
            error.message,
          )
        )
          throw error;
      }
    });
    try {
      await use(page);
    } finally {
      if (!page.isClosed())
        await page.context().unrouteAll({ behavior: "ignoreErrors" });
      await client.dispose();
    }
  },
});
export { expect };
