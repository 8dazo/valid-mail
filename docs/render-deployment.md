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

Redeploy the latest commit after applying the settings. A Git push does not update the build/start commands of a manually configured service. Render supplies `PORT`; the application binds to all interfaces on that port.

## Why the old build fails

The earlier service ran only `go build -o bin/valid-mail .`. The current server embeds `frontend/dist`, which is intentionally excluded from Git. A clean checkout therefore fails with `frontend.go:14:12: pattern all:frontend/dist: no matching files found`. The new build script installs locked frontend dependencies, builds the frontend, verifies Go modules, and produces the existing binary path.

The prior Blueprint also changed the service name and runtime to `weme` / Docker. That does not provide an in-place migration for the existing native Go service. The corrected Blueprint retains `valid-mail` / Go.

## Validation

Build from a clean checkout with Node 24 and Go available:

```bash
bash scripts/render-build.sh
PORT=10000 ENABLE_SMTP=false ENABLE_PROXY_SMTP=false ./bin/valid-mail
```

Request `/healthz`, `/`, `/tools/email`, and `/api/verify` with an invalid email. Check that `/` references frontend assets that return 200. The build requires the repository root, not `frontend/` alone.

On 13 October 2026, the old build failed from a Git archive without generated files. The corrected script completed from that same clean checkout. Its binary passed startup with `PORT=10000`, health, embedded frontend/assets, tool deep links, image/logo requests, invalid-email API handling, and graceful SIGTERM shutdown. These are local build/runtime checks; Render dashboard logs and a hosted redeployment were unavailable for verification.

Render Free does not provide direct outbound SMTP. The Blueprint disables direct and public-proxy SMTP; a stable private `SMTP_PROXY_URL` can be configured securely for mailbox probing. An unavailable SMTP route should leave mailbox existence unconfirmed, independently of service deployment health.
