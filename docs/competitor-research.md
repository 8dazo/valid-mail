# Utility-suite research and implementation decisions

Research date: 13 October 2026. Evidence comes from upstream repository READMEs, source catalogs, published Go module metadata, documented APIs, and local build/tests. This is a source-based comparison, not a hands-on audit of every competitor feature. Release dates indicate published versions; an older stable tag alone does not prove abandonment or active maintenance.

## Competitors

| Suite | Verified source evidence | Strong workflows | What WeMe adopts |
| --- | --- | --- | --- |
| [IT-Tools](https://github.com/CorentinTh/it-tools) | Its [source catalog](https://github.com/CorentinTh/it-tools/blob/main/src/tools/index.ts) contains 86 tool imports. It includes regex, bcrypt, color conversion, HMAC, chmod, text statistics, encoders, hashes, JWT, QR and network utilities. | Fast selection of small developer tasks; separate controls for each operation; client-side utilities and self-hosting. | Regex matches/groups, bcrypt, colors, HMAC, permissions and text analysis, with examples and downloadable output. |
| [DevToys](https://github.com/DevToys-app/DevToys) | Its README describes 30 default tools, including GZip, JWT/QR, formatters, hashes, image compression, Markdown preview and text comparison. It also describes clipboard smart detection and extensions. | Local processing, readable editors, dedicated graphics/text workspaces. | Gzip with a real binary download, improved Markdown/code rendering, structured inspector results, local processing and file import. |
| [CyberChef](https://github.com/gchq/CyberChef) | Its README documents encoding, compression, cryptographic and data-manipulation operations, recipes, saved recipes, file input and local execution. | Composable transformations and clear input/output boundaries. | Safe individual operations, explicit encode/decode controls, preserved Unicode/integers, downloadable artifacts and bounded expensive operations. Multi-step recipes remain a future feature. |
| [OmniTools](https://github.com/iib0011/omni-tools) | Its README lists image/video/audio, PDF, text/list, time, math and JSON/CSV/XML collections, with a self-hosted browser app. | Broad everyday tasks; controls shaped around the data rather than one universal text box. | Tool-specific workspaces, compact generators, document previews, data tables, swatches, barcode images and file import. Media/PDF operations need a separate resource budget. |

IT-Tools is GPL-3.0, CyberChef is Apache-2.0, and OmniTools declares MIT. We compare product workflows and use independently selected Go packages; their application implementations and branding are not copied into WeMe.

## What was implemented

WeMe now contains **39 tools**: **37 local Go/WebAssembly tools** plus email verification and company-email discovery in the Go server. Eighteen tools were added, and existing browser tool logic was migrated to Go packages or Go's standard library. React/TypeScript handles the interface; Highlight.js handles visual code highlighting, not utility algorithms.

| New tool | Competitor gap addressed | Foundation |
| --- | --- | --- |
| Regex tester | Match/group inspection | `dlclark/regexp2` |
| TOML ↔ JSON | Configuration interchange | `pelletier/go-toml/v2` |
| INI ↔ JSON | Sectioned configuration | `go-ini/ini` through canonical `gopkg.in/ini.v1` |
| HTML, CSS, JavaScript, SVG minifiers | Four practical publishing/build operations | `tdewolff/minify/v2` |
| HTML → Markdown | Documentation conversion | `JohannesKaufmann/html-to-markdown/v2` |
| Barcode generator | Code 128, Code 39, EAN labels | `boombuler/barcode` |
| Color converter | HEX/RGB/HSL/HSV and tint/shade preview | `lucasb-eyer/go-colorful` |
| ULID generator/inspector | Timestamp-bearing identifiers | `oklog/ulid/v2` |
| Bcrypt hash/verify | Local password-storage workflow | `golang.org/x/crypto/bcrypt` |
| Text analyzer | Unicode-aware counts | `rivo/uniseg` |
| Certificate inspector | Local public-certificate inspection | Go `crypto/x509` |
| Cron schedule planner | Five upcoming runs with IANA zones | `robfig/cron/v3` and embedded timezone data |
| Gzip text compressor | Copyable Base64 and real `.gz` download | Go `compress/gzip` |
| HMAC generator | Shared-secret message authentication | Go `crypto/hmac` |
| Unix permissions calculator | Octal and symbolic permissions | Go octal parsing and Unix permission masks |

## Go package selection

Versions are pinned in `go.mod` and verified with `go.sum`. The module proxy supplied these release timestamps during research.

| Package | Selected version | Published | Selection notes |
| --- | --- | --- | --- |
| [Goldmark](https://github.com/yuin/goldmark) | v1.8.6 | 2026-09-03 | GFM, tables, task lists, fenced code, footnotes; rendering is followed by sanitization. |
| [bluemonday](https://github.com/microcosm-cc/bluemonday) | v1.0.27 | 2024-07-04 | Allowlist HTML sanitizer; executable elements and event attributes are removed. |
| [minify](https://github.com/tdewolff/minify) | v2.24.19 | 2026-10-03 | Pure Go parsers/minifiers; code is displayed, not executed. |
| [go-toml](https://github.com/pelletier/go-toml) | v2.4.3 | 2026-07-05 | TOML 1.0 parsing and marshaling; signed 64-bit integer boundary is explicit. |
| [go-ini](https://github.com/go-ini/ini) | v1.67.3 | 2026-06-08 | Mature section/key API; canonical module path is `gopkg.in/ini.v1`. INI values remain strings. |
| [regexp2](https://github.com/dlclark/regexp2) | v1.12.0 | 2026-04-18 | Named groups/lookarounds; 100 ms matching budget plus a worker deadline. It is .NET-compatible, not JavaScript RegExp. |
| [barcode](https://github.com/boombuler/barcode) | v1.1.0 | 2025-07-22 | Pure Go encoding; wrapper adds scanner quiet zones and bounded raster sizes. |
| [go-colorful](https://github.com/lucasb-eyer/go-colorful) | v1.4.1 | 2026-08-02 | sRGB/HSL/HSV and Lab blending; alpha is explicitly unsupported. |
| [ULID](https://github.com/oklog/ulid) | v2.1.2 | 2026-07-23 | Crypto-random IDs and strict overflow-safe decoding. |
| [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) | v2.5.2 | 2026-06-07 | Purpose-built HTML conversion; no page fetch or script execution. |
| [uniseg](https://github.com/rivo/uniseg) | v0.4.7 | 2024-02-08 | Grapheme/word boundaries; combined emoji count as one grapheme. |
| [cron](https://github.com/robfig/cron) | v3.0.1 | 2020-01-04 | Stable scheduling API; used for forecasts only, with five results and no jobs registered. |
| [x/crypto](https://github.com/golang/crypto) | v0.58.0 | 2026-10-09 | Bcrypt with 72-byte password handling and bounded cost. Requires Go 1.26+. |
| [pretty](https://github.com/tidwall/pretty) | v1.2.2 | 2026-09-27 | Formats JSON bytes without rounding integer tokens. |
| [jsonrepair](https://github.com/kaptinlin/jsonrepair) | v0.4.9 | 2026-08-23 | Repairs malformed JSON only on request; valid input bypasses repair and preserves numeric tokens. |
| [Go YAML](https://github.com/yaml/go-yaml) | v3.0.5 | 2026-07-26 | Maintained canonical module `go.yaml.in/yaml/v3`; explicit graph/alias guard before expansion. |
| [mxj](https://github.com/clbanning/mxj) | v2.7.0 | 2023-02-10 | XML/map conversion; standard XML validation blocks multiple roots, DTD/entities and excessive nesting. Text values remain strings. |
| [phonenumbers](https://github.com/nyaruka/phonenumbers) | v1.8.1 | 2026-07-15 | International numbering metadata; validation is not subscriber verification. |
| [go-qrcode](https://github.com/skip2/go-qrcode) | da1b6568686e | 2020-06-17 | Stable pure Go QR encoder; matrix rendered to scalable SVG, with standard quiet zones. |
| [go-password](https://github.com/sethvargo/go-password) | v0.4.0 | 2026-07-17 | Crypto-random generation; wrapper checks required upper/lowercase groups. |
| [jwt](https://github.com/golang-jwt/jwt) | v5.3.1 | 2026-01-28 | Unverified parsing is intentionally separated from authentication. |
| [cron descriptions](https://github.com/lnquy/cron) | v1.1.1 | 2020-11-23 | Stable human-readable cron descriptions with 24-hour formatting. |
| [go-diff](https://github.com/sergi/go-diff) | v1.4.0 | 2025-06-05 | Bounded semantic text diffs. |
| [uuid](https://github.com/google/uuid) | v1.6.0 | 2024-01-23 | Crypto-random UUIDv4. |
| [go-nanoid](https://github.com/matoous/go-nanoid) | v2.1.0 | 2024-05-16 | Crypto-random URL-safe IDs. |
| [strcase](https://github.com/ettle/strcase) | v0.2.0 | 2023-01-14 | Unicode support; chosen over an inspected ASCII-only alternative. |

The GPL-licensed `RealAlexandreAI/json-repair` candidate was rejected after license inspection and replaced with the MIT-licensed `kaptinlin/jsonrepair` port. An inspected ASCII-only case converter was also rejected in favor of Unicode-aware `ettle/strcase`.

Go standard-library packages cover URL/IP parsing, CSV, Base64, SHA/HMAC, gzip, timestamps and certificate inspection. Their upstream source is [golang/go](https://github.com/golang/go); WeMe adds validation, result metadata and controls, not replacements for those algorithms.

## Issues corrected during the audit

- Markdown/README previews stripped code-language classes and useful document structure. Goldmark plus a narrow sanitizer now preserves fenced code, tables, task lists and details. Monospace styling and syntax highlighting keep code readable. HTML downloads include standalone document styling.
- Arbitrary README images could trigger requests or break relative asset paths. Images are placeholders by default; HTTPS image loading is an explicit choice with an optional asset base URL. Fragment links stay in the document.
- JSON formatting through JavaScript numbers could round large integers. The formatter now works on validated JSON bytes. CSV exports retain numeric tokens; unsupported configuration-number ranges produce errors rather than silent coercion.
- Universal raw output made inspectors and datasets difficult to read. Tables, match highlights, swatches, image previews, generator lists and a separate raw-output control now match the tool's shape.
- Field counters/helpers changed accessible input names. Controls now have stable labels. Long-document layouts, stacked panels, expanded width, line wrapping and file import provide room for larger inputs and outputs.
- Editing/cancelling a running check could leave stale output. Input changes invalidate results and cancel the outstanding request/worker operation.
- Generated image downloads must contain image bytes, not JSON with an image extension. Barcode PNG and gzip binary downloads have dedicated byte handling.
- Generic rate-language could hide a structured mailbox-full reply. Enhanced SMTP mailbox-state codes take priority. Public destinations are pinned against rebinding, with IPv4 first and bounded alternate-IP attempts; IDN domains are normalized before SMTP DNS lookup.

## Resource and privacy choices

The Go toolkit is compiled to a local WebAssembly worker. It is loaded on demand, fingerprinted, gzip-served, and reused across tool interactions. The broad library collection has a substantial first-load artifact; the interface supports cancellation and a separate startup budget. Inputs are not sent to the server by local tools. The native JSON-lines CLI and browser use the same Go core, so library logic is not duplicated in TypeScript.

Validation limits include 100,000 input characters, bounded regex work, limited CSV columns, YAML graph/alias limits, XML nesting limits, bcrypt cost/byte limits, bounded barcode sizes, gzip output limits and a capped result size. Details are reflected in tool errors and notices.

SMTP configuration alone cannot guarantee reachability. A real mail-server connection and usable route are required. The currently claimed Vercel site still serves the old SMTP-disabled deployment; updating its repository does not authenticate or redeploy that account-owned project. Render Free/direct egress and public SOCKS5 availability also require runtime verification. A private SMTP-capable route and sender identity can improve successful checks; connection failures remain unknown, never a fabricated percentage.

## Next collections

| Candidate | Go foundation to investigate | Why it is deferred |
| --- | --- | --- |
| Image resize/format conversion | [disintegration/imaging](https://github.com/disintegration/imaging) | Needs binary upload UX, decoded-pixel limits, codec coverage and mobile memory tests. |
| PDF merge/split/metadata | [pdfcpu](https://github.com/pdfcpu/pdfcpu) | Needs multi-file controls, file/page limits, a separate compute budget and fixture coverage. |
| DNS/SPF/DMARC diagnostics | [miekg/dns](https://github.com/miekg/dns), [emersion/go-msgauth](https://github.com/emersion/go-msgauth) | Requires supported DNS egress and honest handling of incomplete authorization evidence. |
| Multi-step recipes | Reuse the current Go core | Needs typed handoffs, binary/text boundaries, undo and per-step errors before it can match CyberChef's workflow. |
| SQL format/schema tools | Evaluate dialect-specific Go parsers | A formatter should preserve the selected SQL dialect and avoid implying query safety. |

These are researched directions, not implemented features or claims of full competitor parity.
