# Go PDF editor research and WeMe implementation

Research date: 14 October 2026. The links below are upstream GitHub repositories (not claims of guaranteed compatibility). Prefer publicly maintained packages with permissive licenses; inspect release notes, known limitations, and security advisories before upgrading.

| Library | GitHub | Primary strengths | Fit for WeMe |
| --- | --- | --- | --- |
| **SalvioniDigitalSolutions/gopdf** | https://github.com/SalvioniDigitalSolutions/gopdf | Pure-Go PDF parser/writer; change **existing text runs**, restyle, fill forms, render, annotations, redact, merge, visible overlays, X.509 signing support | **Implemented in PDF Studio.** MIT, no CGo or native binaries, but very new and fewer adopters than pdfcpu; use strict input limits and regression fixtures |
| **pdfcpu/pdfcpu** | https://github.com/pdfcpu/pdfcpu | Mature Go document tooling: validate, split, merge, reorder, rotate, watermark, stamp, encryption, annotations, forms, inspect signatures | Recommended alternative for hardening page operations. Apache-2.0. Native text-run replacement is not the same as text watermarking; don't label overlay as real editing |
| **klippa-app/go-pdfium** | https://github.com/klippa-app/go-pdfium | PDFium bindings, precise text/page extraction and rendering, forms, PDFium API | Good rendering/text selection strategy for complex PDFs; introduces PDFium runtime and deployment complexity (or WASM/subprocess). Not used in this Go Free instance |
| **digitorus/pdfsign** | https://github.com/digitorus/pdfsign | Certificate-backed digital signing, PKCS#7, signature validation, timestamp/cert operations | Future independent **cryptographic signing** feature, requiring user-controlled certificate/private key and secure key handling. Do not confuse a drawn signature with digital signing |
| **signintech/gopdf** | https://github.com/signintech/gopdf | Generate PDFs, embedded fonts, images, drawing/vector graphics | Useful for from-scratch authoring; not a complete existing-document text editor |
| **ledongthuc/pdf** | https://github.com/ledongthuc/pdf | Text and styled-text extraction | Useful for searchable text extraction; primarily a reader |
| **go-pdf/fpdf** | https://github.com/go-pdf/fpdf | High-level PDF creation and layout | Archived in March 2025; not recommended for a newly maintained editor |

## Implemented now

- `/tools/pdf`: dedicated responsive UI with upload, page navigation, original/edited preview, click-to-place mark coordinates, drawn/typed signature options, formatted text controls, page operations, downloadable output, and use-output-as-input for subsequent passes.
- `POST /api/pdf/inspect`: page dimensions, text runs, interactive fields, whether the PDF is encrypted or signed.
- `POST /api/pdf/preview`: page preview as PNG. If unsupported, the UI falls back to the browser's PDF renderer.
- `POST /api/pdf/edit` with `operation`: `replace` (existing content text), `restyle` (font/size/color), `text`, `sign` (visible signature mark), `watermark`, `highlight`, `rectangle`, `rotate`, `extract`, `reorder`, `merge`, `fill`, `redact`.
- In-memory uploads and temporary output files removed after response, 12 MB per PDF, 28 MB request budget, 120-page limit, page-bound checks, 2 MB signature image cap.

## What this editor does **not** claim

- Visual drawn/typed signatures **are not certificate-backed cryptographic signatures**. Add `digitorus/pdfsign` separately when a clear certificate upload/secure handling flow exists; never label the current UI as eIDAS or qualified electronic signing.
- Existing text editing requires the PDF to have usable text operators/fonts. Many scanner-only or flattened-image PDFs have no editable text. OCR is a separate, later feature.
- PDF layout is fixed. Text replacements are limited by available glyphs; complex multi-column reflow and arbitrary full-document WYSIWYG editing need a more advanced document model. The Go package can return an error instead of silently mangling a PDF.
- Redaction is for selectable text; a bitmap scan needs OCR + removal of underlying image pixels, not a superficial black overlay.
- Editing any certificate-signed document can invalidate its signature; this UI explicitly warns users.
- Neither password-protected nor intentionally malformed PDFs are universally supported. Use on a copy and inspect the downloaded result.

## Testing/acceptance

The Go tests cover PDF creation, inspect, replacement with text extraction evidence, merge/extract/rotate, image signature, true redaction, PNG preview, malformed upload and out-of-range pages. The main CI runs the full frontend build, Go race tests and Playwright browser smoke tests. For broader compatibility, add a corpus of real PDFs from Word, Google Docs, Acrobat, scanned/image-only sources, multi-column reports, form-heavy PDFs, and PDFs with signatures/encryption.

## Useful next additions

1. Certificate-backed digital signatures via digitorus/pdfsign, with key material handled ephemerally, explicit disclosure and verification.
2. Browser PDF.js selection geometry for precise drag-and-drop editing, including multi-page editing and an undo/redo operation list.
3. OCR and safer image-aware redaction, using a worker service rather than blocking the Render Free web service.
4. pdfcpu validation/optimization/encryption/password-protected operations, plus compatibility fallbacks for PDFs that the young pure-Go editor cannot process.
