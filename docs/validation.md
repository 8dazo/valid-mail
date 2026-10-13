# Validation evidence

Validated on 13 October 2026 using Node 24.19.0 and checksum-verified Go 1.27.2. WeMe contains 39 tools: 37 local Go/WebAssembly utilities and two Go server email tools. These checks validate the tested workflows and boundaries; they do not guarantee every possible input or external mail provider.

## Application checks

| Check | Result | What it exercises |
| --- | --- | --- |
| TypeScript and Vite production build | Passed | Strict types, Go/WASM generation, fingerprinted assets and embedded frontend output |
| Vitest | **22 passed** | Every local example through the real native Go core; conversion round trips, malformed input, known hash vectors, Unicode, passwords/IDs, sanitizer boundaries, API errors and optional browser storage |
| Playwright, local production bundle | **30 passed** | Desktop and mobile Chromium; all 37 local examples through the actual WASM worker, stable labels, search/favorites, responsive layouts and reduced motion |
| Playwright, live Vercel staging | **30 passed** | The same desktop/mobile suite through TLS-verifying egress; real deployed assets, worker startup/reuse and local tool execution |
| Go race tests | Passed | Toolkit correctness/bounds, server APIs, DNS, local SMTP protocol fixtures, private-destination guards, concurrency, deadlines/cancellation, routing and security headers |
| `go vet` and `go mod verify` | Passed | Static checks and dependency checksum verification |
| README generation check | Passed | Counts, catalog names and stack versions match application source |

The latest pastel redesign retains the original coastal hero, heading and header. Two additional desktop/mobile checks exercise its workflow shortcuts, real tool navigation from the footer, keyboard FAQ toggles and collection CTA.

The browser suite also checks fenced Markdown syntax highlighting and monospace/whitespace styling, GFM tables/task lists/details, standalone HTML downloads, image placeholders, UTF-8 file import, CSV pagination, layout switches, invalidated stale results, real barcode PNG/gzip bytes, regex groups, color swatches, conditional bcrypt controls, API cancellation and reuse of one local engine without API uploads. Email/finder rendering uses controlled responses where required, alongside the independent live API checks below.

Known-vector and boundary tests cover HMAC/SHA, gzip round trips and expansion limits, large integer preservation, CSV formula escaping/duplicate headers, YAML graph/alias limits, XML roots/DTD/nesting limits, regex time limits and Unicode groups, bcrypt byte/cost limits, cron timezones, certificate private-key rejection and barcode quiet zones. Generated Code 128, Code 39, EAN and QR images were independently decoded with ZXing-C++; the QR payload matched `https://example.com`.

## Build and deployment checks

- The native Render build script completed from a clean source snapshot without generated frontend files, installed dependencies or credentials. Its artifact started on `PORT=10000` and passed health, a tool deep link and invalid-email API checks.
- The four-stage Docker image built with verified Go modules and locked npm dependencies. The injected HTTPS proxy and transient public trust bundle retained TLS verification; proxy credentials were not copied into the image.
- The image ran as `nonroot:nonroot`, with a read-only filesystem, no capabilities and resource limits. Health, embedded HTML/deep links, invalid-email API handling and the compressed WASM asset passed. SIGTERM produced exit code zero without an OOM.
- WASM responses used `application/wasm`, gzip and immutable fingerprinted caching; decompressed bytes had the correct WASM magic. The tested first-load transfer was approximately 6.6 MiB compressed (approximately 28 MiB uncompressed). It loads on demand and is reused during SPA tool navigation.
- A fresh Vercel staging deployment built with the actual Vercel Go SDK and served the upgraded suite. All 30 browser checks passed there. Anonymous previews expire unless claimed; this is staging evidence, not a permanent release.

The currently claimed original project is https://temporary-sonic-jade-y0jexj7.vercel.app. Its old anonymous token is invalid and the observed version still has SMTP disabled. Updating that account-owned project needs owner authentication or its Git deployment integration. No Render dashboard/log credentials or hosted Render URL were available; a successful local Render build does not prove a hosted Render deployment.

## Live external API checks

Native DNS is unavailable in this cloud instance. Real public DNS and page fetching were therefore checked on the Vercel runtime, independently of browser rendering mocks.

| Hosted request | Observed result |
| --- | --- |
| Invalid email syntax | `invalid / bad_syntax`; SMTP not attempted |
| `person@example.com` with `smtp=false` | `invalid / no_mx`; SMTP not attempted |
| `postmaster@gmail.com` with `smtp=false` | Valid address/domain, MX present; mailbox `unknown / smtp_not_requested`; completed in under one second |
| `postmaster@gmail.com` with `smtp=true` | SMTP enabled and attempted; connection failed after approximately 9 seconds; valid address/domain, mailbox `unknown / smtp_connection_failed` |
| Finder for Ada Lovelace at example.com | Success; one public page fetched, zero observed emails/candidates |

SMTP protocol fixtures verify a recipient accepted while random recipients are rejected (96% verdict confidence), catch-all acceptance, nonexistent/disabled/full mailboxes, provider policy blocks, temporary replies, verified STARTTLS failure, deadlines and cancellation. Quick checks skip SMTP. Public IPv4/IPv6 destinations are vetted and pinned against DNS rebinding; alternate-IP attempts are bounded. IDN domains are normalized before mail-server lookup.

These fixtures establish protocol handling. The real hosted connection failure does **not** establish working public mailbox reachability. Reliable mailbox probing still needs supported outbound port-25 connectivity or a stable private SOCKS5 route configured securely as `SMTP_PROXY_URL`, with an appropriate sender/HELO identity. No message body is sent. An unavailable route leaves mailbox existence unknown and the UI displays **Not measured**, rather than a fabricated percentage.

## Dependency checks

The frontend audit reported zero known vulnerabilities. The official Go vulnerability scan found no vulnerable called symbols or imported packages; one advisory remains in an unused package of a required module. This is not a claim that every dependency is vulnerability-free. Runtime notices cover the actual shipped dependency graph; the GPL JSON-repair candidate was replaced with an MIT implementation before shipment.

GitHub CI runs the clean Render build, README check, formatting, frontend tests/audit, Go race/vet checks and full browser suite on pushes to `main` and pull requests. Current local/deployed evidence is recorded above; remote CI status belongs to the corresponding GitHub run. Recheck audits and upstream versions when dependencies change. Saved cloud setup instructions have been updated for the Go worker; saving them does not publish a snapshot or prove restoration in a new task.
