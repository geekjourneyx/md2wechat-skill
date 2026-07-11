# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Made the embedded advanced-layout catalog the single source of truth. The CLI no longer loads layout modules from the user config directory, project `layout/` directory, or `MD2WECHAT_LAYOUT_DIR`, and layout discovery no longer exposes override-specific fields or counts.

## [2.10.1] - 2026-07-08

### Changed
- Updated the Volcengine image provider default model to `doubao-seedream-5-0-pro-260628` and exposed the refreshed Seedream 5.0 model catalog in provider discovery.

## [2.10.0] - 2026-07-08

### Added
- Added `md2wechat advise <article.md> --json`, a deterministic, side-effect-free article enhancement router for optional title, cover, layout, and micro-edit recommendations.
- Added `article_advice` to `capabilities --json` so Agents can discover the `advise` command contract, supported recommendation tools, action limits, and no-side-effect boundary.
- Added stricter `inspect --json` readiness coverage for theme compatibility and custom AI prompt execution paths.

### Changed
- Documented `inspect --json data.readiness.targets/blockers` as the publish preflight truth source, while keeping `advise --json` scoped to optional enhancement decisions.
- Tightened Agent-facing repository rules to keep deterministic routing free of subjective hardcoded phrase lists and style-quality heuristics.

## [2.9.0] - 2026-06-26

### Added
- Added `md2wechat title suggest <article.md> --json`, a no-side-effect host-Agent handoff that prepares a WeChat title suggestion prompt request for external model execution.
- Added the bundled `title/wechat-title-expert` prompt catalog entry with strict JSON output guidance, title-count limits, scoring fields, and trust boundaries for WeChat title candidates.
- Added `title_generation` to `capabilities --json` so Agents can discover that title suggestions require `--json`, an external model, and no provider, image API key, WeChat credentials, upload, draft, or write-back side effects.

### Changed
- Derived `capabilities.commands` from the root command manifest while preserving the stable command order, reducing drift when new top-level commands are added.
- Updated user docs and both Agent skill entry points with minimal routing guidance for title suggestions, plus repository rules that treat `SKILL.md` edits as runtime-contract changes rather than routine documentation churn.

## [2.8.0] - 2026-06-24

### Added
- Added no-side-effect image plan mode for `generate_image`, `generate_cover`, and `generate_infographic` via `--plan --json`, allowing host Agents to use their own Image Gen tools without md2wechat image-provider credentials.
- Extended `capabilities --json` with image plan-mode discovery metadata so Agents can distinguish direct provider generation from host-Agent plan execution.

### Changed
- Documented the Agent Image Gen workflow across user docs and tightened Agent skill guidance while keeping direct image-provider generation unchanged.
- Refined the README around the professional API, Agent workflow, image plan mode, multi-account publishing, fixed WeChat egress, and commercial licensing paths.

## [2.7.0] - 2026-06-23

### Added
- Added WeChat fixed-egress proxy configuration through `wechat.proxy_url` / `WECHAT_PROXY_URL` for upload, draft creation, and image-post side effects.
- Added dedicated WeChat HTTP client wiring so configured proxy routing stays scoped to WeChat API calls instead of affecting API conversion, image generation providers, discovery commands, or local preview.
- Added API-key validation for WeChat proxy side effects, aligning fixed-egress proxy usage with other advanced API features.
- Added regression coverage for proxy config loading, environment override, validation, secret masking, API-key gating, and WeChat SDK client isolation.

### Changed
- Documented the fixed-egress proxy as an advanced API service capability that provides a stable WeChat whitelist IP and a complete proxy URL, without exposing deployment ports or private infrastructure details in public docs.

## [2.6.0] - 2026-06-16

### Added
- Added gated multi-account WeChat configuration with named accounts, `--wechat-account` selection, and the local-only `config wechat-accounts` discovery command.
- Added API-key validation for named-account WeChat side effects using `HEAD /api/auth/validate` before uploads, image generation uploads, draft creation, and image post creation.
- Added regression coverage for named-account selection, API-key gate failures, direct legacy account compatibility, and secret-free account discovery output.

### Changed
- Preserved legacy single-account `wechat.appid` / `wechat.secret` behavior without requiring `MD2WECHAT_API_KEY`.
- Documented multi-account configuration, selection order, API-key validation boundaries, and local-only account discovery across user and Agent entry points.

## [2.5.0] - 2026-06-11

### Added
- Added `md2wechat skills list/read` documentation across Agent, install, Obsidian, OpenClaw, and discovery entry points so Agents can read the SOP embedded in the current CLI binary.
- Added the missing `/theme-gallery` featured API themes to CLI discovery: `nyt-classic`, `github-readme`, `mint-fresh`, `sunset-amber`, `ink-minimal`, `lavender-dream`, `coffee-house`, and `bauhaus-primary`.

### Changed
- Tightened release-check documentation guards so embedded skill discovery, current-version SOP reads, and OpenClaw installer next steps cannot drift silently.
- Updated API theme documentation and command help from 40 to 48 professional themes.

## [2.4.0] - 2026-05-28

### Added
- Extended `inspect --json` with `data.readiness.schema_version`, `data.readiness.targets`, and `data.readiness.blockers` so Agents can decide whether `preview`, `convert`, `upload`, or `draft` is ready, blocked, degraded, or not requested.
- Added blocker-to-target mapping for publish checks such as `MISSING_API_KEY`, `LOCAL_IMAGE_MISSING`, and `MISSING_COVER`.
- Added CLI contract tests for the new `inspect --json` readiness target/blocker projection.

### Changed
- Kept existing readiness booleans (`convert_ready`, `upload_ready`, `draft_ready`, `preview_fidelity`) in place while documenting `targets/blockers` as the preferred Agent decision surface.
- Calibrated README, Agent guide, discovery docs, FAQ, quickstart, smoke docs, usage docs, and both skill entry points around the single `data.readiness` contract.
- Clarified that `doctor --json` readiness is local configuration attemptability, while `inspect --json` readiness is single-article target state.
- Updated inspect and preview command help from generic readiness wording to target-state wording.

### Migration Guide
No migration required. Existing `data.readiness.convert_ready`, `upload_ready`, `draft_ready`, and `preview_fidelity` fields remain available. New Agents should prefer `data.readiness.targets` and `data.readiness.blockers` for execution decisions.

## [2.3.1] - 2026-05-28

### Added
- Added README theme showcase screenshots for `default`, `bytedance`, `elegant-gold`, `elegant-green`, and `sspai-red` under `assets/theme-showcase/`.
- Added the `wechat-native` featured API theme with the native WeChat green texture style.

### Changed
- Expanded API theme collection entries from `api.yaml` into selectable runtime themes so collection-defined themes are discoverable through `themes list/show --json`.
- Renamed the classic curated theme group to 精选主题 / `featured_themes`.
- Calibrated public docs and command help around 40 professional themes and the current advanced layout theme showcase.

### Fixed
- Fixed theme discovery drift where themes listed inside `api.yaml`, such as `elegant-green`, were not available to `convert --theme`.

## [2.3.0] - 2026-05-27

### Added
- Added `md2wechat doctor --json`, a local-only readiness diagnostic for config loading, default convert mode, API key presence, default theme compatibility, layout catalog availability, and WeChat draft credential presence. `doctor` does not perform live authentication, upload images, or create drafts.
- Added `doctor` to `capabilities --json` command discovery.
- Added explicit `body_format` metadata to layout module specs and JSON discovery so Agents can choose `fields`, `rows`, `json_object`, or `json_array` without inferring syntax from examples.

### Changed
- Tightened theme discovery and compatibility metadata with `selectable`, `metadata_incomplete`, and `style` fields so Agents can choose themes from current CLI truth rather than guessing from stale docs.
- Made API/AI theme compatibility fail closed during conversion: API mode rejects AI themes, AI mode rejects API themes, and non-selectable theme collections cannot be used as concrete themes.
- Simplified `version --json` to the stable version contract only, removing non-essential build metadata from the CLI response.
- Updated README, discovery docs, FAQ, config docs, Agent guide, and both skill entry points to align with the discovery-first `doctor` → temporary formatted Markdown → `layout validate` → `convert` flow.
- Calibrated Agent-facing discovery guidance to use the smallest task-specific discovery set instead of a fixed full-catalog startup sequence.
- Replaced stale public block-syntax wording with concrete `:::module` / `:::hero` style examples.

### Fixed
- Fixed discovery drift where capabilities did not fully reflect the active layout, brand, and doctor commands.
- Fixed the Agent layout workflow boundary so automated formatting no longer needs to mutate the user's source Markdown.
- Fixed `layout validate` so malformed legacy-style openers fail closed instead of being ignored.
- Fixed JSON layout module required-field metadata so empty JSON objects no longer validate or render as usable modules.

### Technical Details
- **New package**: `internal/doctor`
- **New command**: `doctor`
- **New contract tests**: CLI JSON envelope tests for doctor readiness, theme compatibility, discovery truth, and minimal version output
- **Product acceptance**: validated `examples/article.md` through generated temporary Markdown, `layout validate`, API `convert`, exact `preview`, raw `:::` absence checks, and source hash preservation

### Migration Guide
No migration required. Existing conversion, preview, upload, draft, theme, and layout commands remain compatible. Agents should prefer `doctor --json` before API-dependent execution and keep the user's source Markdown read-only when generating formatted temporary artifacts.

## [2.2.1] - 2026-05-25

### Fixed
- Fixed OpenAI GPT Image generation for `generate_image`, `generate_cover`, and `generate_infographic` when GPT Image models return `b64_json` instead of a downloadable URL.
- Added fail-closed validation so empty provider image results stop at the image layer instead of surfacing as `download generated image: local file not found:`.

### Changed
- Updated the OpenAI default image model to `gpt-image-2` and the default OpenAI image size to `auto` according to current OpenAI Image API guidance.

## [2.2.0] - 2026-05-11

### Added
- **Brand Profile（品牌档案）**: 新增 `brand init` 和 `brand show` 命令，支持在 `~/.config/md2wechat/brand.md` 存储品牌档案
  - `brand init`：幂等初始化，创建带注释的 Markdown 模板文件；目录不存在时自动创建
  - `brand show`：以 JSON envelope 返回当前品牌档案原始文本（`data.content`）和文件路径（`data.path`）
  - 支持 `BRAND_INITIALIZED` / `BRAND_SHOWN` / `BRAND_NOT_FOUND` / `BRAND_READ_FAILED` 四个状态码
  - Brand Profile 由 Agent 读取原始文本，CLI 完全不解析此文件（工具职责分离）
  - **格式选择 Markdown 而非 YAML**：LLM 理解自然语言优于结构化字段；Markdown 可写反例、具体示例，信息密度更高
- **Markdown Expression Diagnosis（排版诊断层）**: `skills/md2wechat/SKILL.md` 新增三步诊断框架
  - Step 1 意图识别：目标读者、conversion goal、memorability anchor
  - Step 2 内容映射：四目标框架（attention / readability / memorability / conversion）
  - Step 3 模块选择：结合 Brand Profile 自然语言偏好和 CLI discovery 输出，从可验证模块中选择组合
- **Brand Profile 协议**: `SKILL.md` 新增完整 Agent 读取协议
  - 优先级链：CLI flag → 用户本轮明确指令 → Brand Profile 自由文本偏好 → config.yaml → CLI discovery → Agent 保守默认
  - 解释规则：Brand Profile 是自由 Markdown prompt，不按 YAML 或固定字段解析
  - 主题/模块偏好必须通过 `themes list --json` 和 `layout list/show` 验证；无效主题不静默降级
  - 3 问引导流程：用户主动设置 Brand Profile 时生成可读 Markdown，而不是结构化配置
- **`docs/AGENT-GUIDE.md`**: AI Agent 操作手册，12 大章节 685 行，覆盖
  - 能力发现（5 条 discovery 命令）
  - Brand Profile 操作（check / init / 读取 / 降级）
  - Markdown 排版诊断（三步法 + 四目标框架）
  - 文章转换、发布流程、高级排版模块操作
  - 异常处理与降级（4 种常见故障场景）
  - 决策优先级链、完整场景示例（5 个）、快速参考表

### Changed
- `skills/md2wechat/SKILL.md`：追加两大章节（Markdown Expression Diagnosis + Brand Profile 协议），从 199 行扩展至 ~400 行
- `platforms/openclaw/md2wechat/SKILL.md`：与 Claude Code 版本完全同步（同步新增章节，199 → ~400 行）
- `docs/CONFIG.md`：新增 Brand Profile 完整章节（Schema / 字段说明表 / 降级行为 / 配置对比表）

### Technical Details
- **New Files**: `cmd/md2wechat/brand.go`, `cmd/md2wechat/brand_test.go`, `docs/AGENT-GUIDE.md`, `docs/BRAND-PROFILE.md`
- **Modified Files**: `cmd/md2wechat/main.go`, `docs/CONFIG.md`, `skills/md2wechat/SKILL.md`, `platforms/openclaw/md2wechat/SKILL.md`, `docs/AGENT-GUIDE.md`
- **New code constants**: `BRAND_INITIALIZED`, `BRAND_INIT_FAILED`, `BRAND_SHOWN`, `BRAND_NOT_FOUND`, `BRAND_READ_FAILED`
- **Tests**: 8 contract tests in `brand_test.go`（init×4 + show×4），全部通过 `t.Setenv(HOME)` + `t.TempDir()` 隔离

### Migration Guide
无需迁移。Brand Profile 为可选配置，Agent 专用，不影响任何现有 CLI 工作流。

### Changed
- **License**: Upgraded from MIT to Source Available License (BUSL 1.1 base + custom Additional Use Grant)
  - Personal use, learning, evaluation, non-profit, and contributions remain free without authorization
  - Commercial use (SaaS, client delivery, rebranding, white-labeling, AI training data) requires a written commercial license
  - Change Date: 2030-01-01 — project automatically relicenses to Apache 2.0 on that date
  - Historical versions released under MIT are unaffected; this license applies to all new versions from this commit onward
  - Contact for commercial licensing: skrphper@gmail.com / https://jieni.ai / https://x.com/seekjourney

## [2.1.0] - 2026-04-27

### Added
- **`md2wechat layout` command tree**: 4 new CLI subcommands for AI-agent-friendly discovery of 43 advanced WeChat layout modules (`:::module` syntax)
  - `layout list [--category] [--serves] [--content-type] [--industry] [--tag] --json` — filtered module discovery
  - `layout show <name> --json` — full spec with fields, when_to_use, example, metadata
  - `layout render <name> --var KEY=VALUE --json` — generate `:::module` syntax from structured inputs
  - `layout validate [--file | --stdin] --json` — validate `:::module` usage, unknown modules warn not error
- **43 built-in layout modules** across 7 categories (opening, judgment, infographic, evidence, brand, conversion, sprint4)
- **4-level module override**: builtin → ~/.config/md2wechat/layout/ → ./layout/ → $MD2WECHAT_LAYOUT_DIR
- **`internal/layoutcatalog` package**: schema, loader, renderer, validator with 20+ unit tests
- **Integration test fixtures**: 3 article shapes (opinion-piece, data-report, mixed-with-unknown)
- **E2E test suite**: gated by MD2WECHAT_E2E=1, validates each module's example against /api/convert
- **`docs/LAYOUT.md`**: 保姆级高级排版模块教程，覆盖全部 43 个模块的字段说明、使用场景、示例和常见错误
- **`--intensity authentic` for `humanize`**: new standalone intensity level that bypasses the 24-pattern AI-trace detection pipeline and instead applies six-dimension writing-quality rules (用词 / 句式 / 语气 / 内容表达 / 结构 / 整体原则). The goal is writing that reads like a skilled human, not merely text with AI markers removed. Aliases: `authentic` / `natural` / `真实` / `自然`.
- **`docs/HUMANIZE.md`**: 保姆级 humanize 命令教程，覆盖四种强度选择逻辑、`--show-changes` 输出格式、Agent/JSON 用法、管道输入、与 `write` 集成，以及 `authentic` vs `aggressive` 的本质区别

### Fixed
- **9 YAML module field doc errors** (`sprint4` category): corrected JSON key names in YAML field documentation so Agents no longer generate wrong keys at render time
  - `quote-card`: `quote`/`content` → `text`
  - `tweet`: `content`→`text`, `author`→`name`, `date`→`timestamp`
  - `definition`: added `term`/`def`/`termLabel` fields, anti_pattern clarifies `def` not `definition`
  - `stat-row`, `question`, `resource-list`, `comparison-table`, `changelog`: replaced misleading pipe-format rows with correct JSON object field docs

### Changed
- **README.md**: restructured from 1941 → 388 lines as a commercial landing page. Removed 3 Mermaid diagrams, 5 repeated install sections, 200-line AI quality scoring system, and ModelScope flowchart. New structure: Hero → API unlock banner → comparison table → quickstart → command table → 高级排版模块 → Agent discovery → AI vs API table → Coding Agent platforms → FAQ → community → Star History.
- **SKILL.md (both Claude Code and OpenClaw)**: added API-mode-only blockquote for layout modules; removed `https://www.md2wechat.cn` URL from public skill files (security); added gentle AI→API nudge rule; added `authentic` intensity example and four-level intensity reference table for `humanize`
- **`docs/DISCOVERY.md`**: added `prompts show authentic --kind humanizer` example
- **AGENTS.md / .github/copilot-instructions.md**: added layout API-only constraint notes (internal dev docs, URL retained)
- **Docs version references**: bumped `v2.0.7` → `v2.1.0` across `QUICKSTART.md`, `USAGE.md`, `FAQ.md`, `INSTALL.md`

### Technical Details
- **New Package**: `internal/layoutcatalog` (schema.go, loader.go, renderer.go, validator.go)
- **New Files**: `internal/assets/builtin/layout/**/*.yaml` (43 module definitions), `docs/LAYOUT.md`, `docs/HUMANIZE.md`
- **New Prompt**: `internal/assets/builtin/prompts/humanizer/authentic.yaml` — standalone template rendered with only `{{CONTENT}}`, bypassing `base.yaml`
- **New Commands**: `cmd/md2wechat/layout.go`, `cmd/md2wechat/layout_test.go`, `cmd/md2wechat/layout_e2e_test.go`
- **New Test Data**: `internal/layoutcatalog/testdata/integration/`
- **Error Codes**: LAYOUT_MODULE_NOT_FOUND, LAYOUT_INVALID_FILTER, LAYOUT_MISSING_REQUIRED_FIELD, LAYOUT_INVALID_FIELD_VALUE, LAYOUT_VALIDATE_HAS_ERRORS, LAYOUT_VALIDATED

### Migration Guide
No migration required. Layout commands are purely additive. Existing workflows and config files are unchanged.

## [2.0.7] - 2026-03-30

### Added
- Added `volcengine` / `volc` as a built-in image provider for Volcengine Ark, with support for `doubao-seedream-5-0-260128` and `doubao-seedream-5-0-lite-260128`.
- Added provider-level `supported_models` discovery to `providers list --json`, `providers show --json`, and `capabilities --json`, so agents and scripts can inspect image model catalogs before choosing `--model`.

### Changed
- Switched `config init` and the bundled example config to a `volcengine` image setup by default, including the Ark base URL, default Seedream model, and `2K` size tier.
- Made image defaults provider-aware, so Volcengine now falls back to `2K` when `api.image_size` is omitted instead of inheriting an OpenAI-shaped square size.
- Consolidated image-provider model metadata into the shared provider registry and re-audited high-signal docs and both skill entry points to keep provider, model, and discovery guidance aligned.
- Removed the `skills/md2wechat/references/` directory and folded the remaining guidance back into the main `SKILL.md` entry point.

### Fixed
- Fixed Volcengine `ModelNotOpen` guidance to point users to the Doubao model activation path instead of a vague Ark-console hint.
- Fixed JSON-mode CLI tests to stream captured stdout/stderr safely, preventing large discovery payloads from deadlocking the test process.

## [2.0.6] - 2026-03-28

### Added
- Added an npm distribution package at `@geekjourneyx/md2wechat` so users can install the CLI through `npm install -g` without compiling Go locally.
- Added `scripts/install.js` and `scripts/run.js` as a thin npm wrapper layer that downloads the version-matched GitHub Release binary and verifies `checksums.txt`.
- Added `--cover-media-id` for `convert --draft` and `inspect --draft`, so article drafts can reuse an existing WeChat permanent cover asset without re-uploading a local file.

### Changed
- Updated the release workflow so npm packaging is smoked against the same release artifact on Linux and Windows before publish, then published to npm only after the GitHub Release assets are live.
- Extended local and CI quality gates to validate npm package contents and keep `package.json`, `VERSION`, and marketplace metadata aligned.
- Updated install documentation to include the npm path while keeping Homebrew, fixed-version installers, and Go install as first-class alternatives.
- Tightened both skill entry points so `小绿书` / `图文笔记` / `newspic` requests route to `create_image_post` instead of being treated as standard article drafts.
- Narrowed the npm support contract to the actual published release matrix: macOS `amd64/arm64`, Linux `amd64/arm64`, and Windows `amd64`.

### Fixed
- Fixed Windows npm smoke and local-bundle installs by treating Windows drive-letter paths in `MD2WECHAT_RELEASE_BASE_URL` as local filesystem paths instead of unsupported URL schemes.
- Fixed agent-facing draft guidance so README, usage docs, and both `SKILL.md` entry points consistently describe `--cover` and `--cover-media-id` as the valid draft-cover contract.
- Closed documentation drift around image-provider setup by removing stale `GOOGLE_API_KEY` guidance and keeping Gemini/Google instructions aligned on `IMAGE_API_KEY`.

## [2.0.5] - 2026-03-26

### Added
- Added `inspect` as a release-grade confirmation command that resolves final article metadata, readiness, and publish risks before conversion or draft creation.
- Added `preview` as a lightweight standalone HTML preview flow that writes a local preview artifact without upload or draft side effects.
- Added a typed `internal/inspect` confirmation layer so metadata sources, readiness, and checks are computed once and reused by CLI and preview flows.

### Changed
- Updated `capabilities --json` to advertise `inspect` and `preview` as first-class commands.
- Kept the preview implementation Go-only with embedded template rendering instead of introducing a separate frontend stack or editable workbench.
- Clarified that API mode can produce exact preview HTML when conversion succeeds, while AI mode degrades honestly to a confirmation page instead of pretending to show final layout.
- Reframed installer next steps and the main README around a confirm-first flow: `inspect` -> `preview` -> `convert` / `--draft`.

### Fixed
- Closed documentation drift around release version anchors by updating install and high-signal entry-point docs to `2.0.5`.
- Closed agent drift by updating both skill entry points to describe the new confirm-first flow and AI preview degradation behavior consistently.
- Tightened `scripts/release-check.sh` so version anchors, marketplace metadata, confirm-first docs, and both skill entry points are verified together before release work is considered done.
- Fixed JSON-mode CLI behavior so `inspect --json`, `preview --json`, and related machine-readable commands no longer mix config banners into stdout.
- Fixed `convert --preview` to emit pure HTML on stdout instead of wrapped terminal markers, making redirect-to-file and agent use more reliable.
- Fixed WeChat draft error guidance so `45004` now points users and agents to digest/description limits instead of encouraging body-content guesswork.
- Re-audited README, USAGE, DISCOVERY, FAQ, and both skill entry points to keep metadata-vs-body semantics, preview behavior, and image replacement rules aligned.

## [2.0.4] - 2026-03-25

### Added
- **CLI flags for article metadata**: Added `--title`, `--author`, and `--digest` flags to `convert` command for explicit metadata override
- **Metadata validation**: Added length validation for title (32 chars), author (16 chars), and digest (128 chars) with clear error messages
- **Frontmatter stripping**: `convert` now properly strips frontmatter from markdown body before passing to converter, preventing metadata from appearing in generated HTML
- **ArticleDocument structure**: Introduced `ArticleDocument` to separate metadata extraction from body parsing, improving testability

### Changed
- **background_type default**: Changed default `background_type` from `default` to `none` for cleaner output
- **Metadata priority**: Clarified metadata resolution order: CLI flags → frontmatter → first markdown heading → fallback
- **Test coverage**: Added comprehensive tests for metadata override, frontmatter stripping, and validation

### Fixed
- Fixed metadata extraction to properly handle `digest`, `summary`, and `description` frontmatter fields with correct priority
- Fixed first-line title fallback logic to avoid treating non-heading body text as title
- Fixed outdated documentation about background_type default value

### Technical Details
- **New Files**: None (structure changes only)
- **Modified Files**:
  - `cmd/md2wechat/convert.go` - Added CLI flags, validation, and document parsing
  - `cmd/md2wechat/convert_test.go` - Added 4 new test cases
  - `internal/converter/metadata.go` - Added `ArticleDocument` and `ParseArticleDocument`
  - `internal/converter/converter.go` - Added `normalizeRequest` to handle metadata merging
  - `internal/converter/ai.go` - Updated to use stripped body and resolved metadata
  - `internal/config/config.go` - Changed default background_type to "none"
  - Documentation updates across README, docs/, and SKILL.md files

### Migration Guide
No breaking changes. The new CLI flags are optional. If you relied on the previous `background_type: "default"` behavior, explicitly pass `--background-type default` or set it in your config file.

## [2.0.3] - 2026-03-21

### Added
- Added Homebrew-friendly multi-platform release archives for `darwin/linux` and `amd64/arm64`, so the project now publishes tarballs that formula-based installs can consume directly.
- Added automatic tap update support in the release workflow, including generated `Formula/md2wechat.rb` output and a release-time push path to `geekjourneyx/homebrew-tap`.

### Changed
- Upgraded the OpenClaw skill metadata to `metadata.clawdbot` with explicit `brew` and `go install` entries, and rewrote the skill body around discovery-first usage instead of installer instructions.
- Promoted `brew install geekjourneyx/tap/md2wechat` as a first-class installation path in user-facing docs while keeping the fixed-version release installer path for checksum-verified installs.
- Standardized version injection around `main.Version` so local builds, release builds, formula tests, and `version --json` verification all resolve the same symbol.

### Fixed
- Fixed Homebrew tap CI so it installs `geekjourneyx/tap/md2wechat` through the tap workflow managed by `setup-homebrew`, avoiding local formula rejection, remote mismatch handling, and post-job cleanup failures.
- Re-audited README, INSTALL, QUICKSTART, FAQ, OPENCLAW, plugin marketplace metadata, and both skill entry points to keep release references aligned at `2.0.3`.

## [2.0.2] - 2026-03-21

### Added
- Added a dedicated `install-openclaw.ps1` so Windows OpenClaw installation now has a fixed-version installer that provisions both the skill shell and the CLI instead of relying on the generic CLI-only PowerShell installer.
- Added more agent-friendly installer output for shell and PowerShell flows, including immediate next-step verification commands, PATH repair hints, and explicit `config init` / `capabilities` guidance.

### Changed
- Reworked both Coding Agent and OpenClaw paths so the skill layer is now knowledge-and-workflow only; both paths assume an already-installed `md2wechat` CLI on a formal user-level install path rather than wrapper-managed runtime bootstrapping.
- Standardized the recommended installation path around fixed-version installers and direct copy-paste commands for agents, without requiring `MD2WECHAT_RELEASE_BASE_URL` exports or recommending `go install` as an end-user path.
- Simplified OpenClaw installation so the shell installer provisions the skill shell to `~/.openclaw/skills/md2wechat/` and installs the CLI to the user-level executable path instead of an OpenClaw-private runtime directory.
- Refreshed README, FAQ, INSTALL, QUICKSTART, and OPENCLAW onboarding so “send this to your agent” scripts consistently export PATH before validating `md2wechat`, reducing first-run failures in fresh shells.
- Updated release automation and release checks to match the new OpenClaw contract: no OpenClaw wrapper script, new `install-openclaw.ps1`, and smoke verification against an installed CLI on the environment path.

### Removed
- Removed `skills/md2wechat/scripts/run.sh`; the coding-agent skill no longer acts as a runtime provisioner or wrapper and now documents direct `md2wechat` usage only.
- Removed `platforms/openclaw/md2wechat/scripts/run.sh`; the OpenClaw skill no longer ships a runtime wrapper and instead assumes the CLI has been installed by the fixed-version installer.
- Removed `platforms/openclaw/md2wechat/references/runtime.md`; OpenClaw runtime expectations were folded back into `platforms/openclaw/md2wechat/SKILL.md`.

### Fixed
- Removed coding-agent runtime auto-download during normal execution; the Coding Agent path now rejects missing or mismatched CLIs instead of fetching a remote binary from GitHub Releases at execution time.
- Fixed OpenClaw packaging drift on Windows by pointing skill install metadata to the dedicated OpenClaw PowerShell installer instead of the generic CLI installer.
- Fixed high-signal onboarding drift where README / FAQ / OPENCLAW scripts could validate `md2wechat` before repairing `PATH`, which caused avoidable failures in fresh shells.
- Tightened installer output and OpenClaw guidance so CLI paths, expected skill directory layout, and fixed-version installation flow are described consistently across docs and release metadata.
- Re-audited README, FAQ, INSTALL, QUICKSTART, OPENCLAW, both `SKILL.md` files, installer comments, and release metadata to keep version anchors aligned at `2.0.2`.

## [2.0.1] - 2026-03-20

### Fixed
- Fixed the release workflow so PowerShell installer smoke now runs on `windows-latest` instead of Ubuntu, and PowerShell version smoke parses CLI JSON output correctly.
- Fixed the release workflow binary smoke so release version injection is validated against the actual bundled artifact path.
- Tightened OpenClaw installation guidance to reflect current ClawHub behavior: `clawhub install md2wechat` installs the skill shell only and does not guarantee runtime provisioning.
- Added OpenClaw runtime version checks so the OpenClaw wrapper rejects mismatched runtimes instead of silently executing them.
- Added coding-agent runtime version checks so `skills/md2wechat/scripts/run.sh` only accepts PATH-discovered runtimes when the version matches the current skill version.
- Improved coding-agent runtime fallback behavior so GitHub Releases download failures no longer produce misleading missing-temp-file errors.

## [2.0.0] - 2026-03-20

### Added
- Added a typed publish pipeline with shared article, asset, and action models across `convert`, image generation, draft creation, and image-post flows.
- Added built-in prompt catalog and discovery commands: `capabilities`, `providers`, `themes`, and `prompts`.
- Added built-in image prompt presets with explicit use-case metadata, including cover, infographic, banner, ticket, sketchnote, Apple-keynote, and Victorian engraving variants.
- Added higher-level image generation commands `generate_cover` and `generate_infographic`, plus `generate_image --preset` support.
- Added per-command image model override via `--model` for image generation commands.
- Added dedicated onboarding and operator docs, including configuration, discovery, WeChat credential/IP whitelist, smoke verification, and image provider guides.

### Changed
- Reworked the CLI around a stable machine-readable contract with consistent `success/code/message/schema_version/status/data/error` JSON envelopes.
- Split skill packaging into a coding-agent path (`skills/md2wechat/`) and an OpenClaw path (`platforms/openclaw/md2wechat/`), with platform-specific installation and runtime guidance.
- Promoted built-in embedded assets for default themes, writer styles, and prompt resources so the CLI no longer depends on repo-relative files to function.
- Hardened release engineering with a single `VERSION` source, `release-check`, CI/release gates, checksum-verified assets, and installer smoke tests.
- Standardized discovery-first documentation so README, docs, and both skills point users and agents to CLI capabilities instead of hand-maintained lists.
- Updated default image provider models to match current provider guidance, including `gpt-image-1.5` for OpenAI and newer Gemini defaults.

### Fixed
- Fixed frontmatter title fallback so Markdown titles are resolved correctly when frontmatter omits `title`, including CRLF input handling.
- Fixed remote download and image handling boundaries, including stronger SSRF protection and consistent remote/local image resolution.
- Fixed `convert --json --output` so requested HTML artifacts are written before returning JSON success payloads.
- Fixed `create_image_post --from-markdown` so resolved local and remote image paths work correctly outside the article directory.
- Fixed version-pinned installer behavior so release installers no longer silently drift to newer assets.
- Fixed Gemini image sizing so aspect ratio and image size are forwarded to the official API config, and documented provider-specific size behavior such as ModelScope's `WIDTHxHEIGHT` requirement.

## [1.11.1] - 2026-03-19

### Changed
- Tightened the Markdown-to-WeChat pipeline around shared metadata parsing, image placeholder回填, and remote image download boundaries.
- Expanded command-level and module-level tests across `cmd/md2wechat`, `internal/converter`, `internal/draft`, `internal/humanizer`, `internal/wechat`, and `internal/writer`.
- Promoted checksum-verified, version-pinned release assets and installer scripts as the primary install path.
- Moved the OpenClaw installer to the same version-pinned release + checksum flow as the main CLI installer.
- Added build-once, smoke-same-artifact, release-same-artifact flow to the release workflow.
- Hardened installation and execution docs so fixed-version release assets are the primary install path.
- Added repository execution guidance in `docs/AGENTS.md` for development, validation, and release discipline.

## [1.11.0] - 2026-03-12

### Fixed
- **IsAIRequest 函数修复**: 修复了 `IsAIRequest` 始终返回 `false` 的 bug
  - 原代码使用魔法数字 `14` 进行字符串切片比较，但前缀 `"AI_MODE_REQUEST:"` 实际长度为 16，导致比较永远不匹配
  - `ExtractAIRequest` 同样因切片偏移错误 (`[14:]`) 会在结果中保留 `"T:"` 残留前缀
  - 改用 `strings.HasPrefix` / `strings.TrimPrefix`，与 `generator.go` 中的现有模式保持一致

### Changed
- **aiModePrefix 常量**: 引入包级常量 `aiModePrefix = "AI_MODE_REQUEST:"` 消除魔法数字，所有引用统一使用该常量

### Technical Details
- **Modified Files**: `internal/converter/ai.go`

### Migration Guide
No migration required.

## [1.10.0] - 2025-02-10

### Added
- **Image Post (小绿书/Newspic)**: Create WeChat image-only posts with up to 20 images
  - New `create_image_post` command for 小绿书 creation
  - Support comma-separated image paths: `--images photo1.jpg,photo2.jpg`
  - Extract images from Markdown files: `--from-markdown article.md`
  - Comment settings: `--open-comment`, `--fans-only`
  - Preview mode: `--dry-run` for testing without upload
  - Stdin support for description content
- **WeChat API Documentation**: Updated `references/wechat-api.md` with complete draft/add API documentation
  - Added `article_type` field (`news`/`newspic`)
  - Added comment settings fields
  - Added image_info structure for newspic
  - Added cover cropping fields
- **v2 API Themes (38 total)**: Massive theme expansion with new series system
  - **Minimal Series (8)**: minimal-gold, minimal-green, minimal-blue, minimal-orange, minimal-red, minimal-navy, minimal-gray, minimal-sky
  - **Focus Series (8)**: focus-gold, focus-green, focus-blue, focus-orange, focus-red, focus-navy, focus-gray, focus-sky
  - **Elegant Series (8)**: elegant-gold, elegant-green, elegant-blue, elegant-orange, elegant-red, elegant-navy, elegant-gray, elegant-sky
  - **Bold Series (8)**: bold-gold, bold-green, bold-blue, bold-orange, bold-red, bold-navy, bold-gray, bold-sky
  - Theme version tracking: v1.0 (6 basic) vs v2.0 (32 new)
  - Theme preview gallery: https://md2wechat.app/theme-gallery
- **Background Type Control**: New `--background-type` parameter for API mode
  - Options: `default` (default background), `grid` (grid texture), `none` (transparent)
  - Configuration file support: `api.background_type`
  - Environment variable: `DEFAULT_BACKGROUND_TYPE`
- **Google Gemini Image Provider**: Native Google SDK integration
  - Official `google.golang.org/genai` SDK
  - Direct API calls without third-party wrappers
  - Supports official Gemini image models: gemini-3-pro-image-preview, gemini-2.0-flash-exp-image-generation
  - Full aspect ratio support with official size mappings
- **OpenRouter Image Provider**: Multi-model provider access
  - Support for various AI models through OpenRouter API
  - OpenRouter format with image_size and aspect_ratio parameters
  - Configuration: `image_provider: openrouter`

### Changed
- **Article Structure**: Extended with newspic support fields (backward compatible with `omitempty`)
  - `article_type`: news (default) or newspic
  - `need_open_comment`: comment settings
  - `only_fans_can_comment`: fan-only comment settings
  - `image_info`: image list for newspic
- **API Base URL Configuration**: New `md2wechat_base_url` parameter
  - Support for v2 API endpoint: https://md2wechat.app (internal testing)
  - Configuration file: `api.md2wechat_base_url`
  - Environment variable: `MD2WECHAT_BASE_URL`
- **Theme File Structure**: Reorganized with version tracking
  - v1.0 themes: 6 basic themes (default, bytedance, apple, sports, chinese, cyber)
  - v2.0 themes: 32 new series themes (minimal/focus/elegant/bold series)
  - New `themes/api.yaml`: Complete theme catalog for reference

### Technical Details
- **New Files**:
  - `cmd/md2wechat/create_image_post.go` - CLI command implementation
  - `internal/image/gemini.go` - Google Gemini provider with official SDK
  - `internal/image/gemini_test.go` - Gemini provider tests
  - `internal/image/openrouter.go` - OpenRouter provider implementation
  - `internal/image/openrouter_test.go` - OpenRouter provider tests
  - `themes/api.yaml` - Complete v2 theme catalog
  - `themes/elegant-gold.yaml`, `themes/focus-green.yaml`, `themes/minimal-blue.yaml`, `themes/bold-red.yaml` - Series examples
- **Modified Files**:
  - `internal/draft/service.go` - Added `CreateImagePost`, `GetImagePostPreview`, `extractImagesFromMarkdown`
  - `internal/wechat/service.go` - Added `CreateNewspicDraft` for direct API call
  - `cmd/md2wechat/main.go` - Registered `createImagePostCmd`
  - `cmd/md2wechat/convert.go` - Added background_type flag, updated help text
  - `internal/config/config.go` - Added DefaultBackgroundType, MD2WechatBaseURL fields
  - `internal/converter/api.go` - Added BackgroundType to APIRequest, added base URL support
  - `internal/converter/converter.go` - Added BackgroundType to ConvertRequest
  - `internal/image/provider.go` - Added gemini, openrouter providers
  - `skills/md2wechat/references/themes.md` - Updated with v2 themes and background type
  - `themes/*.yaml` - Reorganized with version markers (v1.0 vs v2.0)
  - `docs/CONFIG.md` - Added md2wechat_base_url, background_type documentation
  - `docs/IMAGE_PROVISIONERS.md` - Added Gemini and OpenRouter provider guides
  - `README.md` - Updated with v2 API themes, background type, upgraded API announcement

### Migration Guide
No migration required. All new features are additive:
- `create_image_post` is a new command
- New v2.0 themes are opt-in via `--theme` parameter
- `background_type` changed to default to "none" (was "default" in v2.0.3)
- Image providers are configurable, no breaking changes

### Technical Details
- **New Files**:
  - `cmd/md2wechat/create_image_post.go` - CLI command implementation
- **Modified Files**:
  - `internal/draft/service.go` - Added `CreateImagePost`, `GetImagePostPreview`, `extractImagesFromMarkdown`
  - `internal/wechat/service.go` - Added `CreateNewspicDraft` for direct API call (SDK doesn't support newspic)
  - `cmd/md2wechat/main.go` - Registered `createImagePostCmd`
  - `skills/md2wechat/SKILL.md` - Added image post documentation
  - `skills/md2wechat/references/wechat-api.md` - Complete draft API documentation

### Migration Guide
No migration required. The `create_image_post` command is a new feature and doesn't affect existing functionality.

---

## [1.8.0] - 2025-02-05

### Added
- **OpenClaw Platform Support**: Full compatibility with OpenClaw AI Agent platform
  - ClawHub installation: `clawhub install md2wechat`
  - One-click install script: `scripts/install-openclaw.sh`
  - OpenClaw configuration guide: `docs/OPENCLAW.md`
  - SKILL.md metadata for OpenClaw compatibility
- **OpenClaw Badge**: Added to README.md header

### Changed
- **Directory Structure Simplification**:
  - Renamed `skill/` to `skills/` (unified naming)
  - Removed redundant `plugins/` directory
  - Removed `plugin.json` (keeping only `marketplace.json`)
  - Removed `manifest.json` (version hardcoded in `run.sh`)
- **Binary Provisioner Refactoring** (`run.sh`):
  - Reduced from 375 lines to 154 lines (-59%)
  - Changed cache directory from `~/.cache/claude/` to `~/.cache/md2wechat/`
  - Simplified version management (plain text file instead of JSON)
  - Removed invalid jsDelivr mirror (binaries are in GitHub Releases, not in repo)
  - Improved error messages with cleaner formatting
- **Documentation Updates**:
  - Updated README.md with OpenClaw support section
  - Fixed project structure diagram
  - Added OpenClaw vs Claude Code comparison table

### Removed
- `plugins/` directory (redundant copy of skills)
- `.claude-plugin/plugin.json` (replaced by marketplace.json)
- `skill/md2wechat/manifest.json` (version now in run.sh)
- `scripts/sync.sh` (no longer needed without plugins/)

### Technical Details
- **New Files**:
  - `docs/OPENCLAW.md` - OpenClaw installation guide
  - `scripts/install-openclaw.sh` - OpenClaw installer script
- **Modified Files**:
  - `.claude-plugin/marketplace.json` - source changed to `"."`
  - `skills/md2wechat/SKILL.md` - added `metadata.openclaw` for compatibility
  - `skills/md2wechat/scripts/run.sh` - complete refactoring

### Migration Guide
No migration required for existing Claude Code users. The skill continues to work the same way.

For OpenClaw users:
1. Install via ClawHub: `clawhub install md2wechat`
2. Configure in `~/.openclaw/openclaw.json`
3. See `docs/OPENCLAW.md` for detailed instructions

---

## [1.7.0] - 2025-01-25

### Added
- **ModelScope Image Provider**: Native integration with Alibaba ModelScope image generation service
  - Async API support with task-based polling mechanism (task_id + status check)
  - Configurable polling interval (default 5s) and max polling time (default 120s)
  - Model: `Tongyi-MAI/Z-Image-Turbo` with support for custom image sizes
  - Free tier available for testing
- **Write Command Stdin Support**: Pipe input support for non-interactive usage
  - `echo "content" | md2wechat write --style dan-koe`
  - Heredoc support for multi-line content input
- **Image Size Parameter**: ModelScope provider supports separate `width` and `height` parameters
  - Default size: 1024x1024
  - Configurable via `IMAGE_SIZE` or `image_size` in config
- **Documentation Updates**:
  - ModelScope configuration guide in README.md
  - New FAQ entries for content size limit, ModelScope setup, and stdin usage
  - Updated `docs/IMAGE_PROVISIONERS.md` with ModelScope documentation

### Changed
- `internal/image/provider.go`: Added ModelScope (`modelscope`, `ms`) provider to factory
- `internal/image/provider.go`: Updated error hints to include ModelScope in supported providers list
- `cmd/md2wechat/write.go`: Added `readStdin()` function for pipe/redirection detection
- `cmd/md2wechat/write.go`: Added `io` import for stdin reading functionality
- `internal/wechat/service.go`: Fixed `maskMediaID()` to handle empty strings safely
- `skills/md2wechat/SKILL.md`: Added stdin/heredoc usage examples
- `skills/md2wechat/references/writing-guide.md`: Added non-interactive input methods section

### Technical Details
- **New Files**:
  - `internal/image/modelscope.go` - ModelScope provider implementation (372 lines)
    - `ModelScopeProvider` struct with async polling support
    - `parseSize()` function for WIDTHxHEIGHT format parsing
    - `createTask()` for async job creation
    - `pollTaskStatus()` for status polling with timeout
    - `getTaskStatus()` for task status retrieval
    - `handleErrorResponse()` for comprehensive error handling
  - `internal/image/modelscope_test.go` - Unit tests (9 test cases, all passing)

### Configuration
```yaml
# ModelScope Configuration Example
api:
  image_provider: modelscope
  image_key: ms-your-token-here
  image_base_url: https://api-inference.modelscope.cn
  image_model: Tongyi-MAI/Z-Image-Turbo
  image_size: 1024x1024
```

### Migration Guide
No migration required. ModelScope is a new optional image provider. To use it:

1. Get API Key from [modelscope.cn](https://modelscope.cn/my/myaccesstoken)
2. Set `IMAGE_PROVIDER=modelscope`
3. Set `IMAGE_API_KEY=ms-your-token-here`
4. Run: `md2wechat generate_image "A golden cat"`

---

## [1.6.0] - 2025-01-19

### Added
- **AI Writing Trace Removal (Humanizer)**: Remove AI-generated text patterns to make content sound more natural
  - New `humanize` command for standalone AI trace removal
  - Support for 24 AI writing pattern types based on humanizer-zh method
  - Three intensity levels: `gentle`, `medium` (default), `aggressive`
  - Quality scoring system with 5 dimensions (directness, rhythm, trust, authenticity, conciseness)
  - Style priority principle: preserves writing style characteristics when combined with creator styles
- **Write + Humanize Integration**: Combine writing style generation with AI trace removal
  - New `--humanize` flag for `write` command
  - New `--humanize-intensity` flag to control processing intensity
  - `--show-changes` flag to view modification details and quality scores
  - `-o, --output` flag for `humanize` command to save results to file
- **Pattern Detection Categories**:
  - Content patterns: overemphasis, promotional language, vague attribution
  - Language patterns: AI vocabulary, negative parallelism, three-part formula
  - Style patterns: dash overuse, bold abuse, emoji decoration
  - Filler avoidance: padding phrases, over-qualification, generic conclusions
  - Collaboration traces: conversational filler, knowledge cutoff disclaimers
- **Documentation**:
  - New `references/humanizer.md` with complete AI trace removal guide
  - Updated SKILL.md with humanizer natural language examples
  - Updated manifest.json version to 1.6.0

### Changed
- `cmd/md2wechat/main.go`: Added `humanizeCmd` to root command registration
- `cmd/md2wechat/humanize.go`: Implemented complete humanize command with flags
- `cmd/md2wechat/write.go`: Added humanizer integration flags and output handling

### Technical Details
- **New Files**:
  - `internal/humanizer/` - AI writing trace removal module
    - `result.go` - Core data structures (HumanizeRequest, HumanizeResult, Score, Change)
    - `prompt.go` - Humanizer-zh prompt builder with 24 pattern types
    - `humanizer.go` - Core processing logic and response parsing
  - `cmd/md2wechat/humanize.go` - Humanize command implementation
  - `skills/md2wechat/references/humanizer.md` - Humanizer documentation

### Breaking Changes
- None

### Migration Guide
No migration required. The humanize command is a new feature and doesn't affect existing functionality.

---

## [1.5.0] - 2025-01-17

### Added
- **Writer Style Assistant**: AI-powered writing assistance with customizable creator styles
  - New `write` command for assisted article generation
  - Support for multiple input types: idea, fragment, outline, title
  - Automatic cover prompt generation matching writing style
- **Dan Koe Style**: First built-in creator writing style
  - Profound, sharp, grounded tone for personal growth and opinion pieces
  - Complete writing framework with hooks, structures, and quote extraction
  - Victorian Woodcut/Etching style cover generation
- **Custom Style System**: YAML-based style definitions in `writers/` directory
  - Easy to add custom creator styles
  - Configurable writing prompts and cover styles
- **Image Size Control**: New `--size` parameter for `generate_image` command
  - Support for 16:9 ratio (2560x1440) for WeChat cover images
  - Multiple preset sizes: 2048x2048, 1920x1920, 2560x1440, 1440x2560, etc.
- **Documentation**:
  - New `writers/README.md` with custom style guide
  - New `docs/WRITING_FAQ.md` for writing beginners
  - New `references/writing-guide.md` with complete write command reference
  - Updated `references/image-syntax.md` with size parameter documentation
  - Enhanced README.md with write command workflows and diagrams
  - Enhanced SKILL.md with writing assistance natural language examples

### Changed
- README: Added API service notice at top with md2wechat.cn contact information
- README: Updated author section with donation information and QR codes
- README: Added writer style comparison table and workflow diagrams
- SKILL.md: Improved LLM instruction following with explicit trigger conditions
- Sync script: Added `writing-guide.md` to file synchronization list

### Technical Details
- **New Files**:
  - `internal/writer/` - Writer style assistant module
    - `types.go` - Core data structures
    - `style.go` - Style management system
    - `generator.go` - Article generation logic
    - `assistant.go` - High-level assistant API
    - `cover_generator.go` - Cover prompt generation
  - `cmd/md2wechat/write.go` - Write command implementation
  - `writers/dan-koe.yaml` - Dan Koe style configuration
  - `writers/README.md` - Custom style guide
  - `docs/WRITING_FAQ.md` - Writing functionality FAQ
  - `skills/md2wechat/references/writing-guide.md` - Writing command reference

### Breaking Changes
- None

### Migration Guide
No migration required. The write command is a new feature and doesn't affect existing functionality.

---

## [1.4.0] - 2025-01-14

### Added
- **Provider Pattern**: Extensible image generation service architecture
- **TuZi Integration**: Support for TuZi (tu-zi.com) image generation service
  - Models: `doubao-seedream-4-5-251128` (default), `gemini-3-pro-image-preview`
  - Sizes: 2048x2048 (default), 1920x1920, 2560x1440, 1440x2560, 3072x2048, 2048x3072, 3840x2160, 2160x3840
- **Natural Language Image Generation**: Generate images via conversational interface
  - Method 1: Insert into article at specific position
  - Method 2: Standalone image generation
  - Method 3: Manual Markdown syntax `![alt](__generate:prompt__)`
- **Configuration Fields**: `image_provider`, `image_api_base`, `image_model`, `image_size`
- **Documentation**: New `docs/IMAGE_PROVISIONERS.md` with complete provider configuration guide
- **Sync Script**: `scripts/sync.sh` to keep `skill/` and `plugins/` directories synchronized
- **Makefile Target**: `make sync` for easy directory synchronization

### Changed
- README: Enhanced platform-specific download instructions (Mac Intel vs Apple Silicon)
- README: Added AI image generation section with natural language examples
- `scripts/install.sh`: Added Linux ARM64 detection and support
- Default TuZi image size: `1024x1024` → `2048x2048` (TuZi requires minimum 3.7M pixels)

### Fixed
- URL download with query parameters: Fixed file name too long error when downloading generated images
- Platform detection in install script for ARM64 systems
- API base URL documentation: Updated to correct `https://api.tu-zi.com/v1`

### Technical Details
- **New Files**:
  - `internal/image/provider.go` - Provider interface and factory
  - `internal/image/openai.go` - OpenAI DALL-E provider
  - `internal/image/tuzi.go` - TuZi image generation provider
  - `docs/IMAGE_PROVISIONERS.md` - Provider configuration guide
  - `scripts/sync.sh` - Directory synchronization script

---

## [1.3.1] - 2025-01-12

### Added
- Auto-download binary from GitHub releases on first run
- User-friendly error messages with actionable guidance
- System dependencies declaration in `manifest.json`
- XDG-compliant cache directory (`~/.cache/claude/`) for binary storage
- Fallback to local `bin/` directory for development/offline usage
- Automatic version checking and update prompt when binary is outdated

### Changed
- Binary naming format now uses hyphens (`md2wechat-linux-amd64`) to match GitHub releases
- Error messages now show human-readable platform names (e.g., "macOS (Apple Silicon)")
- Download progress displays concise status information

### Fixed
- Binary name mismatch between workflow and run.sh that caused download failures

---

## [1.3.0] - 2025-01-11

### Added
- Plugin Marketplace support with `.claude-plugin/marketplace.json`
- One-command Claude Code installation via `/plugin marketplace add`
- Prominent Claude Code installation section at top of README
- Claude Code badge for quick identification
- Detailed binary installation instructions with location guidance
- Installation steps for Windows, Mac, and Linux users

### Changed
- Updated docs/QUICKSTART.md with Claude Code section at the beginning
- Enhanced docs/USAGE.md with Claude Code integration guide
- Improved download table with installation locations
- Added collapsible installation steps for each platform

### Installation
```bash
# Claude Code users (simplest)
/plugin marketplace add geekjourneyx/md2wechat-skill
/plugin install md2wechat@geekjourneyx-md2wechat-skill
```

---

## [1.2.0] - 2025-01-11

### Added
- Claude Code Skill support with `.claude-plugin/plugin.json`
- Claude Code Skill in `skills/md2wechat/` directory for distribution
- New API themes: `bytedance`, `apple`, `sports`, `chinese`, `cyber`
- Comprehensive troubleshooting guide in SKILL.md
- API theme selection section in README.md
- CHANGELOG.md with version history and upgrade guide

### Changed
- Updated themes.md with complete API and AI theme documentation
- Enhanced HTML guide with AI theme specific requirements
- Improved SKILL.md with detailed error handling and FAQ
- Updated command help text to reflect all available themes
- Enhanced FAQ.md with IP whitelist configuration guide

### Removed
- `leo` theme (deprecated)

---

## [1.1.0] - 2025-01-11

### Added
- YAML-based theme system for AI mode
- AI themes: `autumn-warm`, `spring-fresh`, `ocean-calm`
- Custom theme support with `custom.yaml`
- Theme configuration with color schemes and style info
- Reference documentation for themes, HTML guide, image syntax, and WeChat API

### Changed
- Refactored theme system to use YAML files instead of hardcoded prompts
- Updated SKILL.md with new AI theme workflow
- Enhanced themes.md with detailed style specifications

---

## [1.0.1] - 2025-01-11

### Fixed
- Mermaid diagrams rendering for GitHub documentation

---

## [1.0.0] - 2025-01-11

### Added
- Initial release of md2wechat
- API mode conversion using md2wechat.cn API
- AI mode conversion with Claude AI
- WeChat draft upload functionality
- Image upload to WeChat material library
- Configuration management with YAML support
- Command-line interface with cobra
- Multi-platform binary support (Windows, macOS, Linux)
- Comprehensive documentation (README, FAQ, USAGE)
- Test draft command for HTML validation

### Features
- Convert Markdown to WeChat Official Account formatted HTML
- Support for local images, online images, and AI-generated images
- Automatic image compression and optimization
- Draft creation with cover image support
- Environment variable and config file support
- Claude Code Skill integration

---

## Version History Summary

| Version | Date | Description |
|---------|------|-------------|
| [1.9.0] | 2025-02-06 | Image Post (小绿书/newspic) support |
| [1.8.0] | 2025-02-05 | OpenClaw support, directory simplification, run.sh refactoring |
| [1.7.0] | 2025-01-25 | ModelScope image provider, write command stdin support |
| [1.6.0] | 2025-01-19 | AI writing trace removal (Humanizer), write + humanize integration |
| [1.5.0] | 2025-01-17 | Writer style assistant, Dan Koe style, image size control |
| [1.4.0] | 2025-01-14 | TuZi image provider, natural language image generation |
| [1.3.1] | 2025-01-12 | Auto binary download, user-friendly errors, system dependencies |
| [1.3.0] | 2025-01-11 | Plugin Marketplace support, enhanced installation docs |
| [1.2.0] | 2025-01-11 | Claude Code plugin support, new API themes |
| [1.1.0] | 2025-01-11 | YAML theme system, AI themes (autumn-warm, spring-fresh, ocean-calm) |
| [1.0.1] | 2025-01-11 | Fixed Mermaid diagrams for GitHub rendering |
| [1.0.0] | 2025-01-11 | Initial release with full md2wechat functionality |

---

## Upgrade Guide

### From v1.1.0 to Unreleased

**New API Themes Available:**
```bash
# New themes available
md2wechat convert article.md --theme bytedance
md2wechat convert article.md --theme apple
md2wechat convert article.md --theme sports
md2wechat convert article.md --theme chinese
md2wechat convert article.md --theme cyber
```

**Claude Code Skill Integration:**
```bash
# Install as Claude Code Skill
cp -r skills/md2wechat ~/.claude/skills/
```

### From v1.0.0 to v1.1.0

**Theme System Migration:**
- Old theme names have been updated
- AI themes now use YAML configuration files
- Update your commands to use new theme names:
  - `autumn-warm` instead of `elegant`
  - `spring-fresh` instead of `minimal`
  - `ocean-calm` instead of `tech`

---

## Links

- [GitHub Repository](https://github.com/geekjourneyx/md2wechat-skill)
- [Documentation](README.md)
- [Issues](https://github.com/geekjourneyx/md2wechat-skill/issues)
