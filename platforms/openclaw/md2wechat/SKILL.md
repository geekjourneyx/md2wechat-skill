---
name: md2wechat
description: Convert Markdown to WeChat Official Account HTML, inspect supported providers/themes/prompts, generate article images, create drafts, write with creator styles, and remove AI writing traces.
homepage: https://github.com/geekjourneyx/md2wechat-skill
metadata: {"clawdbot":{"emoji":"📝","requires":{"bins":["md2wechat"],"env":["WECHAT_APPID","WECHAT_SECRET"]},"install":[{"id":"brew","kind":"brew","formula":"geekjourneyx/tap/md2wechat","bins":["md2wechat"],"label":"Install md2wechat (brew)"},{"id":"go","kind":"go","module":"github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@latest","bins":["md2wechat"],"label":"Install md2wechat (go)"}]}}
---

# md2wechat

Use `md2wechat` when the user wants to:

- convert Markdown into WeChat Official Account HTML
- preview or upload drafts
- inspect live capabilities, providers, themes, and prompts
- generate covers, infographics, or other article images
- create image posts
- write in creator styles or remove AI writing traces

## Defaults And Config

- Use this skill only when `md2wechat` is already available on `PATH`.
- Draft upload and publish-related actions require `WECHAT_APPID` and `WECHAT_SECRET`.
- Image generation may require additional image-service configuration in `~/.config/md2wechat/config.yaml`.
- `convert` defaults to `api` mode unless the user explicitly asks for `--mode ai`.
- Check configuration in this order:
  1. `~/.config/md2wechat/config.yaml`
  2. environment variables such as `MD2WECHAT_BASE_URL`
  3. project-local `md2wechat.yaml`, `md2wechat.yml`, or `md2wechat.json`
- If the user asks to switch API domain, update `api.md2wechat_base_url` or `MD2WECHAT_BASE_URL`.
- Treat live CLI discovery output as the source of truth. Do not guess provider names, theme names, or prompt names from repository files alone.

## Discovery First

Run these before selecting a provider, theme, or prompt:

- `md2wechat version --json`
- `md2wechat capabilities --json`
- `md2wechat providers list --json`
- `md2wechat themes list --json`
- `md2wechat prompts list --json`
- `md2wechat prompts list --kind image --json`
- `md2wechat prompts list --kind image --archetype cover --json`

Inspect a specific resource before using it:

- `md2wechat providers show openrouter --json`
- `md2wechat themes show autumn-warm --json`
- `md2wechat prompts show cover-default --kind image --json`
- `md2wechat prompts show cover-hero --kind image --archetype cover --tag hero --json`
- `md2wechat prompts show infographic-victorian-engraving-banner --kind image --archetype infographic --tag victorian --json`
- `md2wechat prompts render cover-default --kind image --var article_title='Example' --json`

When choosing image presets, prefer the prompt metadata returned by `prompts show --json`, especially `primary_use_case`, `compatible_use_cases`, `recommended_aspect_ratios`, and `default_aspect_ratio`.

## Core Commands

Configuration:

- `md2wechat config init`
- `md2wechat config show --format json`
- `md2wechat config validate`

Conversion:

- `md2wechat convert article.md --preview`
- `md2wechat convert article.md --draft --cover cover.jpg`
- `md2wechat convert article.md --mode ai --theme autumn-warm --preview`

Image handling:

- `md2wechat upload_image photo.jpg`
- `md2wechat download_and_upload https://example.com/image.jpg`
- `md2wechat generate_image "A cute cat sitting on a windowsill"`
- `md2wechat generate_image --preset cover-hero --article article.md --size 2560x1440`
- `md2wechat generate_cover --article article.md`
- `md2wechat generate_infographic --article article.md --preset infographic-comparison`
- `md2wechat generate_infographic --article article.md --preset infographic-dark-ticket-cn --aspect 21:9`
- `md2wechat generate_infographic --article article.md --preset infographic-handdrawn-sketchnote`

Drafts and image posts:

- `md2wechat create_draft --file article.html --cover cover.jpg --title "Article Title" --desc "Article summary"`
- `md2wechat test-draft article.html cover.jpg`
- `md2wechat create_image_post -t "Weekend Trip" --images photo1.jpg,photo2.jpg`
- `md2wechat create_image_post -t "Travel Diary" -m article.md`
- `echo "Daily check-in" | md2wechat create_image_post -t "Daily" --images pic.jpg`
- `md2wechat create_image_post -t "Test" --images a.jpg,b.jpg --dry-run`

Writing and humanizing:

- `md2wechat write --list`
- `md2wechat write --style dan-koe`
- `md2wechat write --style dan-koe --input-type fragment article.md`
- `md2wechat write --style dan-koe --cover-only`
- `md2wechat write --style dan-koe --cover`
- `md2wechat write --style dan-koe --humanize --humanize-intensity aggressive`
- `md2wechat humanize article.md`
- `md2wechat humanize article.md --intensity aggressive`
- `md2wechat humanize article.md --show-changes`
- `md2wechat humanize article.md -o output.md`

## Agent Rules

- Start with discovery commands before committing to a provider, theme, or prompt.
- Prefer `generate_cover` or `generate_infographic` over a raw `generate_image "prompt"` call when a bundled preset fits the task.
- Validate config before any draft, publish, or image-post action.
- If the user asks for AI conversion or style writing, be explicit that the CLI may return an AI request/prompt rather than final HTML or prose unless the workflow completes the external model step.
- Do not perform draft creation, publishing, or remote image generation unless the user asked for it.

## Safety And Transparency

- Reads local Markdown files and local images.
- May download remote images when asked.
- May call external image-generation services when configured.
- May upload HTML, images, drafts, and image posts to WeChat when the user explicitly requests those actions.
