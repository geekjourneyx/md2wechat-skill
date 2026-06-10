---
name: md2wechat
description: Convert Markdown to WeChat Official Account HTML and drafts with discovery-first CLI commands.
---

# md2wechat CLI Skill

This embedded skill is bundled with the installed binary. Prefer it over repository docs when operating an unknown md2wechat version.

## Discovery First

Use the CLI as the source of truth:

```bash
md2wechat version --json
md2wechat capabilities --json
md2wechat skills read md2wechat --json
```

Only run catalog discovery when it affects the next decision:

```bash
md2wechat themes list --json
md2wechat layout list --json
md2wechat providers list --json
md2wechat prompts list --kind image --json
md2wechat doctor --json
```

## Safe Workflow

For article work, confirm before executing side effects:

```bash
md2wechat inspect article.md --json
md2wechat preview article.md --json
md2wechat convert article.md --preview
```

Add `--upload`, `--draft`, `--cover`, or `--cover-media-id` only when the user explicitly asks for upload or draft creation.

## Decision Boundaries

- `convert` defaults to API mode.
- Advanced `:::module` layout blocks render only in API mode.
- AI mode does not render advanced layout modules.
- `preview` is local and does not upload images or create drafts.
- `doctor --json` is local readiness only; it does not validate live auth.
- `inspect --json` is the article-specific source of truth for `data.readiness.targets` and `data.readiness.blockers`.

## Side Effects

- Read-only discovery commands: `version`, `capabilities`, `skills`, `providers`, `themes`, `prompts`, `layout list/show`, `doctor`, `inspect`.
- Local file writes: `preview`, `convert --output`, `write --output`, `humanize --output`, generated image commands.
- Network or WeChat side effects: image generation providers, `upload_image`, `download_and_upload`, `convert --upload`, `convert --draft`, `create_draft`.

## Failure Handling

- Unknown theme or mode mismatch: run `md2wechat themes list --json` and choose a `selectable: true` theme whose `type` matches the mode.
- Layout syntax errors: run `md2wechat layout validate --file <file> --json`, inspect the failing module with `layout show`, then validate again.
- Draft blockers: read `inspect --json` readiness before assuming WeChat credentials or cover media are valid.
- Configuration uncertainty: run `md2wechat doctor --json` and `md2wechat config show --format json`.
