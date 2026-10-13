# Deploy WeMe to Vercel

Deploy the repository root, not just `frontend/`. The current Vercel Go framework supports a standalone Go HTTP server. The frontend is built first and embedded in the server binary, so the same deployment serves the UI, deep links, assets, email validation, and work-email discovery.

`vercel.json` selects the Go framework, installs frozen frontend dependencies, builds the frontend and server, uses the project’s default function duration, enables bounded direct SMTP checks, and disables public proxy/background metadata updates. The root package.json selects Node 24. The Go toolchain directive selects Go 1.27.2. No Render backend URL or cross-origin API proxy is required.

## Deploy

Use a current Vercel CLI (the configuration was prepared with v63.1.2):

```bash
vercel login
vercel deploy --prod --yes
```

If an automation token is supplied securely in the environment, use `vercel deploy --prod --yes --token "$VERCEL_TOKEN"`. Do not commit tokens, project credentials, or environment files.

The CLI supports `vercel deploy --temporary --yes` without login. After network access was applied, this succeeded for WeMe. The temporary site is https://temporary-sonic-jade-y0jexj7.vercel.app and was verified live. The user has since claimed that project. Its original anonymous deployment token can no longer update it; redeployment requires the owning account’s authentication or its Git integration. Deployment credentials are kept only in the ignored .vercel directory.

## Verify

- Confirm the URL returned by Vercel actually completes its build.
- Request `/`, `/tools/json`, `/healthz`, and the referenced asset URLs.
- Run the Playwright checks with `WEME_BASE_URL` set to the deployed URL.
- Exercise a real MX lookup and company-page discovery. Vercel's runtime connectivity must be checked independently of this cloud machine's DNS failure.
- In-memory rate limits apply per function instance. Use an edge quota for a shared limit across replicas.

## Cloud access prerequisites

The current cloud proxy initially denies `vercel.com` and `api.vercel.com`; the needed domains are saved in the environment draft, together with a personal VERCEL_TOKEN requirement for authenticated production deployment. Network access has since been verified, and anonymous deployment succeeded. A token or Vercel CLI login is still needed for authenticated deployment; claiming the temporary site can be done in the browser. Saving a draft alone does not authenticate the CLI or publish the site.

## Mailbox verification

Direct SMTP is enabled in the deployment configuration with a 25-second total budget. Live Vercel checks could not connect to Gmail’s SMTP servers, and the tested public proxy pool had no healthy routes. Configure a stable private SOCKS5 route using the `SMTP_PROXY_URL` secret, plus an `SMTP_FROM_EMAIL` / `SMTP_HELLO_NAME` identity you control when required by recipient servers. Public proxy mode remains optional; its availability cannot be guaranteed. Domain validity and SMTP confidence are shown separately. A failed connection leaves the mailbox unconfirmed.
