# Validation evidence

The current implementation was checked in the cloud machine using Node 24 and the checksum-verified Go 1.27.2 toolchain.

- Strict TypeScript check and Vite production build passed.
- Twenty-two Vitest tests passed, covering every local tool example, conversion round trips, malformed inputs, standard SHA vectors, password/ID generation, XML/YAML boundaries, Markdown sanitization, CSV formula escaping, API errors, and optional storage.
- Sixteen Playwright checks passed across desktop Chromium and a mobile Chromium viewport. They exercised the full local collection, search/favorites, direct tool routes, examples, resets, QR downloads, sanitized Markdown, DNS-outage display, SMTP confidence/evidence, finder success/error rendering, reduced motion, and overflow.
- Go tests passed with the race detector; `go vet` and module checksum verification passed. Checks cover existing library logic, email extraction, bounded/null-MX DNS behavior, live local SMTP protocol fixtures, cancellation/deadlines, proxy consensus/cold starts, private-destination blocking, API parsing, concurrency/rate limits, forwarded-header trust, embedded routing, and browser security headers.
- `npm audit` reported zero known vulnerabilities across installed runtime and development dependencies at validation time.
- The Docker production image built through the existing HTTPS egress proxy with TLS verification and the injected public trust bundle. It ran as `nonroot:nonroot` with a read-only filesystem, no capabilities, and resource limits. Embedded UI, health, an inconclusive DNS-error response, and a graceful zero-exit shutdown passed.
- The Vite development server and same-origin API proxy were checked with live requests.

## External runtime limits

Native DNS in this cloud instance is unavailable. Real MX and public-page requests were therefore verified independently on Vercel, with TLS verification retained. Finder rendering tests use controlled success and failure responses; a separate real hosted finder request fetched example.com successfully and returned zero candidates.

The official Go vulnerability scan completed: no vulnerabilities were found in called symbols or imported packages. Five advisories exist in required modules that are not reached by this application. The frontend dependency audit reported zero known vulnerabilities.

## Live Vercel validation

The user claimed the original deployment at https://temporary-sonic-jade-y0jexj7.vercel.app. Its previous anonymous token is invalid, so updating that owned project requires account authentication or a configured Git integration.

A new temporary staging deployment at https://temporary-express-silver-e2mqji3.vercel.app was built with the actual Vercel Go SDK. Real syntax and null-MX checks passed. The hosted finder fetched one example.com page and returned success with zero candidates. SMTP was enabled, and a real Gmail-domain check attempted the connection but could not reach the mail servers (approximately 21 seconds). Public proxy mode was also exercised: no healthy SMTP-capable route was available, and direct fallback was inconclusive. These checks demonstrate correct failure handling; they do not establish working public mailbox reachability.

All sixteen browser checks also passed against the staging deployment through TLS-verifying cloud egress, across desktop and mobile. Every local utility example completed without runtime errors. Email/finder rendering used controlled responses where necessary, alongside the real hosted API checks above.

The SMTP percentage and evidence UI is restored. Local SMTP protocol fixtures verify recipient acceptance with random-recipient rejection (96% verdict confidence), catch-all acceptance, missing recipients, policy blocks, temporary responses and TLS failure. Request cancellation and total deadlines are enforced. A private SOCKS5 route can be supplied securely using SMTP_PROXY_URL for reliable outbound SMTP. No message body is sent. Mailbox existence stays unknown when no route completes; the UI shows “Not measured” rather than an invented percentage.

GitHub's published run summaries confirm that both commits passed remote CI: [af5b9b5](https://github.com/8dazo/valid-mail/actions/runs/38362572569) and [3a06b9d](https://github.com/8dazo/valid-mail/actions/runs/38383333666). The latter exercises the Render build script from a clean runner. These results do not establish a successful Render-hosted deployment or fresh cloud-task restoration.
