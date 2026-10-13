# WeMe

**Small tools. A little breathing room.**

WeMe turns the original email checker into a collection of **21 everyday utilities**, with a React + TypeScript interface and a Go server. The visual design pairs a generated, dithered Mediterranean coast with self-hosted Instrument Serif and DM Sans fonts, a custom SVG mark, teal controls, and consistent workspaces.

![WeMe coastal artwork](frontend/public/images/weme-coast.webp)

## The collection

| Validation | Conversion | Generators | Developer | Text |
| --- | --- | --- | --- | --- |
| Email validator | JSON ↔ YAML | QR code | JSON formatter & repair | Markdown preview |
| Phone validator | XML ↔ JSON | Secure passwords | JWT decoder | Text comparison |
| URL inspector | CSV ↔ JSON | UUID & Nano ID | Cron explainer | Case converter |
| IP inspector | Base64 | | SHA-256 / SHA-512 | |
| Work email finder | Unix timestamps | | | |
| | URL encoder/decoder | | | |

Nineteen tools run locally in the browser; email validation and work-email discovery use the same-origin server. Search, category filters, favorites, recent tool IDs, examples, copying, downloads, mobile layouts, keyboard navigation, and reduced-motion support are included. The scrolling strip credits libraries used by WeMe; it does not claim sponsorship.

See [tool research and source selection](docs/tool-research.md) and [third-party notices](docs/third-party.md).

## Develop

Use Node 24+ and Go 1.25+. The deployment build pins Node 24.19.0 and Go 1.27.2.

```bash
cd frontend
npm ci
npm run build
cd ..
go mod download
go run -mod=readonly .
```

The server listens on port 8080 by default. It embeds the frontend build, so **run `npm run build` before compiling Go**. On this cloud machine, first run `source /workspace/.cloud/weme/activate.sh` to select the installed Go compiler and caches.

For hot reload, start the Go service as above, then run `npm run dev` in `frontend/` in a second terminal. Vite proxies API requests to the Go server. Build the frontend again before testing the embedded production version.

`make build` builds `bin/weme`; `make test` builds the frontend, runs frontend unit tests, runs the Go race tests, and runs `go vet`.

## Validate

```bash
cd frontend
npm run build
npm test
npm audit
cd ..
go test -race -count=1 ./...
go vet ./...
```

Browser checks exercise every local tool example on desktop and mobile:

```bash
# Keep the Go server running first.
cd frontend
npx playwright install chromium
npm run test:e2e
```

If system Chromium exists at `/usr/bin/chromium`, the test configuration uses it. `PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH` overrides the binary; `WEME_BASE_URL` overrides the server address. Browser screenshots and traces are written only to ignored test-result directories.

## Production deployment

For Vercel, deploy the repository root using the included `vercel.json`. It builds the frontend and runs the complete Go HTTP application on Vercel's Go runtime. See [Vercel deployment instructions](docs/vercel-deployment.md); deploying only the frontend omits the email APIs.

```bash
docker build -t weme .
docker run --rm -p 8080:8080 --read-only --cap-drop=ALL \
  --security-opt=no-new-privileges --memory=512m --cpus=2 weme
```

The multi-stage Docker build installs frozen npm dependencies, builds the frontend, verifies Go dependencies, and produces a static binary. The final distroless image runs as a non-root user. `render.yaml` preserves the existing native Go service and uses `scripts/render-build.sh` to build the frontend before the server. See the [Render deployment instructions](docs/render-deployment.md) for existing service settings. TLS should terminate at the hosting platform or reverse proxy; the application serves HTTP internally. No deployment is published by local build commands.

On this cloud machine, use `python3 scripts/cloud-docker-build.py` to build through the already injected HTTPS proxy. The helper supplies its host mapping and mounts the existing public CA bundle transiently, preserving TLS verification. It never writes proxy credentials into repository files or the final image.

The existing APIs remain available. Direct frontend routes include `/tools/email`, `/tools/json`, and the other catalog IDs. `/finder` opens the new shared finder workspace. Static files use immutable caching for hashed assets; HTML revalidates and API responses are not cached.

## Privacy and boundaries

- Inputs, results, JWTs, and passwords are never saved by the interface. Optional localStorage stores only favorite/recent **tool IDs**. There are no external fonts, analytics, or tracking scripts.
- Browser utilities make no API requests. Email and finder inputs are sent to this server; avoid logging request bodies or email query strings at the reverse proxy.
- Email validity establishes syntax and mail routing, not mailbox ownership. Null MX records are treated as no mail routing. DNS transport failures produce `unknown`, not `invalid`.
- Phone validity describes numbering rules, not an active subscriber. JWT decoding does **not** authenticate a token or verify a signature.
- Markdown is sanitized with an explicit HTML allowlist. XML DTD/entities are blocked, YAML expansion is limited, CSV formula exports are escaped, and generators use cryptographic randomness.
- Direct SMTP is enabled with a 25-second total budget. Public proxy probing remains opt-in; see the [email service documentation](docs/email-service.md) for their configuration.

## Server configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | Internal HTTP port |
| `VERIFY_MAX_CONCURRENCY` | `8` | Simultaneous email checks; valid range 1–64 |
| `ENABLE_SMTP` | `true` | Direct SMTP; needs outbound port 25 |
| `SMTP_PROXY_URL` | empty | Optional private SOCKS5 route, stored as a secret; preferred over public proxies |
| `SMTP_TOTAL_TIMEOUT` | `25s` | Total mailbox verification budget |
| `ENABLE_PROXY_SMTP` | `false` | Optional SOCKS5 mailbox probing |
| `AUTO_UPDATE_DISPOSABLE` | `false` | Optional disposable-domain updates |
| `TRUST_PROXY` | `false` | Trust forwarded client IPs only behind a proxy that overwrites those headers |

Email verification allows 30 checks per IP per minute; discovery allows 10 per minute. These in-memory limits apply per instance. For multiple replicas, enforce a shared quota at the edge. Verification and SMTP concurrency, request bodies, headers, network timeouts, rate-limit state, and harvested-page cache size are bounded. The service handles SIGTERM/SIGINT with a ten-second graceful shutdown.

## Current cloud limitation

The onboarding machine's native DNS resolver refuses requests. The local frontend, server, and browser utilities work; real MX checks return an inconclusive DNS result, and public-page discovery cannot fetch company pages. The finder also uses direct HTTP/HTTPS sockets, which need a supported outbound route. Restore native DNS and that route before relying on these two server-assisted tools in this cloud environment. A passing `/healthz` reports process health, not external DNS readiness.
