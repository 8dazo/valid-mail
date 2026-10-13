# Valid Mail

A small self-hosted email validity, work-email discovery, and reachability service written in Go.

The primary validity check uses [AfterShip/email-verifier](https://github.com/AfterShip/email-verifier) for syntax and domain metadata. The existing confidence-aware SMTP probe layer remains separate and can run through rotating SOCKS5 proxies when mailbox-level evidence is available.

The web UI and JSON API are served by the same process, so the project can run as one Render service.

## Product surfaces

- `/` — verify a known email address
- `/finder` — find likely work-email candidates from a person's name and company

## What it checks

### Validity — no SMTP required

- Email syntax
- DNS MX records
- Disposable email domains
- Role accounts such as `support@` and `admin@`
- Free email providers
- Common domain typo suggestions

### Work-email discovery — no LinkedIn scraping

The finder accepts a person name plus either a company name or a known company domain.

When only a company name is supplied, Valid Mail:

1. Generates likely official-domain candidates from the company name.
2. Keeps only public DNS destinations and safely fetches candidate homepages.
3. Scores company-name/title matches, domain-name similarity, and mail capability.
4. Crawls a small robots-aware set of public company pages such as `/about`, `/team`, `/people`, `/contact`, `/press`, `security.txt`, and relevant sitemap links.
5. Extracts real `@company-domain` emails from those pages.
6. Learns the dominant company email pattern when enough examples exist.
7. Generates a small ranked set of candidates for the requested person.
8. Runs SMTP-independent validity checks on each generated candidate.

The web crawler uses bounded concurrency, short timeouts, page limits, caching, same-domain link discovery, response-size limits, and SSRF protection. It does not use the SMTP ProxyScrape pool and does not scrape LinkedIn.

### Optional mailbox reachability — existing SMTP layer

- SMTP recipient response codes and enhanced status codes
- Catch-all behavior using multiple random non-existent recipients
- Mailbox-full / disabled / temporary / greylisted / rate-limited / policy-blocked responses
- Independent-route confirmation before a public proxy rejection can mark an address invalid

## Validity model

Every successful `/api/verify` response includes a `validity` object that is deliberately independent of SMTP and proxy availability.

`validity.status` is one of:

- `valid` — syntax is valid and the domain publishes MX records
- `invalid` — syntax is invalid or the domain does not publish MX records
- `risky` — the address uses a disposable domain or has a possible domain typo that should be reviewed

Example:

```json
{
  "validity": {
    "status": "valid",
    "scope": "address_and_domain",
    "reason": "syntax_and_domain_valid",
    "evidence": "The address syntax is valid and the domain publishes MX records. This validates the address/domain structure, not the existence of the individual mailbox.",
    "checks": {
      "syntax_valid": true,
      "has_mx": true,
      "disposable": false,
      "role_account": false,
      "free_provider": false
    }
  }
}
```

A `valid` result does **not** claim that the individual mailbox exists or belongs to a particular person. That is why mailbox reachability remains a separate signal.

## Finder API

### POST `/api/find-work-email`

Company name only:

```bash
curl -X POST http://localhost:8080/api/find-work-email \
  -H 'Content-Type: application/json' \
  -d '{"name":"Jane Smith","company":"Acme Labs"}'
```

If you already know the company domain, provide it to skip automatic domain resolution:

```bash
curl -X POST http://localhost:8080/api/find-work-email \
  -H 'Content-Type: application/json' \
  -d '{"name":"Jane Smith","company":"Acme Labs","domain":"acme.com"}'
```

The response includes:

- `domain_resolution` — selected domain, confidence, method, and high-scoring alternatives
- `observed_email_count` — public company-domain emails found
- `pages_fetched` — bounded public pages actually read
- `pattern` / `pattern_confidence` — inferred naming convention
- `candidates` — ranked work-email candidates, public source evidence when exact matches were found, and SMTP-independent validity

The older `POST /api/find-email` endpoint remains available when the caller already knows the company domain.

## Reachability model

SMTP checks have a total deadline, cancellation support, and recipient-specific classification. Policy blocks remain inconclusive. The UI shows verdict confidence and protocol evidence.

`reachability.status` is one of:

- `valid` — target accepted and random recipients were rejected (non-catch-all evidence)
- `invalid` — strong recipient-specific negative evidence, normally confirmed over multiple routes when public proxies are used
- `risky` — mailbox may exist but SMTP cannot prove it safely (catch-all, full mailbox, target accepted while catch-all test is inconclusive)
- `unknown` — route blocked, greylisted, rate-limited, unavailable, or otherwise not recipient-specific

`reachability.confidence` describes confidence in the displayed reachability verdict. It is **not** a probability that an email belongs to a real person.

### Why negative consensus?

Public proxy IPs are often blocked or poorly reputed. One `550`-style response is therefore not enough unless it is clearly recipient-specific and confirmed by the configured number of independent routes. The default is two routes.

### Catch-all detection

When the target is accepted, Valid Mail probes multiple random addresses at the same domain. If all random recipients are accepted, the domain is reported as catch-all and the target is `risky`, not `valid`. If all random recipients are explicitly rejected as missing, the target can be classified much more confidently.

## Run locally

Requires Go 1.25+.

```bash
go mod download
go test ./...
go run .
```

Open `http://localhost:8080` or `http://localhost:8080/finder`.

## Verify API

### POST `/api/verify`

```bash
curl -X POST http://localhost:8080/api/verify \
  -H 'Content-Type: application/json' \
  -d '{"email":"hello@example.com"}'
```

The response includes:

- `verification` — upstream syntax/domain metadata
- `validity` — SMTP-independent address/domain validity
- `reachability` — existing mailbox-level SMTP assessment when available

### GET `/api/verify?email=...`

```bash
curl 'http://localhost:8080/api/verify?email=hello@example.com'
```

### GET `/healthz`

Returns service health plus proxy-pool size and the active SMTP verification policy.

## SMTP / proxy behavior

Render Free blocks direct outbound SMTP, so the default deployment can use the free ProxyScrape SOCKS5 list. The pool only retains proxies that can open a real SMTP connection and read a `220` greeting.

Public proxies are untrusted and unstable. Valid Mail never sends authentication credentials or an email message body through them, and the API never exposes the proxy IPs it used.

The public-web finder does **not** use these proxies. Web discovery intentionally uses direct, rate-limited, robots-aware HTTP requests instead of proxy rotation for block evasion.

## Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port; Render injects this. |
| `ENABLE_SMTP` | `true` | Enable direct SMTP (requires outbound port 25). |
| `SMTP_PROXY_URL` | empty | Optional stable private SOCKS5 URI. Configure securely; this route takes priority over public proxies and direct SMTP. |
| `ENABLE_PROXY_SMTP` | `false` | Optional SMTP through the rotating SOCKS5 pool; enable explicitly. |
| `SMTP_FROM_EMAIL` | empty | Null envelope sender by default; configure a sender on a domain you control for better server acceptance. |
| `SMTP_HELLO_NAME` | empty | Hostname used for `EHLO`/`HELO`; use a domain you control. |
| `SMTP_CONNECT_TIMEOUT` | `7s` | SMTP connect timeout. |
| `SMTP_TOTAL_TIMEOUT` | `25s` | Total request budget across DNS, routes and protocol work. |
| `SMTP_OPERATION_TIMEOUT` | `8s` | SMTP operation deadline. |
| `SMTP_MAX_ATTEMPTS` | `3` | Maximum independent SMTP routes tried per request. |
| `SMTP_CATCHALL_PROBES` | `2` | Number of random recipient probes after the target is accepted. |
| `SMTP_NEGATIVE_CONSENSUS` | `2` | Independent recipient-missing replies required before marking invalid via public proxies. |
| `SMTP_MAX_CONCURRENCY` | `4` | Maximum simultaneous mailbox probes. |
| `PROXY_SOURCE_URL` | ProxyScrape free SOCKS5 API | Source for proxy candidates. |
| `PROXY_MAX_CANDIDATES` | `40` | Candidates tested each refresh (Render currently overrides this higher). |
| `PROXY_MAX_HEALTHY` | `8` | Maximum healthy proxies retained. |
| `PROXY_PROBE_CONCURRENCY` | `12` | Concurrent health probes. |
| `PROXY_PROBE_TIMEOUT` | `4s` | Per-proxy SMTP health timeout. |
| `PROXY_REFRESH_INTERVAL` | `5m` | Healthy-pool refresh interval. |
| `AUTO_UPDATE_DISPOSABLE` | `false` | AfterShip disposable-domain updater. |

## Notes on certainty

Syntax + MX validation can establish that an address is structurally plausible and its domain is configured to receive email. It cannot prove that the mailbox exists.

Pattern inference is also probabilistic. An inferred address should be treated as a candidate unless it was found directly on a public company page.

SMTP probing cannot bypass a provider that deliberately hides recipient existence. Catch-all domains can remain ambiguous. Definitive proof that a user controls an address still requires a delivered verification link/OTP or subsequent bounce processing.

## Dependency

The service pins `github.com/AfterShip/email-verifier` v1.5.0 or newer compatible version used by this project. v1.5.0 includes important SMTP-result and security fixes over v1.4.1.

## WeMe integration

The current interface and production build are documented in the root [README](../README.md). Build the TypeScript frontend before compiling Go. SMTP now defaults to disabled, and DNS transport failures are reported as inconclusive rather than invalid. The JSON APIs and separate SMTP assessment remain available.

Public proxy availability and recipient acceptance are outside the application’s control. A stable SOCKS5 route that permits mail-server port 25 can be configured as the `SMTP_PROXY_URL` secret. Credentials are never returned in results. SMTP sends no message body; it only negotiates the connection and recipient commands. Unknown results have no measured confidence in the UI.

## Verification depth and current UI

`POST /api/verify` accepts an optional boolean `smtp`. It defaults to true; `{ "email": "user@example.com", "smtp": false }` performs the quick address/DNS path without mailbox probes. GET requests can use `smtp=false`. The UI exposes the same choice and shows the check time, duration, configured/attempted SMTP state, route, reply, catch-all evidence and verdict confidence. Unknown mailbox results show “Not measured.”

SMTP normalizes internationalized domains before lookup, pins public destinations, prefers IPv4, and bounds alternate-IP attempts. Transport, policy and TLS failures do not establish a missing recipient. Configuration enables attempts; actual mailbox evidence still requires a working SMTP-capable route.
