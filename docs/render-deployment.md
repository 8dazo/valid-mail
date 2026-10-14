# Deploy WeMe to Render

The existing Render service uses the native Go runtime and the name `valid-mail`. Keep that service identity: the Blueprint now preserves it and builds the TypeScript frontend before compiling the Go server. The standalone Dockerfile is still available for a separately created Docker service.

## Existing service settings

For a Blueprint-managed service, sync the updated `render.yaml`. For a manually configured service, set these values in its dashboard:

| Setting | Value |
| --- | --- |
| Runtime | Go |
| Branch | `main` |
| Root directory | Repository root (leave empty) |
| Build command | `bash scripts/render-build.sh` |
| Start command | `./bin/valid-mail` |
| Health check path | `/healthz` |
| `GO_VERSION` | `1.27.2` |
| `NODE_VERSION` | `24.19.0` |
| `ENABLE_SMTP` | `false` — don't attempt blocked direct port 25 |
| `ENABLE_PROXY_SMTP` | `true` — enable the rotating free SOCKS5 pool |
| `SMTP_MAX_ATTEMPTS` | `3` — independent proxy attempts per request |
| `SMTP_NEGATIVE_CONSENSUS` | `2` — explicit missing-recipient replies before marking invalid |
| `PROXY_MAX_CANDIDATES` | `120` |
| `PROXY_MAX_HEALTHY` | `5` |
| `PROXY_PROBE_CONCURRENCY` | `30` |
| `PROXY_PROBE_TIMEOUT` | `3s` |
| `PROXY_REFRESH_INTERVAL` | `5m` |

Redeploy the latest commit after applying the settings. A Git push does not update the build/start commands of a manually configured service. Render supplies `PORT`; the application binds to all interfaces on that port.

## Why the old build fails

The earlier service ran only `go build -o bin/valid-mail .`. The current server embeds `frontend/dist`, which is intentionally excluded from Git. A clean checkout therefore fails with `frontend.go:14:12: pattern all:frontend/dist: no matching files found`. The build script installs locked frontend dependencies, compiles the local Go/WebAssembly toolkit, builds the frontend, verifies Go modules, and produces the existing binary path.

The prior Blueprint also changed the service name and runtime to `weme` / Docker. That does not provide an in-place migration for the existing native Go service. The corrected Blueprint retains `valid-mail` / Go.

## Validation

Build from a clean checkout with Node 24 and Go available:

```bash
bash scripts/render-build.sh
PORT=10000 ENABLE_SMTP=false ENABLE_PROXY_SMTP=false ./bin/valid-mail
```

Request `/healthz`, `/`, `/tools/email`, and `/api/verify` with an invalid email. Check that `/` references frontend assets that return 200. The build requires the repository root, not `frontend/` alone.

On 13 October 2026, the old build failed from a Git archive without generated files. The corrected script completed from that same clean checkout. Its binary passed startup with `PORT=10000`, health, embedded frontend/assets, tool deep links, image/logo requests, invalid-email API handling, and graceful SIGTERM shutdown. These are local build/runtime checks; Render dashboard logs and a hosted redeployment were unavailable for verification.

## Free SMTP proxy routing (restored)

The Blueprint **disables direct SMTP** and enables the earlier free ProxyScrape SOCKS5 pool. On startup (and every five minutes) the Go service fetches candidate proxies, then tests each against real MX servers on port 25, retaining at most five that deliver a valid SMTP `220` greeting. A mailbox request tries up to three independent healthy routes. A recipient-missing reply needs confirmation from two routes before the API can label the mailbox invalid. Connection timeouts, policy rejection, a lack of proxies, and inconclusive results all return **unknown** rather than falsely marking mailboxes invalid.

```bash
# Verify the running instance's effective settings and proxy health.
curl -fsS https://YOUR-RENDER-SERVICE.onrender.com/healthz
# Expected: "direct_smtp":false, "proxy_smtp_enabled":true,
# and (after warm-up) a "proxy_pool" with its "healthy" count.
```

For **manually configured** Render services, updating `render.yaml` alone does not alter Dashboard environment variables: set `ENABLE_SMTP=false` and `ENABLE_PROXY_SMTP=true` on the **existing** `valid-mail` service, then redeploy. Blueprint-managed services need a Blueprint sync. A deployment can be healthy with `proxy_pool.healthy=0` if no free proxy currently supports SMTP. Public proxies are unreliable and untrusted: never transmit passwords, session tokens, SMTP AUTH, or message bodies through them; this tool performs only unauthenticated SMTP recipient probes. The more dependable production alternative is a trusted private `SMTP_PROXY_URL` or hosting with permitted port-25 egress.
