# Valid Mail

A tiny self-hosted email verification service built in Go using [AfterShip/email-verifier](https://github.com/AfterShip/email-verifier).

It serves both a web UI and a JSON API from one process, so it is easy to deploy on Render.

## What it checks

- Email syntax
- DNS MX records
- Disposable email domains
- Role accounts such as `support@` or `admin@`
- Free email providers
- Common domain typos / suggestions
- Reachability / SMTP details when SMTP checking is enabled and usable

> A passing syntax + MX check does not prove that the mailbox exists. SMTP verification is optional because many cloud providers block outbound port 25 and some mail servers intentionally hide recipient status.

## Run locally

Requires Go 1.25+.

```bash
go mod download
go run .
```

Open `http://localhost:8080`.

## API

### POST `/api/verify`

```bash
curl -X POST http://localhost:8080/api/verify \
  -H 'Content-Type: application/json' \
  -d '{"email":"hello@example.com"}'
```

### GET `/api/verify?email=...`

```bash
curl 'http://localhost:8080/api/verify?email=hello@example.com'
```

### GET `/healthz`

Simple health check for Render or uptime monitoring.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port. Render injects this automatically. |
| `ENABLE_SMTP` | `false` | Enables AfterShip SMTP mailbox probing. Keep disabled unless the host/network allows outbound SMTP. |
| `AUTO_UPDATE_DISPOSABLE` | `false` | Enables the library's automatic disposable-domain metadata updater. |

## Render

This repository includes `render.yaml`, but it can also be deployed as a normal Go Web Service.

- Runtime: Go
- Build command: `go build -o bin/valid-mail .`
- Start command: `./bin/valid-mail`
- Health check path: `/healthz`

The app is intentionally a single service: no database and no frontend build step are required.

## Dependency

The service uses `github.com/AfterShip/email-verifier` and currently pins the latest tagged release used by this project in `go.mod`.
