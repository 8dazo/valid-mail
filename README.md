# Valid Mail

A small self-hosted email verification service built in Go using [AfterShip/email-verifier](https://github.com/AfterShip/email-verifier).

It serves a web UI and JSON API from one process. On hosts such as Render Free that block direct SMTP, it can optionally fetch public SOCKS5 proxies from ProxyScrape, probe them for outbound SMTP access, cache the healthy ones, and rotate them for mailbox checks.

## Checks

- Email syntax
- DNS MX records
- Disposable domains
- Role accounts (`support@`, `admin@`, etc.)
- Free email providers
- Domain typo suggestions
- Optional SMTP reachability without sending an email

> SMTP is a signal, not proof. Catch-all domains and anti-abuse systems can intentionally return ambiguous results.

## Proxy SMTP mode

When `ENABLE_PROXY_SMTP=true`, the service:

1. Fetches SOCKS5 candidates from ProxyScrape's free proxy API.
2. Tests a bounded number of candidates against public MX servers on port 25.
3. Keeps only proxies that return an SMTP `220` banner.
4. Rotates healthy proxies for verification requests.
5. Falls back to DNS/domain-level results if the proxy pool is empty or SMTP is inconclusive.

Public proxies are unstable and untrusted. The service does not send authentication credentials through them. Target addresses can still be visible to a proxy operator during SMTP probing, so this mode is intended for experimentation/MVP use rather than sensitive data.

## Run locally

Requires Go 1.25+.

```bash
go mod download
go run .
```

Open `http://localhost:8080`.

## API

### `POST /api/verify`

```bash
curl -X POST http://localhost:8080/api/verify \
  -H 'Content-Type: application/json' \
  -d '{"email":"hello@example.com"}'
```

### `GET /api/verify?email=...`

```bash
curl 'http://localhost:8080/api/verify?email=hello@example.com'
```

### `GET /healthz`

Returns service state and proxy-pool health without exposing proxy IPs.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port. Render injects this. |
| `ENABLE_SMTP` | `false` | Enables direct SMTP from the host. |
| `ENABLE_PROXY_SMTP` | `true` | Enables the ProxyScrape SOCKS5 SMTP pool. |
| `PROXY_SOURCE_URL` | ProxyScrape free SOCKS5 API | Proxy list source. |
| `PROXY_MAX_CANDIDATES` | `40` | Maximum proxies tested per refresh. |
| `PROXY_MAX_HEALTHY` | `8` | Maximum healthy proxies cached. |
| `PROXY_PROBE_CONCURRENCY` | `12` | Concurrent proxy health probes. |
| `PROXY_PROBE_TIMEOUT` | `4s` | Timeout per SMTP proxy probe. |
| `PROXY_REFRESH_INTERVAL` | `5m` | Healthy-pool refresh interval. |
| `SMTP_MAX_ATTEMPTS` | `2` | Maximum proxy SMTP attempts per email check. |
| `SMTP_MAX_CONCURRENCY` | `4` | Maximum concurrent SMTP verifications. |
| `SMTP_CONNECT_TIMEOUT` | `7s` | SMTP connection timeout. |
| `SMTP_OPERATION_TIMEOUT` | `7s` | SMTP command timeout. |
| `SMTP_FROM_EMAIL` | empty | Optional real sender address for `MAIL FROM`. Recommended for higher reliability. |
| `SMTP_HELLO_NAME` | empty | Optional real hostname/domain for `EHLO`. Recommended for higher reliability. |
| `AUTO_UPDATE_DISPOSABLE` | `false` | Enable AfterShip disposable-domain auto updates. |

## Render

`render.yaml` configures a free Go service in Singapore. Direct SMTP stays disabled; proxy SMTP is enabled.

- Build: `go build -o bin/valid-mail .`
- Start: `./bin/valid-mail`
- Health: `/healthz`

The app has no database and no frontend build step.
