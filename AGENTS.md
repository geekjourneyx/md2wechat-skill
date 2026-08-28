# Agent 协作规范

This repository is a Go CLI project in stabilization. Treat documentation and execution rules as part of the product.

## Working Rules

1. Start by checking the current tree and the exact files that are in scope.
2. Do not revert user changes unless explicitly asked.
3. Prefer `apply_patch` for edits and avoid destructive commands.
4. Keep changes scoped; if a task crosses file boundaries, state the boundary before editing.
5. Preserve CLI and documentation compatibility unless the task explicitly calls for a breaking change.
6. When a task depends on providers, themes, or prompt templates, prefer CLI discovery output over guessing or stale docs.

## Anti-Noise Agent Decision Rules

Agent-facing commands must stay deterministic, observable, and explainable. Do not add hardcoded phrase lists, style word lists, or subjective quality heuristics to drive Agent decisions in commands such as `advise`, `inspect`, `preview`, `convert`, `upload`, or `draft`.

Allowed deterministic rules must be based on observable structure, explicit CLI/config state, or platform constraints, such as headings, images, metadata limits, local asset existence, credentials, supported themes, layout syntax, and command capabilities.

If a decision requires judging tone, naturalness, AI traces, writing quality, or editorial taste, keep it outside deterministic CLI routing. Require explicit user intent, host-Agent judgment, or an existing dedicated command. Do not make `advise` or readiness commands infer those judgments from hardcoded text patterns.

Every Agent-facing rule should pass these checks:

1. Is the signal observable without subjective interpretation?
2. Would two agents reach the same result from the same inputs?
3. Is the recommendation explainable from concrete evidence?
4. Does it avoid writing, rewriting, uploading, or publishing as a side effect?
5. Is it preventing a real wrong next step rather than adding a nice-to-have suggestion?

## Discovery-First Workflow

Before assuming a feature, resource, or prompt exists, inspect the running CLI:

1. `md2wechat capabilities --json`
2. `md2wechat providers list --json`
3. `md2wechat themes list --json`
4. `md2wechat prompts list --json`
5. `md2wechat layout list --json`

Use `show` or `render` only when a task depends on a specific resource:

1. `md2wechat providers show <name> --json`
2. `md2wechat themes show <name> --json`
3. `md2wechat prompts show <name> --kind <kind> --json`
4. `md2wechat prompts render <name> --kind <kind> --var KEY=VALUE --json`
5. `md2wechat layout show <name> --json`
6. `md2wechat layout render <name> --var KEY=VALUE`
7. `md2wechat layout validate --file <file.md> --json`

Treat these commands as the source of truth for:

- currently supported image providers
- currently visible themes after override resolution
- bundled and overridden prompt catalog entries
- advanced layout counts: 77 recommended source scenarios, 56 recommended syntax names, 3 compatibility modules, 4 base enhancements, and 63 render-layer syntax capabilities
- the nine recommended layout categories: `opening`, `infographic`, `judgment`, `evidence`, `conversion`, `brand`, `sprint4`, `free-layout`, and `interactive`
- humanizer intensity levels: `gentle` / `medium` / `aggressive` / `authentic`

**高级排版模块约束**：`layout` 模块使用通用 `:::name` 语法，仅在 **API 模式**（`convert` 默认）下渲染。`block` 是保留语法名，不是通用占位符。
AI 模式（`--mode ai`）不解析 `:::name` 高级排版块，这些块将以普通段落输出。

每个 recommended 语法必须有一个 canonical executable witness；只有结构不同的 renderer 分支才增加 variant witness。77 场景到推荐语法名的 mapping 只保存在测试中，不进入 CLI discovery 或运行时 catalog。

真实 conformance 必须证明 84 个 witness（56 个 recommended canonical、25 个结构不同的 non-default branch、3 个 compatibility）都由目标 API 语义渲染：响应成功、模块 marker 与稳定内容存在、原始 `:::name` fence 不残留。`layout validate` 成功不等于远端 renderer 已部署。

新内容读取 `layout show` 时，依次读取 `input_positions`、primary `body_format` 与对应 `Opener`/`Fields`/`Rows`/`Body`、canonical `Variants[].Name`、canonical `Example`。`compatible_body_formats` 和 `Variants[].Aliases` 只读且只服务旧稿兼容，不得作为新内容选择。

## Verification Order

1. Run `gofmt -l .` when Go files change.
2. Run `go vet ./...` after structural changes.
3. Run `make quality-gates` before declaring release-related or CI-sensitive work done. This is the local source of truth for the same gate sequence enforced in GitHub Actions.
4. Run `GOCACHE=/tmp/md2wechat-go-build go test ./...` for focused regression work when you do not need the full gate.
5. Run `GOCACHE=/tmp/md2wechat-go-build go test -cover ./...` when coverage is part of the task.
6. If `make release-check` exists in this branch, keep it green, but do not treat it as a substitute for `make quality-gates`.
7. When release assets or installer scripts change, run artifact smoke and installer smoke against the same bundle before calling the work done.
8. If the task touches release or installer paths, keep the documented primary path versioned and non-`latest`.
9. If the task touches Homebrew distribution, verify the tap formula is generated from the same release bundle and points only to versioned release assets.
10. If the task publishes a new npm version, manually trigger `npx cnpm sync @geekjourneyx/md2wechat` after npm publish so `npmmirror` users do not get a stale tarball 404 on the fresh release.
11. Before a release that changes layout rendering, theme rendering, API conversion, or release assets, run a real rendering E2E smoke test against the local API when that service is available. If the local API service is not running in the current environment, record the skip reason and keep `make quality-gates` green; do not treat missing `localhost:3000` alone as a release blocker for unrelated changes.

```bash
# 1. 确认本地 API 服务已启动（默认 http://localhost:3000）
# 2. 创建包含核心语法类别的测试文件，运行转换
./md2wechat convert <layout-test.md> --mode api --output /tmp/layout-smoke.html

# 3. 检查 HTML 中没有残留的原始 ::: 语法
python3 -c "
modules = ['hero','toc','verdict','audience-fit','myth-fact','metrics','compare','steps',
           'timeline','quote','callout','definition','author-card','subscribe',
           'faq','checklist','cta','notice','summary']
html = open('/tmp/layout-smoke.html').read()
failed = [m for m in modules if ':::' + m in html]
ok = [m for m in modules if ':::' + m not in html]
print(f'✅ 渲染成功: {len(ok)}/{len(modules)}')
if failed: print('FAIL (raw syntax not rendered):', failed)
else: print('PASS: all modules rendered')
"

# 4. 用 validate 命令检查语法正确性
./md2wechat layout validate --file <layout-test.md> --json
```

**通过标准（执行时）**：
- `convert` 输出的 HTML 中，所有已知模块均无 `:::name` 原始语法残留
- `layout validate` 返回 `LAYOUT_VALIDATED`（errors: 0）
- 两项都通过才可把 layout/API rendering 相关变更视为完成；若因本地 API 不可用而跳过，发布记录中必须写明未执行原因。

## Test Discipline

Tests are part of the product contract, not a coverage vanity metric.

When adding or changing a command, workflow, or user-facing behavior, start from first principles:

1. Which failure would most damage user trust?
2. Which failure would most mislead an agent into taking the wrong next step?
3. Which boundary must stay identical across `inspect`, `preview`, `convert`, `upload`, and `draft`?

Required rules:

1. Do not add tests just to raise a coverage number.
2. Prefer tests that protect a real contract, invariant, or blocking user workflow.
3. If a new command or feature changes a decision boundary, add tests for both the accepted path and the rejected path.
4. If a new feature introduces a “confirmation” step, its tests must prove that the confirmation layer stays aligned with the execution layer.
5. When behavior depends on combinations of inputs, prefer table-driven matrix tests over one-off happy-path tests.
6. When the code talks to external systems, keep a small number of real smoke tests, but make the default regression suite deterministic and local-first.
7. If a test is brittle, environment-dependent, or mostly re-tests implementation details without protecting user trust, do not add it unless there is a concrete regression history behind it.
8. A change is not CI-safe until the exact local gate that CI runs has passed. Do not rely on “I ran tests” if CI also runs lint, formatting, or release-consistency checks that you did not run locally.

Priority order for effective tests:

1. CLI contract tests
   - invalid input rejection
   - exit codes
   - JSON envelope / stdout stability
2. confirmation-vs-execution consistency tests
   - `inspect` / `preview` must not declare a path ready when `convert` / `draft` would fail
3. blocking readiness matrix tests
   - metadata limits
   - missing local assets
   - missing or invalid cover
   - missing credentials
4. core publish-path tests
   - asset rewrite
   - draft mapping
   - WeChat error translation
5. real smoke tests for one minimal external closed loop

Coverage guidance:

1. Low coverage alone is not a reason to add tests.
2. High-risk modules with stable contracts should be covered before low-risk helper modules.
3. A smaller set of durable, high-signal tests is better than broad but shallow assertion spam.
4. If `go test -cover ./...` is unreliable in the current environment, do not force it blindly; document the limitation and keep the deterministic subset healthy.

## Release And Version Discipline

1. Keep the version source singular and explicit when the repository has one.
2. Keep install scripts, release notes, changelog, and release assets aligned.
3. If the repository does not yet provide a workflow or release gate, document the gap instead of inventing it.
4. Release work is not complete until these are explicitly re-audited and aligned:
   - `VERSION`
   - `package.json`
   - `.claude-plugin/marketplace.json`
   - `skills/md2wechat/SKILL.md`
   - `platforms/openclaw/md2wechat/SKILL.md`
   - `scripts/install.sh`
   - `docs/INSTALL.md`
   - `docs/FAQ.md`
5. Run `bash scripts/quality-gates.sh` before declaring release work done. Step 0 checks VERSION / package.json / marketplace.json / CHANGELOG in under 1 second and fails fast if any are misaligned.
6. If any of the files above still carry an old version, stale release URL, stale maintainer identity, or outdated command examples, block the release and fix them before tagging.
7. Before committing or tagging a release, run a final release-safety audit and report the result:
   - Sensitive information: no real API keys, tokens, AppSecret values, proxy credentials, private server IPs, private ports, or ignored deployment details are present in tracked files.
   - Necessary scope: every changed tracked file is explainable as product code, tests, public documentation, release metadata, or repository operating rules for this release. Remove unrelated churn before commit.
   - Compatibility: unchanged/default configurations still follow the old behavior. For optional features, explicitly verify the unset path does not add new auth requirements, network routing, config validation failures, or changed command output that would break existing users.
   - Local-only artifacts: confirm `docs/superpowers/`, scratch notes, generated smoke files, and private environment files remain ignored or untracked and are not staged.

## Documentation Discipline

1. Document only supported paths as the primary path.
2. Label aspirational or not-yet-shipped paths clearly.
3. If a feature is not backed by current code or release assets, do not describe it as shipped.
4. Prefer short, operational instructions over marketing copy.
5. If any CLI command, subcommand, flag, JSON response shape, prompt asset, provider, or theme behavior changes, treat documentation sync as part of the same task.
6. The minimum documentation review set for CLI changes is:
   - `README.md`
   - `docs/DISCOVERY.md`
   - `docs/FAQ.md`
   - the relevant `SKILL.md`
   - `docs/HUMANIZE.md` when humanizer behavior or intensity levels change
7. If the CLI change affects setup, config, install, or migration behavior, also review:
   - `docs/CONFIG.md`
   - `docs/QUICKSTART.md`
   - `docs/USAGE.md`
   - `docs/OPENCLAW.md` when platform behavior changes
8. Do not stop at “updated one doc”. Re-audit the highest-signal entry points so user-facing docs, agent-facing docs, and CLI behavior stay aligned.
9. Treat `SKILL.md` changes as agent-runtime contract changes, not routine documentation churn. Review the relevant skill entry points whenever CLI behavior, prompt assets, providers, themes, or Agent workflows change, but modify them only when the change helps an Agent choose the correct command or avoid a dangerous side effect. Do not add full prompts, scoring formulas, long tutorials, README-style explanations, implementation file paths, release notes, or marketing copy to `SKILL.md`.
10. When image prompts change, treat both skill entry points as mandatory review targets, but apply the `SKILL.md` necessity rule above before editing:
   - `skills/md2wechat/SKILL.md`
   - `platforms/openclaw/md2wechat/SKILL.md`
11. Keep configuration naming layers explicit:
   - config-file YAML keys such as `api.image_base_url`
   - environment variables such as `IMAGE_API_BASE`
   - `config show --format json` output keys such as `image_api_base`
   Do not mix these three layers in docs or agent guidance.

## Image Prompt Discipline

When adding or changing `internal/assets/builtin/prompts/image/*.yaml`, treat the change as a product contract update, not just a content tweak.

Minimum required fields for builtin image prompts:

- `name`
- `kind: image`
- `description`
- `version`
- `archetype`
- `primary_use_case`
- `recommended_aspect_ratios`
- `default_aspect_ratio`
- `metadata.author`
- `metadata.provenance`
- `template`

Use these when applicable:

- `compatible_use_cases`
- `tags`
- `examples`
- `metadata.inspired_by`

Required constraints:

1. `default_aspect_ratio` must appear in `recommended_aspect_ratios`.
2. If a preset can serve both cover and infographic scenarios, express that through `compatible_use_cases` rather than duplicating the preset.
3. Do not move long image prompts back into Go code when the prompt catalog can carry them.

Required verification after image prompt changes:

1. `gofmt -l .`
2. `GOCACHE=/tmp/md2wechat-go-build go test ./internal/promptcatalog ./cmd/md2wechat`
3. `GOCACHE=/tmp/md2wechat-go-build go test ./...`
4. Re-audit:
   - `README.md`
   - `docs/DISCOVERY.md`
   - `docs/FAQ.md`
   - `skills/md2wechat/SKILL.md`
   - `platforms/openclaw/md2wechat/SKILL.md`

The repository should fail closed on prompt drift: missing primary use case, default aspect ratio, or attribution metadata should be blocked by tests rather than caught manually later.

## Escalation

1. Ask for approval before commands that require network access or writing outside the workspace.
2. If a command fails because of sandboxing, rerun it with the required escalation instead of working around it.

## Local-Only Artifacts

`docs/superpowers/` (plans, specs, scratch notes) is gitignored and must never be committed to the remote repository. Keep all superpowers working files local only.

## Brand Profile

The Brand Profile lives at `~/.config/md2wechat/brand.md` (Markdown format, not YAML). Initialize with `md2wechat brand init`. Agents read it for style context; the CLI does not parse it. Never commit brand.md to the repository.
