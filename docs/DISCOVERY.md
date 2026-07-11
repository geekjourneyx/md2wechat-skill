# 能力发现与 Prompt Catalog

`md2wechat` 提供了一组面向 Agent 和自动化脚本的发现命令，用来在执行前先确认当前 CLI 支持什么能力、有哪些可用资源、当前配置是否就绪。

这组命令的定位不是替代 `--help`，而是提供**可机读、可枚举、可提前探测**的能力接口。

## 推荐使用方式

对于 Agent、脚本或 CI，discovery 应服务于下一步决策，而不是每次任务都全量枚举。使用最小必要集合：

- 版本、能力或行为不确定：`md2wechat version --json`、`md2wechat capabilities --json`
- API、草稿、上传或配置 readiness：`md2wechat doctor --json`，必要时再 `md2wechat config show --format json`
- 多公众号本地配置检查：`md2wechat config wechat-accounts --json`
- 文章排版且用户未指定主题或模块：`md2wechat themes list --json`、`md2wechat layout list --json`
- 已有文章或初稿，不确定下一步是否需要标题、封面或排版：`md2wechat advise <article.md> --json`
- 已指定某个资源：使用对应的 `providers show`、`themes show`、`prompts show` 或 `layout show`
- 图片生成或图片 prompt 选择：`md2wechat providers list --json`、`md2wechat prompts list --kind image --json`
- 公众号标题建议：`md2wechat title suggest <article.md> --json`，必要时再 `md2wechat prompts show wechat-title-expert --kind title --json`
- Agent SOP 不确定或需要读取当前版本 skill：`md2wechat skills list --json`、`md2wechat skills read md2wechat --json`
- 简单本地操作，例如 `preview`、`humanize` 或用户已给完整命令和 flags：不要运行无关的 catalog discovery

## 能力总览

```bash
md2wechat capabilities --json
```

返回内容包含：

- 已开放的高层命令能力
- `convert` 支持的模式和默认模式
- `inspect` / `preview` 这类确认层命令
- 当前可枚举的图片 provider
- `title_generation` 的 host-Agent handoff 边界
- 当前可枚举的 theme
- 当前可枚举的 prompt catalog
- `data.article_advice` 是否可用、命令名、建议工具、响应 code、是否要求 JSON、是否有副作用
- `layout` catalog 是否可用、模块数量、schema version、是否仅 API 模式渲染
- `skills` 是否作为当前二进制的内置 Agent SOP 入口开放

示例片段：

```json
{
  "article_advice": {
    "available": true,
    "command": "advise",
    "tools": ["title", "cover", "layout", "micro_edit"],
    "response_code": "ADVISE_COMPLETED",
    "requires_json": true,
    "side_effects": false
  },
  "layout": {
    "available": true,
    "module_count": 53,
    "recommended_syntax_count": 53,
    "recommended_scenario_count": 68,
    "compatibility_module_count": 3,
    "base_enhancement_count": 4,
    "render_syntax_count": 60,
    "supports_validate": true,
    "api_mode_only": true,
    "schema_version": "1"
  },
  "commands": ["convert", "inspect", "advise", "preview", "layout", "themes", "skills"]
}
```

未出现在 `commands` 中的命令不应被 Agent 当成可执行能力。未来工作流不通过 `capabilities` 预告。

## 内置 Skill SOP

```bash
md2wechat skills list --json
md2wechat skills read md2wechat
md2wechat skills read md2wechat --json
```

`skills` 命令把 `skills/md2wechat/SKILL.md` 随二进制一起嵌入，供 Agent 在离线或只拿到二进制的环境中读取当前版本 SOP。这样 Agent 不需要猜 README、联网拉仓库，或读取一个可能和当前 CLI 版本不一致的本地 skill 副本。

`skills list` 始终返回标准 JSON envelope，包含可用 skill、数量和 frontmatter 摘要。`skills read` 默认输出原始 Markdown，适合直接喂给 Agent；加 `--json` 时会返回标准 JSON envelope，并把 Markdown 放在 `data.content`。

## 本地体检

```bash
md2wechat doctor --json
```

`doctor` 是本地只读诊断命令。它检查配置能否加载、默认转换模式、API key 是否存在、默认主题是否兼容当前模式、layout catalog 是否可用、WeChat 草稿凭证是否存在。

它不会调用远程 API，不会验证 live auth，不会上传图片，也不会创建草稿。JSON 输出中的 `data.live` 固定为 `false`，`data.readiness.format_api` / `data.readiness.advanced_layout` / `data.readiness.draft` 用来帮助 Agent 判断下一步能不能执行 API 转换、高级排版渲染或草稿创建。

注意：`doctor` 的 `data.readiness.*` 是本地配置可尝试性；`inspect` 的 `data.readiness.targets/blockers` 是单篇文章的执行目标状态。

## 多公众号发现

```bash
md2wechat config wechat-accounts --json
```

`config wechat-accounts` 是本地只读命令，用来列出配置文件中的命名公众号账号、当前解析到的账号和 `default_account`。它不调用 `/api/auth/validate`，不要求 `MD2WECHAT_API_KEY`，也不会输出 secret。

命名账号只在上传、生成并上传图片、创建草稿、创建图片消息，以及 `convert --upload` / `convert --draft` 等有微信副作用的路径上触发 API-key 校验。

## 确认层命令

在真正执行 `convert`、`upload`、`draft` 之前，推荐先调用：

```bash
md2wechat inspect article.md --json
md2wechat inspect article.md --upload --json
md2wechat inspect article.md --draft --cover ./cover.jpg --json
md2wechat preview article.md --json
```

其中：

- `inspect` 用来确认最终标题、作者、摘要来源，以及 `data.readiness.targets/blockers` 中的 `convert/upload/draft` 目标状态。
- Agent 在决定是否继续 `convert`、`upload`、`draft` 前，必须用即将执行的 same publish target 运行 `inspect`，并读取 `inspect --json` 的 `data.readiness.targets` 和 `data.readiness.blockers`。
- `inspect` 的 `checks` 会直接暴露语义边界，例如 `TITLE_BODY_MISMATCH`、`DIGEST_METADATA_ONLY`、`IMAGE_REPLACEMENT_REQUIRES_UPLOAD_OR_DRAFT`。
- `preview` 第一版会生成本地 HTML 预览文件；`--json` 返回输出路径和 render metadata。
- `preview --mode ai` 不会声称展示最终视觉稿，只会明确降级为确认页。
- `--json` 走稳定 machine-readable contract；stdout 只保留 JSON，便于 Agent 和脚本直接解析。

## Article Advice

```bash
md2wechat advise article.md --json
```

`advise --json` 是只读决策路由器。它根据文章内可观察信号推荐是否调用 `title suggest`、`generate_cover --plan --json`，或检查/应用合适的 layout modules；也可能建议人工确认的 `micro_edit`。

它不生成内容、不写回文件、不上传、不创建草稿。发布前是否可执行仍以 `inspect --json` 的 `data.readiness.targets/blockers` 为准。

## 图片 Provider

```bash
md2wechat providers list --json
md2wechat providers show openai --json
md2wechat providers show openrouter --json
md2wechat providers show volcengine --json
```

当前 CLI 会枚举 provider 元数据，包括：

- `name`
- `aliases`
- `description`
- `required_config`
- `optional_config`
- `default_base_url`
- `default_model`
- `supported_models`
- `supports_size`
- `current`
- `configured`

当前内置支持的图片 provider：

- `openai`
- `tuzi`
- `modelscope` / `ms`
- `openrouter` / `or`
- `gemini` / `google`
- `volcengine` / `volc`

## 主题发现

```bash
md2wechat themes list --json
md2wechat themes show default --json
md2wechat themes show autumn-warm --json
```

主题信息来自运行时 ThemeManager，遵循当前加载优先级：

1. `MD2WECHAT_THEMES_DIR`
2. `./themes`
3. `~/.config/md2wechat/themes`
4. 内置 theme 资产

这意味着纯二进制安装也能列出官方默认主题，用户和平台仍可通过目录覆盖内置主题。

`themes list --json` 和 `themes show --json` 返回的是稳定 view，而不是内部 Go struct。关键字段包括：

- `name`
- `type`: `api` 或 `ai`
- `selectable`: 是否可被 `convert --theme` 直接选择
- `api_theme`: API 服务实际使用的主题名
- `style`: Agent 选主题时可参考的风格元数据
- `metadata_incomplete`: 主题缺少稳定风格字段时为 `true`

Collection descriptor 也会出现在发现结果里。例如 `api-collection` 用于描述 API 主题集合，但它不是可执行主题，所以 `selectable: false`。`api.yaml` 中的分组条目会展开为可执行 API 主题。Agent 必须选择 `selectable: true` 且 `type` 匹配当前模式的主题。

`convert` 和 `inspect` readiness 现在会 fail closed：

- `--mode api` 不能选择 AI 主题；
- `--mode ai` 不能选择 API 主题；
- `selectable: false` 的主题不能用于转换；
- `--mode ai --custom-prompt` 是例外路径，custom prompt 本身就是排版提示词来源，不要求主题匹配；检查这个路径时，`inspect` 也要带同样的 `--custom-prompt`。

## Prompt Catalog

Prompt catalog 是一组内置并可覆盖的 YAML 资产，当前主要用于：

- `humanizer`
- `write` 的润色流程
- `title suggest` 的公众号标题候选生成请求
- 后续扩展的图片 archetype

### 列出 Prompt

```bash
md2wechat prompts list --json
md2wechat prompts list --kind humanizer --json
md2wechat prompts list --kind refine --json
md2wechat prompts list --kind title --json
md2wechat prompts list --kind image --json
md2wechat prompts list --kind image --archetype cover --json
md2wechat prompts list --kind image --archetype infographic --json
md2wechat prompts list --kind image --tag editorial --json
```

### 查看 Prompt 定义

```bash
md2wechat prompts show medium --kind humanizer --json
md2wechat prompts show authentic --kind humanizer --json
md2wechat prompts show default --kind refine --json
md2wechat prompts show wechat-title-expert --kind title --json
md2wechat prompts show cover-default --kind image --json
md2wechat prompts show cover-hero --kind image --archetype cover --tag hero --json
md2wechat prompts show infographic-victorian-engraving-banner --kind image --archetype infographic --tag victorian --json
```

对于图片 prompt，`archetype` 表示主要分组，不代表只能用于这一种场景。优先查看 `prompts show --json` 返回的：

- `primary_use_case`
- `compatible_use_cases`
- `recommended_aspect_ratios`
- `default_aspect_ratio`

这样 Agent 能判断某些信息图 preset 是否也适合作为封面使用。

### 渲染 Prompt 模板

```bash
md2wechat prompts render cover-default \
  --kind image \
  --var article_title='从 0 到 1 做好公众号封面' \
  --var article_summary='一份关于封面图策略的实战清单' \
  --json
```

### 用 preset 直接生成图片

```bash
md2wechat generate_image --preset cover-hero --article article.md
md2wechat generate_cover --article article.md
md2wechat generate_infographic --article article.md --preset infographic-comparison
md2wechat generate_infographic --article article.md --preset infographic-dark-ticket-cn --aspect 21:9
md2wechat generate_infographic --article article.md --preset infographic-handdrawn-sketchnote
md2wechat generate_infographic --article article.md --preset infographic-apple-keynote-premium
md2wechat generate_infographic --article article.md --preset infographic-victorian-engraving-banner --aspect 21:9
md2wechat generate_image --preset cover-hero --article article.md --model gemini-3-pro-image-preview
```

高频图片命令和 prompt catalog 的关系是：

- `generate_image`: 通用入口，可直接传 raw prompt，也可用 `--preset`
- `generate_cover`: `cover` archetype 的薄包装命令
- `generate_infographic`: `infographic` archetype 的薄包装命令
- 三个图片命令都支持 `--model`，用于单次覆盖本次调用的图片模型

如果某个图片 preset 的 `compatible_use_cases` 包含 `cover`，那么它也可以被 `generate_cover` 使用；默认画幅优先跟随 prompt 自身声明的 `default_aspect_ratio`。

### Agent 图片计划

当当前 Agent 运行时暴露 Image Gen 工具时，可以让图片命令只返回计划，交给宿主 Agent 执行：

```bash
md2wechat generate_cover --article article.md --plan --json
md2wechat generate_infographic --article article.md --preset infographic-comparison --plan --json
md2wechat generate_image "一张适合公众号文章的产品发布插画" --plan --json
```

JSON envelope 会返回 `IMAGE_PLAN_READY` 和 `status: action_required`。关键执行边界在 `data` 中：

- `side_effects:false`
- `requires_provider:false`
- `requires_image_api_key:false`
- `execution_owner:"host_agent"`

这条路径不要求或使用 `IMAGE_API_KEY` 进行图片 provider 调用，不请求 provider，也不上传微信。完整教程见 [Agent 图片计划模式](AGENT_IMAGE_GEN.md)。

### 标题建议请求

`title suggest` 会读取文章内容，渲染内置标题 prompt，然后返回一个需要宿主 Agent / 外部模型继续执行的 JSON 请求：

```bash
md2wechat title suggest article.md --json
md2wechat title suggest article.md --target-reader "独立开发者" --count 10 --max-title-chars 25 --hook-level 2 --json
```

JSON envelope 会返回 `TITLE_SUGGEST_REQUEST_READY` 和 `status: action_required`。关键执行边界在 `data` 中：

- `action:"ai_title_suggestion_request"`
- `execution_owner:"host_agent"`
- `prompt_kind:"title"`
- `prompt_name:"wechat-title-expert"`
- `hook_level:1|2|3`
- `hook_level_label:"restrained"|"punchy"|"high_tension"`
- `side_effects:false`
- `requires_external_model:true`
- `recommendation_only:true`

这条路径不会调用模型、不会写回 Markdown、不会创建草稿，也不会自动替用户确认最终标题。Agent 应读取 `data.prompt`，让外部模型生成候选标题，再把候选交给用户确认或在自己的上层流程中选择最高分。

`md2wechat capabilities --json` 的 `data.title_generation` 会暴露标题能力边界：

- `hook_levels`：三个对象 `{level,label,description}`
- `default_hook_level:1`
- `max_recommended_hook_level:2`
- `level_3_requires_evidence_basis:true`

这些字段只描述 host-Agent handoff 的请求能力，不表示 CLI 会本地生成标题、调用模型、选择最终标题或写回文章。

完整的人类/Agent 工作流、`--hook-level` 选择建议和候选 JSON 字段说明见 [公众号标题建议](TITLE_SUGGEST.md)。

当前内置 prompt kind：

- `humanizer`
- `refine`
- `title`
- `image`

当前内置图片 archetype 分组：

- `cover`
- `infographic`

当前内置图片模板示例：

- `cover-default`
- `cover-hero`
- `cover-minimal`
- `cover-metaphor`
- `cover-editorial`
- `cover-illustrated`
- `cover-data-visual`
- `infographic-default`
- `infographic-comparison`
- `infographic-timeline`
- `infographic-dashboard`
- `infographic-hierarchy`
- `infographic-bento`
- `infographic-process`
- `infographic-flat-vector-panorama`
- `infographic-dark-ticket-cn`
- `infographic-handdrawn-sketchnote`
- `infographic-apple-keynote-premium`
- `infographic-victorian-engraving-banner`

## Prompt 资产覆盖顺序

Prompt catalog 的加载优先级为：

1. `MD2WECHAT_PROMPTS_DIR`
2. `./prompts`
3. `~/.config/md2wechat/prompts`
4. 内置 prompt 资产

也就是说，官方默认 prompt 会随二进制一起提供；用户和平台如需自定义，可按上面的顺序覆盖。

## 当前已接入 Prompt Catalog 的能力

目前已经优先使用 prompt catalog 的能力有：

- `humanize`
- `write` 的润色流程
- `title suggest`

当前仍然主要依赖代码或其他资产的部分：

- `convert` 的 API/AI 调用逻辑
- 更复杂的图片生成流程编排

后续扩展更多封面、信息图、配图 archetype 时，应优先新增 `prompts/image/*.yaml`，而不是直接把大段提示词写进 Go 代码。

## Layout Module Discovery (:::module Syntax)

The `layout` subcommand exposes 53 个主推 advanced WeChat layout syntax names (`:::module`) by default. Capability discovery separately reports 68 个主推高级排版场景条目, 3 个兼容模块, 4 个基础增强能力, and 60 项渲染层语法能力; these counts are not interchangeable. The 68 scenarios are source-use-case mappings, while the 53 names are the default CLI discovery objects. Maintainer tests map 48 of those names from source scenarios and keep five guide-only names explicit; the CLI does not publish that test-only mapping.

### Commands

```bash
# List all built-in modules
md2wechat layout list --json

# Filter by purpose (serves one of: attention | readability | memorability | conversion)
md2wechat layout list --serves attention --json
md2wechat layout list --category opening --json
md2wechat layout list --tag brand --json

# Show full spec (fields, serves, when_to_use, example, metadata)
md2wechat layout show hero --json

# Render a :::module block from structured vars
md2wechat layout render hero \
  --var eyebrow=深度观察 \
  --var title="公众号排版的真问题" \
  --json

# Validate :::module usage in a Markdown file
md2wechat layout validate --file article.md --json
md2wechat layout validate --stdin --json < article.md
```

`layout list --json` 会在模块摘要中显示 `body_format`；`layout show --json` 会显示完整 schema、canonical `Example` 和结构不同的 `Variants[].Example`。Schema 定义合法输入；Example 是经过验证的可执行 witness，应复用而不是手猜语法。正文格式共有 `fields` / `rows` / `json_object` / `json_array` / `markdown_images` / `markdown_fields` / `split` / `lines` / `dialogue` 九种；`compatible_body_formats` 只表示模块仍接受的旧正文格式。

默认 `layout list --json` 只返回 recommended lifecycle。旧稿迁移时才运行 `layout list --lifecycle compatibility --json`；不要把 `dialogue`、`gallery`、`longimage` 推荐给新稿。复杂正文使用 `layout render <name> --body-file <path>`（或 `--body-file -` 从 stdin 读取），opener 参数用可重复的 `--param KEY=VALUE`，方括号 caption 用 `--caption`。

When the user says "帮我排版这篇文章" without naming a theme or module, run discovery first, then use `layout list`, `layout show`, and `layout render` as primitives. The CLI does not parse `~/.config/md2wechat/brand.md`; Agents should read Brand Profile themselves and choose the final theme/modules. Keep the source Markdown read-only: create a temporary formatted Markdown artifact, validate it with `layout validate`, then pass that temporary file to `convert`. Saving generated Markdown near the source requires explicit user confirmation.

### `--serves` Filter Values

| Value | Purpose |
|-------|---------|
| `attention` | 让读者知道值不值得读（hero, cards, verdict, audience-fit） |
| `readability` | 让手机阅读不累（part, toc, steps, label-title） |
| `memorability` | 让读者记住一个判断/品牌（verdict, manifesto, author-card） |
| `conversion` | 让读者收藏/关注/咨询/转发/购买（cta, subscribe, faq, cases） |

### Module Override (4-Level)

Custom module YAMLs are merged in this order (later wins):

1. **Builtin** — 53 recommended modules plus 3 compatibility modules (shipped with binary)
2. `~/.config/md2wechat/layout/` — user-global overrides
3. `./layout/` — project-local overrides
4. `$MD2WECHAT_LAYOUT_DIR` — env var (highest priority)

To add or override a module, create `<category>/<name>.yaml` in any override directory with `schema_version: "1"`.

### Unknown Module Strategy

`layout validate` reports **warnings** (not errors) for unknown module names. This allows forward-compatible documents where new modules are used before a CLI upgrade. Only known modules with missing required fields produce **errors** (exit 1).

### JSON Error Codes

| Code | Meaning |
|------|---------|
| `LAYOUT_MODULE_NOT_FOUND` | Named module does not exist in catalog |
| `LAYOUT_INVALID_FILTER` | Missing required input (e.g., neither --file nor --stdin) |
| `LAYOUT_MISSING_REQUIRED_FIELD` | Required field absent in render call |
| `LAYOUT_INVALID_FIELD_VALUE` | Field value not in allowed enum |
| `LAYOUT_VALIDATE_HAS_ERRORS` | Validation found errors (exit 1) |
| `LAYOUT_VALIDATED` | Validation passed clean (exit 0) |

## 与配置的关系

发现命令不会替代配置文件，但会帮助 Agent 确认：

- 当前默认 provider 是什么
- 当前 provider 是否已配置
- 当前是否存在某个 theme / prompt
- 当前 theme 是否可直接选择，是否匹配 `api` / `ai` 模式
- 高级排版模块 catalog 是否可用
- 每个高级排版模块的 `body_format` 和 `compatible_body_formats`，即正文应采用九种受支持格式中的哪一种
- 尚未发布的工作流是否只是声明为 `available: false`

配置主路径仍然是：

- `~/.config/md2wechat/config.yaml`

如需切换 API 域名、图片 provider 或其它默认值，请先看 [CONFIG.md](CONFIG.md)。
