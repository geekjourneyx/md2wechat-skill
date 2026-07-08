# Article Advice

`md2wechat advise <article.md> --json` reads one existing Markdown article and recommends the smallest useful next actions.

Use it when an article or draft already exists and you are unsure whether it needs title, cover, layout, or light wording improvements.

```bash
md2wechat advise article.md --json
```

`advise` is a recommendation-only planner. It does not edit the article, rewrite text, generate titles, generate covers, insert layout modules, upload images, create drafts, or write files.

The top-level decision is one of:

- `no_enhancement_needed`: no optional enhancement is recommended.
- `enhance_minimally`: apply only the recommended small set of next actions.

Recommended tools may include:

- `title`: run title suggestion workflow.
- `cover`: plan or generate a cover separately.
- `layout`: inspect and apply suitable layout modules separately.
- `micro_edit`: make a small human-confirmed edit outside `advise`.

`data.actions` contains command-level recommendations and may include both `command_hint` and structured `command_args`. `data.micro_edits` is a separate list for small human-confirmed wording or structure changes; it is not an execution command.

Use `inspect` before publish decisions. `inspect --json` blocks publish targets through `data.readiness.targets/blockers`; `advise --json` only recommends optional enhancement.
