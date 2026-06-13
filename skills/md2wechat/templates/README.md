# Template Library

This directory stores reusable fallback templates for `md2wechat` article formatting.

## Current Templates

- `event-blue-card-inline.html.tmpl`
  - Origin: derived from a real WeChat event article publish flow
  - Use when: event or announcement content needs calm blue cards, clear sectioning, and fully inline HTML compatible with WeChat
  - Notes: supports overview blocks, info sections, body sections, CTA blocks, and inline images

## Usage Rules

- Start from the closest existing template before creating a new one.
- Prefer placeholder tokens over hardcoded live copy.
- Keep CSS inline and tag usage WeChat-safe.
- Avoid native `<ul>` / `<li>` for visible article lists. WeChat's editor/client can add default list spacing on top of inline styles; use separate `<p>` lines with `&bull;` and explicit margins for compact lists.
- If an AI prompt adds style guidance, apply only the deltas that matter instead of rewriting the template from scratch.
- Add templates with descriptive names tied to content goal and visual system.

## Placeholder Conventions

Use double-underscore tokens such as:

- `__EYEBROW__`
- `__TITLE__`
- `__SUMMARY__`
- `__OVERVIEW_LABEL__`
- `__OVERVIEW_HTML__`
- `__INFO_HEADING__`
- `__INFO_LIST_HTML__` (render as compact `<p>` lines with `&bull;`, not `<ul><li>`)
- `__BODY_SECTIONS_HTML__`
- `__CTA_HEADING__`
- `__CTA_PRIMARY_TEXT__`
- `__CTA_SECONDARY_TEXT__`
- `__QR_URL__`
- `__QR_ALT__`
- `__FOOTER_HTML__`

Keep placeholders explicit and content-shaped so they can be replaced by simple text processing.
