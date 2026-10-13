# WeMe tool collection

Research checked npm package metadata (current versions, licensing, repository identity, and engine requirements) and the upstream GitHub READMEs. The selection prioritizes useful daily tasks, browser support, maintained dependencies, permissive library licenses, and workflows that do not need API keys. Packages are installed through their registries; WeMe wraps their public APIs rather than copying other applications.

| Tool | Upstream foundation | Why it belongs | Execution |
| --- | --- | --- | --- |
| Email validator | [AfterShip/email-verifier](https://github.com/AfterShip/email-verifier), v1.5.0 | Syntax, disposable/role/free-domain metadata, typo suggestions; bounded Go MX lookup | Server |
| Phone validator | [libphonenumber-js](https://github.com/catamphetamine/libphonenumber-js), v1.13.15 | International metadata and E.164 formatting | Browser |
| URL inspector | [validator.js](https://github.com/validatorjs/validator.js), v13.15.35 | HTTP/HTTPS validation; native URL decomposition | Browser |
| IP inspector | [ipaddr.js](https://github.com/whitequark/ipaddr.js), v2.5.0 | IPv4/IPv6 parsing, CIDR masks, network addresses | Browser |
| JSON formatter & repair | [jsonrepair](https://github.com/josdejong/jsonrepair), v3.15.0 | Fix common pasted JSON issues, with a review notice | Browser |
| JSON ↔ YAML | [yaml](https://github.com/eemeli/yaml), v2.9.1 | YAML parsing with duplicate-key and alias limits | Browser |
| XML ↔ JSON | [fast-xml-parser](https://github.com/NaturalIntelligence/fast-xml-parser), v5.11.2 | XML validation and structured conversion, without DTD/entity expansion | Browser |
| CSV ↔ JSON | [Papa Parse](https://github.com/mholt/PapaParse), v5.7.0 | Handles quoting and escaping; exports guard against spreadsheet formulas | Browser |
| JWT decoder | [jwt-decode](https://github.com/auth0/jwt-decode), v4.0.0 | Inspect headers/payloads locally, without asserting authenticity | Browser |
| Cron explainer | [cRonstrue](https://github.com/bradymholt/cRonstrue), v3.32.0 | Readable schedule descriptions | Browser |
| QR code generator | [node-qrcode](https://github.com/soldair/node-qrcode), v1.5.4 | Error correction and downloadable scalable SVGs | Browser |
| Markdown preview | [Marked](https://github.com/markedjs/marked), v18.1.1 + [DOMPurify](https://github.com/cure53/DOMPurify), v3.4.17 | Markdown with an explicit safe HTML allowlist | Browser |
| Text compare | [jsdiff](https://github.com/kpdecker/jsdiff), v9.0.0 | Added/removed words, with input and execution limits | Browser |
| UUID & Nano ID | [Nano ID](https://github.com/ai/nanoid), v6.0.2 + Web Crypto | Secure random identifiers | Browser |
| Password generator | Web Crypto API | Unbiased cryptographic selection; required character groups | Browser |
| SHA hash generator | [noble-hashes](https://github.com/paulmillr/noble-hashes), v2.4.0 | SHA-256/SHA-512 implementations with standard test vectors | Browser |
| Base64 encoder/decoder | TextEncoder, TextDecoder, atob/btoa | Unicode-safe encoding; strict UTF-8 decoding | Browser |
| Timestamp converter | Date / Intl | Explicit seconds/milliseconds, ISO timezone validation | Browser |
| Text case converter | [change-case](https://github.com/blakeembrey/change-case), v5.4.4 | Consistent naming formats and Unicode-aware word splitting | Browser |
| URL encoder/decoder | encodeURIComponent / decodeURIComponent | Query-component escaping, without sending data to another service | Browser |
| Work email finder | Existing Go public-page crawler + AfterShip | Preserve company-page discovery in the shared workspace | Server |

The libphonenumber-js npm package declares its GitLab repository as canonical; the linked GitHub repository provides its public README/mirror. Repository links are source attribution, not sponsorship or endorsement.

## Product and privacy choices

- All 21 tools are implemented. Nineteen run in the browser, with libraries loaded on demand.
- Email and work-email discovery call the same-origin Go server and require native DNS. Discovery also requires outbound company-site HTTP/HTTPS.
- No accounts or third-party analytics. Favorites and recent tool IDs use optional localStorage; inputs, outputs, passwords, and tokens are never persisted by the interface.
- Email validation is address/domain evidence. Phone validation is number-format evidence. JWT decoding is not signature verification. None claims account ownership.
- The visual reference informed a centered serif layout, teal controls, a coastal hero, and a scrolling library strip. The WeMe emblem is an original SVG; coastal artwork is generated, with pixel/dither texture. Fonts are self-hosted Instrument Serif and DM Sans.
- Actual sponsors are not claimed. The strip names libraries used by the application.

## Deliberately deferred

WHOIS, reputation scoring, live exchange rates, SEO crawling, and paid enrichment providers need new external services, credentials, freshness guarantees, or rate-limit policies. They are better added as a second collection once those requirements are defined.
