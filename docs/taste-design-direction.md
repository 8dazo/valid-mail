# WeMe interface art direction — taste-skill adaptation

Reference: [Leonxlnx/taste-skill](https://github.com/Leonxlnx/taste-skill), MIT-licensed design guidance. This is an adaptation of its `taste-skill` and `redesign-skill` principles, not a cloned template or copied stylesheet.

## What belongs to WeMe

WeMe is a free, open-source collection of 40 practical tools. Some run locally in the browser; email checks and PDF work may use the Go backend. The actual tool counts are rendered from the code, not invented social proof. The approved coastal photography, paperwork, hands artwork, and featured tool illustrations are kept.

**Page idea:** A messy input turns into a useful result, as if opening up a working desk. This gives the landing page the shape of a sequence of real tasks—check, tidy, make, then use the actual toolkit—instead of stock software marketing.

## Recomposition

- Hero: preserve the approved clear-water panorama and the primary action to the directory. Better contrast, editorial display type, no fake testimonials.
- Proof: a single inventory band with actual counts, instead of four unrelated candy-gradient statistic cards.
- Features: one wider anchor card followed by two compact companions. Individual image assets stay meaningful and separate from interactive text.
- Human connection: keep the existing edge-to-edge hands composition; no claims that all tools use AI.
- JSON demo: the real interface remains the centerpiece; small illustration is no longer glued to the lower edge.
- Featured tools: alternate large type and supplied illustration as an editorial index, breaking the three equal-card template.
- Directory: clear search, readable category tags, real local/server labels, keyboard-visible focus, and calm cards designed for scanning.
- Price: the factual single free plan and included features. Original full-width photograph remains; remove superficial floating sparkles and use a solid, legible card.
- FAQ/footer: full answers remain. The existing approved panorama and real HTML links stay intact.
- Application screens and PDF Studio: consistent text, input, selection, focus and loading contrast without forcing marketing photography into editing workspaces.

## Behavior and constraints

Keep all 40 tool routes and implementation APIs unchanged. Retain successful email routing and the server/browser processing distinctions. No invented testimonials, external tracking, fake product previews, paid pricing, or non-working actions. Honor reduced-motion and keyboard focus; avoid horizontal scroll on mobile.

## Validation

GitHub Actions runs frontend build, unit tests, Go test/race, and Playwright desktop/mobile tests. The new Playwright coverage specifically checks asymmetrical section geometry, UI focus, search, tool routing, and mobile overflow. CI tests are not a substitute for a full manual visual review of screenshots; use the Playwright trace and output when refining the design.
