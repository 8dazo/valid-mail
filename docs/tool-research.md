# WeMe tool foundations

The collection contains 39 implemented utilities. All utility logic is written in Go: 37 tools execute in a browser WebAssembly worker, while email validation and company-page discovery use the Go HTTP server. The frontend remains React/TypeScript and uses Highlight.js for visual syntax highlighting.

See [competitor research and package selection](competitor-research.md) for source evidence, release dates, implementation decisions, known boundaries and the next collections. The source catalog in `frontend/src/catalog.ts` links each tool to its upstream repository. Exact Go module versions are in `go.mod`, checksums in `go.sum`, and web dependencies in `frontend/package-lock.json`.

WeMe wraps library APIs and standard-library algorithms. The thin Go core lives in `internal/toolkit`; both `cmd/weme-wasm` and `cmd/weme-toolkit` call it. It does not copy the implementation or branding of competitor applications.

## Current collection

- Validation: email, phone, URL, IP, company-email finder and public certificate inspection.
- Conversion: YAML, XML, CSV, TOML, INI, HTML-to-Markdown, color formats, Base64, timestamps, URL components and gzip.
- Generators: QR, Code 128/39/EAN barcodes, secure passwords, UUID/Nano ID and ULID.
- Developer: JSON format/repair, JWT decoding, cron descriptions/forecasts, regex, HTML/CSS/JavaScript/SVG minifiers, SHA, HMAC, bcrypt and Unix permissions.
- Text: Markdown/README preview, semantic diffs, Unicode case conversion and grapheme/word statistics.

## Product boundaries

Local tools do not upload their inputs. External Markdown images are disabled by default and can be enabled explicitly. Email/finder inputs reach the Go server and relevant mail servers or company pages. Inputs/results are not persisted by the interface; optional local storage holds favorite/recent tool IDs.

Verification stays within its evidence: email syntax/DNS does not establish mailbox ownership; SMTP may be inconclusive; phone validation does not establish an active subscriber; JWT decoding does not authenticate a signature; certificate inspection does not establish chain trust or revocation status. Gzip, regex, YAML/XML, bcrypt and generated images have explicit processing limits.
