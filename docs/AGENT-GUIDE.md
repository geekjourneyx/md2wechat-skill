# md2wechat Agent 操作手册

本文档为 AI Agent 提供操作 md2wechat CLI 的完整指引，涵盖所有典型场景的命令序列。

> **阅读对象**：Claude Code、OpenClaw、Copilot CLI 等 AI 代理（Agent），以及希望理解 Agent 决策流程的用户。

---

## 一、能力发现（按任务决策执行）

Agent 应把 CLI discovery 当成事实来源，但不要把所有发现命令当成每次任务的固定仪式。先判断下一步需要做什么，再运行最小必要的 discovery：

- 文章排版且用户未指定主题或模块：运行 `themes list --json` 和 `layout list --json`
- 已指定某个主题、provider、prompt 或 layout 模块：优先运行对应的 `show <name> --json`
- 图片生成或图片 prompt 选择：运行 `providers list --json` 和相关 `prompts list --kind image --json`
- 草稿、上传、API 本地可尝试性或配置排障：运行 `doctor --json` 和必要的 `config show --format json`
- CLI 版本、命令能力或行为边界不确定：运行 `version --json`、`capabilities --json`，必要时再运行 `skills list --json` 和 `skills read md2wechat --json`
- 当前外部 skill、README 或仓库 checkout 可能滞后于 PATH 上的二进制：运行 `skills read md2wechat --json` 读取当前二进制内置 SOP

简单本地操作（例如 `preview`、`humanize`，或用户已经给出完整命令和 flags）不需要运行无关的 provider/theme/prompt/layout discovery。

### 1.1 查看完整能力集

```bash
md2wechat capabilities --json
```

响应内容包括：
- 已开放的所有高层命令
- `convert` 支持的模式（api / ai）与默认模式
- `inspect` 和 `preview` 的可用选项
- 支持的图片 provider、主题、prompt 种类数量

只有在命令能力、默认模式或版本边界不确定时才需要先跑此命令。

### 1.1.1 读取当前二进制内置 SOP

```bash
md2wechat skills list --json
md2wechat skills read md2wechat
md2wechat skills read md2wechat --json
```

`skills list` 始终返回 JSON envelope，用来发现当前二进制内置了哪些 Agent skill。`skills read` 默认输出原始 Markdown，适合直接喂给 Agent；加 `--json` 时返回标准 JSON envelope，并把 SOP 放在 `data.content`。

当外部 skill 安装副本、README 或本地仓库可能不是当前 CLI 版本时，以 `md2wechat skills read md2wechat --json` 为准。OpenClaw 平台 metadata 仍以 `platforms/openclaw/md2wechat/SKILL.md` 为准，不要把它和内置 coding-agent SOP 混成同一层。

### 1.2 查看可用主题

```bash
md2wechat themes list --json
```

列出所有当前可用的主题。主题遵循以下加载优先级：
1. `MD2WECHAT_THEMES_DIR` 环境变量指向的目录
2. `./themes` 本地目录
3. `~/.config/md2wechat/themes` 用户目录
4. 内置主题资产

当需要选择主题，或 Brand Profile / 用户指令指定了主题时，Agent 必须读取 `type` 和 `selectable`：API 模式只能选择 `type: api` 且 `selectable: true` 的主题；AI 模式只能选择 `type: ai` 且 `selectable: true` 的主题。使用 `md2wechat themes show <theme_name> --json` 查看单个主题的详细配置。

### 1.3 查看可用图片 Provider（6+ 个）

```bash
md2wechat providers list --json
```

仅在图片生成、封面、信息图或 provider 选择任务中使用。列出所有支持的图片生成 provider：
- `openai` — OpenAI GPT Image / DALL·E
- `tuzi` — 图子 AI
- `modelscope` / `ms` — ModelScope
- `openrouter` / `or` — OpenRouter
- `gemini` / `google` — Google Gemini
- `volcengine` / `volc` — 火山引擎

使用 `md2wechat providers show <provider_name> --json` 查看单个 provider 需要的配置和支持的模型。

### 1.4 查看 Prompt 模板库

```bash
md2wechat prompts list --json
```

仅在需要选择或渲染 prompt 模板时使用。列出所有内置 prompt 模板，包括：
- `humanizer` — 人性化润色（4 个强度 + authentic 模式）
- `refine` — 内容精修
- `image` — 图片生成

使用 `md2wechat prompts show <name> --kind <kind> --json` 查看单个 prompt 的具体内容和配置。

### 1.5 查看高级排版模块（默认 53 个推荐语法名）

```bash
md2wechat layout list --json
md2wechat layout list --serves attention --json     # 按目标筛选
md2wechat layout list --serves readability --json
md2wechat layout list --serves memorability --json
md2wechat layout list --serves conversion --json
md2wechat layout list --lifecycle compatibility --json  # 仅旧稿迁移
```

仅在高级排版、模块选择或模块语法排障时使用。默认列表只返回 recommended lifecycle；compatibility 列表只用于迁移旧稿。每个推荐模块都有四个服务目标之一或多个：
- `attention` — 吸引注意力（hero、verdict、toc）
- `readability` — 提升可读性（steps、callout、bridge）
- `memorability` — 增强记忆点（quote、metrics、summary）
- `conversion` — 驱动转化（cta、subscribe、faq）

**⚠️ 重要约束**：高级排版语法（`:::module` ... `:::` 语法）**仅在 API 模式**下渲染。使用 `--mode ai` 时，排版语法会被当作普通段落输出。

---

## 二、配置发现与验证

### 2.1 查看当前配置

```bash
md2wechat config show --format json
```

这个命令返回当前生效的配置，已合并以下三层优先级：
1. 配置文件 `~/.config/md2wechat/config.yaml`（或通过 `--config` 指定）
2. 环境变量（如 `WECHAT_APPID`、`IMAGE_API_KEY`）
3. 硬编码默认值

Agent 应该用这个命令来确认当前配置是否就绪，而不是手动逐一检查。

### 2.2 初始化配置文件（仅用户第一次）

```bash
md2wechat config init
```

这会在 `~/.config/md2wechat/config.yaml` 创建示例配置文件。Agent 不应该自动调用此命令，除非用户明确要求重新初始化。

### 2.3 验证配置

```bash
md2wechat config validate
```

`config validate` 只说明配置能加载和解析，不代表每条执行路径都已经就绪。

### 2.4 本地体检（执行前推荐）

```bash
md2wechat doctor --json
```

`doctor` 是本地只读诊断命令。它检查 config load、默认转换模式、API key/base presence、默认主题兼容性、layout catalog、WeChat 草稿凭证。它不做 live auth、不上传图片、不创建草稿。

Agent 应读取：

- `data.overall`：`ready` / `degraded` / `blocked`
- `data.live`：固定为 `false`
- `data.readiness.format_api`：是否可执行 API 转换
- `data.readiness.advanced_layout`：是否可渲染高级排版模块
- `data.readiness.draft`：是否具备创建草稿的本地凭证

`doctor` 的 `data.readiness.*` 是本地配置可尝试性；`inspect` 的 `data.readiness.targets/blockers` 是单篇文章的执行目标状态。两者不要混用。

---

## 三、Brand Profile 操作

Brand Profile 是一个可选的品牌档案，位于 `~/.config/md2wechat/brand.md`，由 Agent 读取但不被 CLI 解析。

它使用 **Markdown 格式**，而不是 YAML，让 Agent 可以直接理解自然语言描述的品牌风格和偏好。

### 3.1 检查 Brand Profile 是否存在

```bash
md2wechat brand show --json
```

响应解读：
- `code: "BRAND_SHOWN"` → 档案存在，`data.content` 包含完整 Markdown 文本
- `code: "BRAND_NOT_FOUND"` → 档案不存在，可引导初始化
- `code: "BRAND_READ_FAILED"` → 文件不可读（权限问题等）

### 3.2 读取 Brand Profile

Agent 读取方式：

```python
import os

brand_path = os.path.expanduser("~/.config/md2wechat/brand.md")
brand_content = ""
if os.path.exists(brand_path):
    with open(brand_path) as f:
        brand_content = f.read()

# brand_content 包含完整的 Markdown 文本
# Agent 将其作为上下文注入到排版决策中
```

或通过 CLI 命令获取：

```bash
result=$(md2wechat brand show --json)
brand_content=$(echo "$result" | jq -r '.data.content')
```

### 3.3 Brand Profile 引导流程（如果不存在）

当 `md2wechat brand show` 返回 `BRAND_NOT_FOUND` 时，Agent 不应阻塞当前任务。只在任务开始前提示一次：

> 我没有看到 Brand Profile，这次会使用系统默认风格。如果你想固定 CTA、作者信息或长期语气，可以稍后让我帮你设置。

然后继续执行当前排版/转换任务。只有当用户明确要求“设置 Brand Profile”或“brand init”时，才进入 3 问引导并创建或编辑 `~/.config/md2wechat/brand.md`。

---

## 四、Markdown 排版诊断

在将文章交给 `convert` 之前，Agent 应该先做"排版诊断"来理解文章的结构和需求。

### 4.0 产物边界

当用户只说“帮我排版这篇文章”，Agent 应自己读取文章、Brand Profile 和 discovery 输出做排版判断。CLI 只提供事实、schema、渲染和校验，不替 Agent 做审美决策。

Agent 不应把 `:::module` 直接写回用户原文。应复制原文到临时 Markdown，插入 `layout render` 生成的模块，运行 `layout validate --file <temp.md> --json`，再把临时 Markdown 交给 `convert`。只有用户明确要求保存排版稿时，才可以把生成稿写到源文件旁边，且不能覆盖原文。

### 4.1 诊断三步法

**Step 1 — 意图识别**

Agent 应先从文章本身和用户指令中判断这三个问题；只有关键信息缺失且会影响执行结果时，才向用户追问：
- 这篇文章的目标读者是谁？
- 这篇文章想让读者下一步做什么（conversion goal）？
- 读完后读者最应该记住什么（memorability anchor）？

**Step 2 — 内容映射**（基于四目标框架）

使用 CLI 默认返回的推荐排版语法中的这些映射关系：

| 目标 | 常见问题 | 推荐模块 | 示例 |
|------|---------|---------|------|
| **attention** | 这篇值不值得看？ | hero, verdict, toc, audience-fit | 用 hero 展示视觉吸引力 |
| **readability** | 手机上读起来累不累？ | steps, callout, bridge, part | 用 callout 突出关键信息 |
| **memorability** | 读完记住什么？ | quote, metrics, summary, author-card | 用 metrics 展示数据 |
| **conversion** | 下一步做什么？ | cta, subscribe, faq, notice | 用 cta 明确行动 |

**Step 3 — 模块选择与验证**

结合 Brand Profile 的自然语言约束和内容特性，从 discovery 结果中选择最合适的推荐模块。保持用户原文只读，把生成稿写入临时 Markdown，然后验证临时稿：

```bash
md2wechat layout validate --file /tmp/md2wechat-format/<run-id>/article.formatted.md --json
```

### 4.2 查看单个模块详情与示例

```bash
# 查看模块完整规范
md2wechat layout show hero --json
md2wechat layout show callout --json

# 渲染模块 Markdown 片段（本地命令，不调用远程 API）
printf '%s\n' '重要提示' | md2wechat layout render callout --body-file - --json
```

读取 `layout list --json` / `layout show --json` 的 `body_format` 决定正文写法：`fields` / `rows` / `json_object` / `json_array` / `markdown_images` / `markdown_fields` / `split` / `lines` / `dialogue`。Schema 决定合法性；优先复用 canonical `Example`，只有结构不同的分支才选 `Variants[].Example`。复杂正文用 `--body-file`，不要拼成大量 `--var`。兼容模块仅用于旧稿迁移。

---

## 五、文章转换（convert）

### 5.1 基础转换命令

#### 模式 A：API 模式（默认，支持高级排版）

```bash
md2wechat convert article.md --output output.html
```

**前置条件**：
- 需要配置 `MD2WECHAT_API_KEY`，可通过 `MD2WECHAT_BASE_URL` 或 `api.md2wechat_base_url` 覆盖 API 地址
- 支持默认发现的 53 个推荐高级排版语法；3 个兼容模块仅用于旧稿迁移
- 输出 HTML 最终，可直接用于微信草稿或发布

**响应示例**：
```json
{
  "success": true,
  "code": "CONVERT_COMPLETED",
  "data": {
    "mode": "api",
    "theme": "minimal-blue",
    "output_file": "/path/to/output.html",
    "image_count": 3,
    "preview": false,
    "upload": false,
    "draft": false
  }
}
```

#### 模式 B：AI 模式（轻量级，不需要 API）

```bash
md2wechat convert article.md --mode ai --output output.html
```

**特点**：
- 不需要 md2wechat API Key
- 不支持高级排版语法（`:::module` 被当作普通文本）
- 适合不使用高级排版模块的轻量场景

**注意**：API 模式失败时，CLI 不会自动切换到 AI 模式。Agent 也不应静默切换，因为 AI 模式不会渲染高级排版模块。只有用户接受失去高级排版效果时，才可以显式改用 `--mode ai`。

#### 模式 C：指定主题

```bash
md2wechat convert article.md --theme bytedance --output output.html
md2wechat convert article.md --theme minimal-blue --output output.html
md2wechat convert article.md --theme autumn-warm --output output.html
```

#### 模式 D：指定其他参数

```bash
md2wechat convert article.md \
  --title "自定义标题" \
  --author "自定义作者" \
  --digest "自定义摘要" \
  --output output.html
```

### 5.2 验证文章元数据（发布前必须）

```bash
md2wechat inspect article.md --json
```

响应包含：
- `data.readiness` — 发布就绪状态，包含旧布尔字段和 `targets/blockers`
- `data.checks` — 详细检查项（如 `TITLE_TOO_LONG`, `MISSING_COVER`）
- `data.metadata` — 解析出的标题、作者、摘要、图片数量

Agent 应优先检查 `data.readiness.targets` 来决定是否可以继续，`blocked` 时读取 `data.readiness.blockers` 中 `blocks` 包含对应目标的检查项。

当用户已有文章或初稿，但没有指定标题、封面或排版模块时，先运行 `md2wechat advise <article.md> --json`。把它当成可选增强建议，不要把它当成发布 readiness；发布前仍以 `inspect --json data.readiness.targets/blockers` 为准。

#### Inspect Readiness 契约

- `inspect --json` 的唯一机器契约是 `data.readiness`；不要创建或读取 `data.agent_readiness`、`data.target_readiness` 或 `ArticleState`。
- `convert_ready`、`upload_ready`、`draft_ready`、`preview_fidelity` 等 legacy booleans 会保留；新 Agent 决策使用 `targets/blockers`。
- `targets.convert` 只使用 `ready` / `blocked`；如果是 `blocked`，不要继续 `convert` / `upload` / `draft`。
- `targets.upload` 和 `targets.draft` 使用 `ready` / `blocked` / `not_requested`；如果用户请求 upload/draft 但目标是 `not_requested`，说明 inspect 运行时没有包含对应目标参数，应重新运行 `md2wechat inspect <article.md> --upload --json` 或 `md2wechat inspect <article.md> --draft --cover ... --json`，再根据 `ready` / `blocked` 决策。
- `targets.preview` 表示预览保真度，只使用 `ready` / `degraded`，不是发布 blocker。
- `blockers` 只包含会阻断至少一个目标的 error-level check；warn/info 仍只留在 `data.checks`。
- `inspect readiness` 覆盖 metadata length、API key、theme/mode compatibility、本地图片、WeChat config、cover inputs。
- 如果执行路径是 `convert --mode ai --custom-prompt ...`，inspect 必须带同样的 `--mode ai --custom-prompt ...`，因为 custom prompt 会替代主题作为 AI 排版提示来源。

### 5.3 预览文章

```bash
md2wechat preview article.md
md2wechat preview article.md --output /path/to/preview.html
```

在浏览器中打开预览，Agent 等待用户确认后才执行后续操作。

只有 `PREVIEW_READY` 才有本次调用可打开的 `output_file`，其内容与 converter HTML 字节一致。`PREVIEW_ACTION_REQUIRED` 的 `--json` 响应会返回 `data.inspect` 和 prompt；它与 `PREVIEW_FAILED` 都不会在本次调用中新建或覆盖预览 HTML。显式输出路径中的既有文件仍可能保留，但必须视为陈旧内容。

---

## 六、发布流程

### 6.1 检查图片是否需要上传

```bash
md2wechat inspect article.md --upload --json
```

检查 `data.readiness.targets.upload` 字段，确认是否满足上传条件；若为 `blocked`，读取 `data.readiness.blockers` 中 `blocks` 包含 `upload` 的检查项。

### 6.2 上传封面图

```bash
md2wechat generate_cover "article title" --output cover.jpg
md2wechat upload_image cover.jpg --json
```

响应包含 `media_id`，记录此 ID 用于后续发布。

### 6.3 创建草稿

```bash
md2wechat convert article.md --draft --cover ./cover.jpg --output draft.html
md2wechat create_draft draft.html --json
```

也可以一步到位：

```bash
md2wechat convert article.md \
  --draft \
  --cover ./cover.jpg \
  --output draft.html
```

响应包含 `media_id` 和 `publish_url`，用于后续发布。

### 6.4 创建图文消息（Image Post / 小绿书）

```bash
md2wechat create_image_post article.md --cover ./cover.jpg --json
```

适用于内容更适合以卡片形式展示的场景。

---

## 七、高级排版模块操作

### 7.1 验证排版语法

在转换前，始终验证 Markdown 文件中的排版语法：

```bash
md2wechat layout validate --file article.md --json
```

响应解读：
- `code: "LAYOUT_VALIDATED"` → 本地 catalog/schema 接受该语法；继续检查 warnings，远端渲染能力仍以真实 convert/conformance 为准
- `code: "LAYOUT_VALIDATE_HAS_ERRORS"` → 有错误，查看 `data.errors` 了解详情
- `data.errors` — 包含错误位置和修复建议

### 7.2 在文章中使用排版模块

在 Markdown 文件中使用 `:::` 语法（**仅 API 模式下生效**）：

```markdown
:::hero
title: 这是标题
subtitle: 这是副标题
:::

:::verdict
content: 这篇文章的核心观点
confidence: high
:::

:::callout level=warning
这是一个重要提示
:::

:::quote
content: 经典引用
author: 引用来源
:::

:::cta
title: 如果这篇对你有启发
action: 关注 / 分享 / 咨询
:::
```

### 7.3 查看模块完整规范

```bash
md2wechat layout show hero --json
md2wechat layout show verdict --json
md2wechat layout show callout --json
```

`body_format` 是模块正文语法契约；先按它写临时 Markdown，再运行 `layout validate`。

### 7.4 测试模块渲染

```bash
printf '%s\n' '测试内容' | md2wechat layout render callout --body-file - --json
md2wechat layout render cta --var title="立即订阅" --var note="持续更新" --json
```

---

## 八、异常处理与降级

### 8.1 API 模式不可用

**症状**：`convert` 命令返回 API key、base URL、网络或远程 API 错误。

**处理**：
```bash
# 1. 本地只读诊断
md2wechat doctor --json

# 2. 查看当前解析到的配置
md2wechat config show --format json

# 3. 如果用户接受不渲染高级排版模块，才显式改用 AI 模式
md2wechat convert article.md --mode ai --output output.html
```

不要静默降级。API 模式是高级排版模块的执行边界；切换到 AI 模式会改变输出能力。

### 8.2 Brand Profile 不存在

**症状**：`md2wechat brand show` 返回 `BRAND_NOT_FOUND`（当命令可用时）

**处理**：
```bash
# 1. 非阻塞提示一次
# Agent: 我没有看到 Brand Profile，这次会使用系统默认风格。
#
# 2. 继续当前任务
#
# 3. 只有用户明确要求设置时，才运行：
md2wechat brand init
```

### 8.3 排版语法错误

**症状**：`md2wechat layout validate --file article.md --json` 返回错误

**处理**：
```bash
# 1. 查看具体错误
md2wechat layout validate --file article.md --json | jq '.data.errors'

# 2. 查看对应模块的正确语法
md2wechat layout show <failing_module_name> --json

# 3. 查看实际渲染示例
md2wechat layout render <module_name> --var KEY=VALUE

# 4. 修复后重新验证
md2wechat layout validate --file article.md --json
```

### 8.4 发布失败

常见原因及处理：

| 错误 | 原因 | 处理方法 |
|------|------|---------|
| `MISSING_COVER` | 缺少封面图 | 上传封面图后重试 |
| `INVALID_TOKEN` | 微信凭证过期 | 检查 `config show --format json` 中的 token，重新授权 |
| `TITLE_TOO_LONG` | 标题超过 64 字符 | 修改标题后重试 |
| `IMAGE_REPLACEMENT_REQUIRES_UPLOAD` | 图片需要上传到微信 | 使用 `--upload` 参数或提前上传 |
| `ip_not_in_whitelist` | 微信 IP 白名单未配置 | 参考 [WECHAT-CREDENTIALS.md](WECHAT-CREDENTIALS.md) |

---

## 九、决策优先级链

当面临多个选择时，按以下顺序应用决策：

```
1. CLI flag（如 --theme default）
    ↓
2. 用户本轮明确指令
    ↓
3. Brand Profile（brand.md 中的自然语言偏好）
    ↓
4. 已解析配置（环境变量 + ~/.config/md2wechat/config.yaml）
    ↓
5. CLI discovery 可验证能力（themes/layout/providers/prompts）
    ↓
6. Agent 基于文章目标的保守默认选择
```

---

## 十、完整场景示例

### 场景 A：用户首次发文，无配置无 Brand Profile

```bash
# 1. 发现本次任务需要的能力
md2wechat doctor --json
md2wechat themes list --json
md2wechat layout list --json

# 2. 检查并初始化配置
md2wechat config show --format json
# 如果缺少关键配置，提示用户运行 md2wechat config init

# 3. Brand Profile 不存在时非阻塞提示一次，继续任务

# 4. 复制原文到临时 Markdown，插入 layout render 生成的模块后验证
md2wechat layout validate --file /tmp/md2wechat-format/<run-id>/article.formatted.md --json

# 5. 转换临时排版稿
md2wechat convert /tmp/md2wechat-format/<run-id>/article.formatted.md --output output.html

# 6. 预览临时排版稿
md2wechat preview /tmp/md2wechat-format/<run-id>/article.formatted.md

# 7. 检查发布就绪
md2wechat inspect /tmp/md2wechat-format/<run-id>/article.formatted.md --draft --cover ./cover.jpg --json
# 继续前确认 data.readiness.targets.draft 不是 blocked；如 blocked，读取 data.readiness.blockers

# 8. 创建草稿
md2wechat convert /tmp/md2wechat-format/<run-id>/article.formatted.md \
  --draft \
  --cover ./cover.jpg \
  --output draft.html
```

### 场景 B：用户有完整配置，快速发文

```bash
# 1. 验证排版语法
md2wechat layout validate --file /tmp/md2wechat-format/<run-id>/article.formatted.md --json

# 2. 转换（复用当前主题和配置）
md2wechat convert /tmp/md2wechat-format/<run-id>/article.formatted.md --output output.html

# 3. 预览
md2wechat preview /tmp/md2wechat-format/<run-id>/article.formatted.md

# 4. 一键创建草稿和发布
md2wechat convert /tmp/md2wechat-format/<run-id>/article.formatted.md \
  --draft \
  --cover ./cover.jpg \
  --theme bytedance
```

### 场景 C：调试排版问题

```bash
# 1. 验证语法并查看具体错误
md2wechat layout validate --file article.md --json

# 2. 查看失败模块的规范
md2wechat layout show <failing_module> --json

# 3. 查看模块渲染示例
md2wechat layout render <failing_module> --var KEY=VALUE

# 4. 修复文件中的语法错误

# 5. 重新验证
md2wechat layout validate --file article.md --json

# 6. 转换并预览
md2wechat convert article.md --mode api --output output.html
md2wechat preview article.md --output preview.html
```

### 场景 D：切换主题或 Provider

```bash
# 1. 查看可用主题
md2wechat themes list --json

# 2. 使用新主题转换
md2wechat convert article.md --theme minimal-blue --output output.html

# 3. 查看可用 Provider
md2wechat providers list --json

# 4. 更新配置或环境变量以切换 Provider
export IMAGE_PROVIDER=tuzi
export IMAGE_API_KEY=your_api_key

# 5. 重新转换
md2wechat convert article.md --output output.html
```

### 场景 E：处理 API 模式不可用

```bash
# 1. 本地诊断
md2wechat doctor --json

# 2. 查看当前配置
md2wechat config show --format json

# 3. 只有用户接受失去高级排版模块时，才显式改用 AI 模式
md2wechat convert article.md --mode ai --output output.html

# 4. 注意：AI 模式不支持高级排版模块
# 预期行为：:::module ... ::: 语法被当作普通文本
```

---

## 十一、常用命令快速参考

| 目标 | 命令 |
|------|------|
| 查看当前能力 | `md2wechat capabilities --json` |
| 发现内置 Agent skill | `md2wechat skills list --json` |
| 读取当前版本 SOP | `md2wechat skills read md2wechat --json` |
| 查看可用主题 | `md2wechat themes list --json` |
| 查看可用 Provider | `md2wechat providers list --json` |
| 查看 Prompt 模板 | `md2wechat prompts list --json` |
| 查看排版模块 | `md2wechat layout list --json` |
| 验证排版语法 | `md2wechat layout validate --file article.md --json` |
| 查看配置 | `md2wechat config show --format json` |
| 验证配置 | `md2wechat config validate` |
| 查看多公众号账号 | `md2wechat config wechat-accounts --json` |
| 转换文章（API） | `md2wechat convert article.md --output output.html` |
| 转换文章（AI） | `md2wechat convert article.md --mode ai --output output.html` |
| 检查发布就绪 | `md2wechat inspect article.md --draft --json` |
| 预览文章 | `md2wechat preview article.md` |
| 创建草稿 | `md2wechat convert article.md --draft --cover ./cover.jpg` |
| 查看模块规范 | `md2wechat layout show hero --json` |
| 测试模块渲染 | `printf '%s\n' 'test' \| md2wechat layout render callout --body-file - --json` |

---

## 十二、调试技巧

### 启用 JSON 输出用于脚本集成

几乎所有命令都支持 `--json` 参数，返回机器可读的 JSON envelope：

```bash
md2wechat convert article.md --json
md2wechat inspect article.md --json
md2wechat layout validate --file article.md --json
```

JSON envelope 格式（v1）：
```json
{
  "success": true,
  "code": "COMMAND_CODE",
  "message": "human-readable message",
  "schema_version": "v1",
  "status": "completed|action_required|failed",
  "retryable": false,
  "data": {},
  "error": ""
}
```

### 检查退出码

- `0` — 成功
- `1` — 通用错误
- `2` — 校验失败（使用 `--strict` 时可获得此码）

### 常见的 JSON 响应码

| code | 含义 | 处理方式 |
|------|------|---------|
| `CONVERT_COMPLETED` | 转换成功 | 继续发布流程 |
| `INSPECT_COMPLETED` | 检查完成 | 查看 `data.readiness.targets/blockers` |
| `PREVIEW_READY` | 预览可用 | 打开或检查 `data.output_file`；文件是原样 converter HTML |
| `PREVIEW_ACTION_REQUIRED` | 需要宿主 Agent 生成最终 HTML | 使用 `data.prompt` 和 `data.inspect`；`data.output_file` 为空，本次调用不新建或覆盖预览 HTML |
| `PREVIEW_FAILED` | 预览转换失败 | 报告错误；不要把显式输出路径中可能保留的陈旧文件当成本次结果 |
| `DOCTOR_COMPLETED` | 本地体检完成 | 查看 `data.overall` 和本地配置 `data.readiness.*` |
| `LAYOUT_VALIDATED` | 本地 catalog/schema 接受 | 检查 warnings；生产渲染仍需真实 convert/conformance 证明 |
| `LAYOUT_VALIDATE_HAS_ERRORS` | 排版有错误 | 查看 `data.errors` 修复 |
| `BRAND_NOT_FOUND` | Brand Profile 不存在 | 可选初始化 |
| `BRAND_SHOWN` | Brand Profile 已读取 | 使用其中配置 |
| `CONFIG_SHOWN` | 配置已读取 | 查看 `data` |

---

*最后更新：与 md2wechat v3.1.0 同步*
