# Copilot Instructions — md2wechat

**md2wechat** is a Go CLI tool that converts Markdown to WeChat Official Account format, with Claude Code and OpenClaw Skill support.

## Build, Test, Lint

```bash
# Build for current platform
make build

# Run all tests (use GOCACHE to avoid conflicts)
GOCACHE=/tmp/md2wechat-go-build go test ./...

# Run a single package test
GOCACHE=/tmp/md2wechat-go-build go test ./cmd/md2wechat/...

# Run a single test by name
GOCACHE=/tmp/md2wechat-go-build go test ./cmd/md2wechat -run TestRunVersionOutputsJSONEnvelope

# Full CI-equivalent gate (run this before any release or CI-sensitive work)
make quality-gates

# Individual checks
gofmt -l .       # formatting check
go vet ./...     # static analysis
make lint        # golangci-lint (same as CI)
make release-check  # version/doc consistency
```

`make quality-gates` runs: format check → `go vet` → golangci-lint → `go test -count=1` → npm pack dry-run → `make release-check`. It is the authoritative local gate — identical to what GitHub Actions runs.

## Architecture

The main pipeline is:

```
cmd/md2wechat  →  inspect/preview (confirmation layer)  →  publish orchestrators  →  AssetPipeline  →  draft/wechat adapters
```

- **`cmd/md2wechat/`** — Cobra commands only. Handles arg parsing, calls internal packages, emits JSON envelopes and exits. No business logic.
- **`internal/inspect`** — Single source of truth for resolved metadata, readiness, and publish checks. `preview` consumes inspect output; it does not re-implement business rules.
- **`internal/preview`** — Read-only HTML confirmation page rendered from inspect state.
- **`internal/publish`** — Main orchestration for article and image-post publish flows. Contains `AssetPipeline` (image upload/generate/download/rewrite) and `model.go` (canonical article/asset/artifact types).
- **`internal/converter`** — Markdown → HTML, frontmatter extraction, image ref parsing, AI/API conversion.
- **`internal/image`** — Image generation, compression, upload/download (injected at runtime).
- **`internal/draft`** — WeChat draft adapter (standard article + `newspic` image post).
- **`internal/wechat`** — WeChat SDK wrapper, material upload with retry, SSRF guard.
- **`internal/promptcatalog`** — Loads prompt YAML assets from `internal/assets/builtin/prompts/`.
- **`internal/config`** — Config loaded from file (`~/.md2wechat.yaml`) then environment variables.

**Two skill paths share the same CLI binary:**
- `skills/md2wechat/` — Claude Code / Codex / OpenCode agent skill
- `platforms/openclaw/md2wechat/` — OpenClaw / ClawHub structured skill

## Key Conventions

### Discovery-first
Before assuming any provider, theme, or prompt exists, query the running CLI:
```bash
md2wechat capabilities --json
md2wechat providers list --json
md2wechat themes list --json
md2wechat prompts list --json
md2wechat layout list --json      # 56 recommended advanced layout syntax names
```
These are the source of truth. Do not guess from docs or stale memory.

### Layout Module Discovery
Advanced layout discovery defaults to 56 recommended syntax names across 9 categories (`opening`, `infographic`, `judgment`, `evidence`, `conversion`, `brand`, `sprint4`, `free-layout`, `interactive`). Three compatibility modules are listed separately for migration:
```bash
md2wechat layout list --json                           # list recommended modules
md2wechat layout list --lifecycle compatibility --json # migration only
md2wechat layout list --serves attention --json        # filter by goal
md2wechat layout show <name> --json                    # inspect a module
md2wechat layout render <name> --var KEY=VALUE         # render syntax block
md2wechat layout validate --file article.md --json     # validate syntax in file
```

Counts use distinct dimensions: 77 recommended source scenarios, 56 recommended syntax names, 3 compatibility modules, 4 base enhancements, and 63 render-layer syntax capabilities. Read a selected `layout show` in this order: `input_positions`, primary `body_format` plus `Opener` / `Fields` / `Rows` / `Body`, canonical `Variants[].Name`, then canonical `Example`; `CompatibleBodyFormats` and `Variants[].Aliases` are migration-only compatibility facts.

**重要约束**：高级排版模块使用通用 `:::name` 语法，仅在 **API 模式**下渲染；`block` 是保留语法名，不是通用占位符。
- `convert`（默认）= API 模式，模块正常渲染
- `convert --mode ai` = AI 模式，`:::name` 高级排版块不渲染，以普通段落输出

Keep one canonical executable example per recommended syntax and add examples only for structurally distinct variants. The 77-scenario mapping stays test-only. Real conformance covers 84 witnesses: 56 canonical, 25 structurally distinct non-default branches, and 3 compatibility witnesses. It must prove semantic markers/stable content with no raw fence; local validation alone is not remote deployment proof.

The 4 `serves` values that every module is mapped to: `attention` | `readability` | `memorability` | `conversion`.

### Brand Profile
Users can define a personal/brand style guide at `~/.config/md2wechat/brand.md` (Markdown format):
```bash
md2wechat brand init      # initialize default brand.md
md2wechat brand show      # display current brand profile
```
The file is read by agents for voice/style context; it is not parsed by the CLI. It is a private local file — never commit it to the repository.

### Local-Only Artifacts
`docs/superpowers/` (plans, specs, working notes) is gitignored and must never be committed. Keep all superpowers files local.

### E2E Rendering Smoke Test (required before every release)

Before tagging or releasing, verify that advanced layout syntax renders correctly through the real API:

```bash
# Build latest CLI
make build

# Convert a test file with core layout modules (--mode api hits localhost:3000)
./md2wechat convert examples/layout-e2e-test.md --mode api --output /tmp/layout-smoke.html

# Check no raw ::: syntax remains in the output HTML
python3 -c "
modules = ['hero','toc','verdict','audience-fit','myth-fact','metrics','compare','steps',
           'timeline','quote','callout','definition','author-card','subscribe',
           'faq','checklist','cta','notice','summary']
html = open('/tmp/layout-smoke.html').read()
failed = [m for m in modules if ':::' + m in html]
ok = [m for m in modules if ':::' + m not in html]
print(f'PASS {len(ok)}/{len(modules)}') if not failed else print('FAIL - not rendered:', failed)
"

# Validate syntax is correct before running convert
./md2wechat layout validate --file examples/layout-e2e-test.md --json
```

**Pass criteria:** Both checks pass (0 raw residuals, 0 validation errors). Do not tag or push a release until both are green.

The canonical test file is `examples/layout-e2e-test.md`. Update it when adding new modules.



### JSON Envelope Contract
Every command emits a stable JSON envelope (schema version `v1`):
```json
{
  "success": true,
  "code": "CODE_CONSTANT",
  "message": "human text",
  "schema_version": "v1",
  "status": "completed|action_required|failed",
  "retryable": false,
  "data": {},
  "error": ""
}
```
All `code` constants are declared in `cmd/md2wechat/main.go`. Adding a new command requires a new constant and a contract test in `main_contract_test.go`.

### Configuration Naming Layers
Three layers exist and must never be mixed in docs or guidance:
- **Config file YAML keys** — e.g., `api.image_base_url`
- **Environment variables** — e.g., `IMAGE_API_BASE`
- **`config show --format json` output keys** — e.g., `image_api_base`

### Prompt Catalog (YAML, not Go code)
Image/humanizer/refine prompts live in `internal/assets/builtin/prompts/`. Do not embed long prompts directly in Go code. Humanizer supports 4 intensity levels: `gentle` / `medium` / `aggressive` / `authentic` — the `authentic` mode uses a standalone YAML prompt (`authentic.yaml`) that bypasses the base template. Every new `image` prompt YAML must include: `name`, `kind`, `description`, `version`, `archetype`, `primary_use_case`, `recommended_aspect_ratios`, `default_aspect_ratio`, `metadata.author`, `metadata.provenance`, `template`. The `default_aspect_ratio` must appear in `recommended_aspect_ratios`.

### Test Discipline
Tests protect contracts, not coverage numbers. Before writing any test, ask:
1. Which failure would most damage user trust?
2. Which failure would most mislead an agent?
3. Which boundary must stay aligned between `inspect`, `preview`, `convert`, and `draft`?

Priority: CLI contract tests → confirmation-vs-execution consistency → blocking readiness matrix → publish-path core → minimal real smoke. Use table-driven tests when behavior depends on input combinations. Do not add tests just to raise coverage.

### Version Consistency
All of these must stay aligned on release:
- `VERSION` file
- `package.json`
- `.claude-plugin/marketplace.json`
- `platforms/openclaw/md2wechat/SKILL.md` (install URLs)
- `CHANGELOG.md`

Quick check (equivalent to quality-gates step 0):
```bash
echo "VERSION:      $(cat VERSION)"
echo "package.json: $(node -e "process.stdout.write(require('./package.json').version)")"
echo "marketplace:  $(sed -n 's/.*"version": "\([0-9][^"]*\)".*/\1/p' .claude-plugin/marketplace.json | head -1)"
echo "CHANGELOG:    $(grep -m1 '^## \[' CHANGELOG.md | sed 's/^## \[\([^]]*\)\].*/\1/')"
```

`bash scripts/quality-gates.sh` runs this as step 0 (fail-fast, < 1 second) before lint/test.

### Documentation Sync
Any change to CLI commands, flags, JSON output shape, providers, themes, or prompts must also update:
- `README.md`
- `docs/DISCOVERY.md`
- `docs/FAQ.md`
- `skills/md2wechat/SKILL.md`
- `platforms/openclaw/md2wechat/SKILL.md`
- `docs/HUMANIZE.md` when humanizer behavior or intensity levels change

If the change affects config, install, or setup: also update `docs/CONFIG.md`, `docs/QUICKSTART.md`, `docs/USAGE.md`.

### Git and Release Rules
- Never `git push`, `git tag`, or `gh release create` without explicit user confirmation.
- Never rebase or amend history unless the user explicitly asks.
- GitHub Release is created automatically by `release.yml` when a tag matching `v*.*.*` is pushed — `gh` CLI is not required.
- After `npm publish`, trigger `npx cnpm sync @geekjourneyx/md2wechat` so npmmirror stays current.
