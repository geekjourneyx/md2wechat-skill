<div align="center">

<h1>
  <img src="assets/favicon.ico" alt="md2wechat logo" width="28" />
  md2wechat
</h1>

<img src="assets/readme-cover.webp" alt="md2wechat" width="520" />

**用 Markdown 写公众号文章，像发朋友圈一样简单**

[![Go Version](https://img.shields.io/badge/Go-1.26.1+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![GitHub Release](https://img.shields.io/badge/download-latest-green)](https://github.com/geekjourneyx/md2wechat-skill/releases)
[![Claude Code](https://img.shields.io/badge/Claude%20Code-Skill-purple)](#-coding-agent-集成)
[![OpenClaw](https://img.shields.io/badge/OpenClaw-Compatible-00b0aa)](#-openclaw-支持)
[![zread](https://img.shields.io/badge/Ask_Zread-_.svg?style=flat&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff)](https://zread.ai/geekjourneyx/md2wechat-skill)

[快速开始](#-5分钟快速上手) • [Coding Agent](#-coding-agent-集成) • [Obsidian / Claudian](#-obsidian--claudian-支持) • [OpenClaw](#-openclaw-支持) • [功能介绍](#-核心功能) • [使用说明](#-使用方法) • [常见问题](#-常见问题)

---

## 立即开始

如果你第一次接触 `md2wechat`，先按这个顺序走：

- 如果你是 mac 用户，优先用 Homebrew 安装 CLI：`brew install geekjourneyx/tap/md2wechat`
- 如果你已经有稳定可用的 Go 环境，也可以选 `go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3`
- 想直接安装 CLI：看 [安装指南](docs/INSTALL.md)
- 想先看支持的主题 / 图片 prompt / provider：执行 discovery 命令
- 想在 Claude Code / Codex / OpenCode 等 Coding Agent 里使用：先安装 CLI，再执行 `npx skills add`
- 想在 Obsidian 的 Claudian 插件里使用：先安装 CLI，再执行 `npx skills add`，然后看 [Obsidian / Claudian 指南](docs/OBSIDIAN.md)

```bash
# 查看当前实例支持什么
md2wechat capabilities --json
md2wechat themes list --json
md2wechat prompts list --kind image --json
md2wechat providers list --json

# 先安装 CLI，再在 Coding Agent 中安装 skill
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash

# 安装 skill
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

### 直接发给 Agent

如果你不想自己组织命令，可以直接把下面的话发给 Claude Code / Codex / OpenCode 等 Agent：

```text
请帮我安装 md2wechat 并验证可用。按这个顺序执行：
1. 运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
2. 运行：npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
3. 运行：export PATH="$HOME/.local/bin:$PATH"
4. 运行：md2wechat version --json
5. 运行：md2wechat config init
6. 运行：md2wechat capabilities --json
如果某一步失败，请直接告诉我失败原因和下一步修复命令，不要省略命令。
```

如果你在 OpenClaw 里使用，可以直接发：

```text
请帮我安装 OpenClaw 版 md2wechat，并验证 CLI 和 skill 都可用。按这个顺序执行：
1. 运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install-openclaw.sh | bash
2. 运行：export PATH="$HOME/.local/bin:$PATH"
3. 运行：md2wechat version --json
4. 运行：md2wechat config init
5. 运行：md2wechat capabilities --json
6. 如果我已经装过 skill，再检查：~/.openclaw/skills/md2wechat/ 是否存在，并确认 `command -v md2wechat` 有输出
如果某一步失败，请直接告诉我失败原因和下一步修复命令，不要省略命令。
```

</div>

---

> ### 先看这里
> - `convert` 默认会返回 `AI request / prompt`，可以由 Claude Code / Codex / OpenAI 等外部模型接力完成 HTML 的生成
> - 主题、图片 prompt、provider 不要靠猜，先执行 `md2wechat themes list --json`、`md2wechat prompts list --kind image --json`、`md2wechat providers list --json`
> - 安装、配置和校验入口分别看：[安装指南](docs/INSTALL.md) / [配置指南](docs/CONFIG.md) / [能力发现](docs/DISCOVERY.md)

---

## 5分钟快速上手

### 第一步：安装工具

推荐先看 [安装指南](docs/INSTALL.md)。如果你只想快速开始：

- mac 用户：优先使用 Homebrew
- 已有 Go 环境：也可以使用 `go install`
- 其他环境：优先使用固定版本 `install.sh`

```bash
brew install geekjourneyx/tap/md2wechat
```

如果你不用 Homebrew，或者想走固定版本 release + checksum 路径，再用安装脚本：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
```

默认会安装到用户级目录：

- macOS / Linux: `~/.local/bin/md2wechat`
- Windows: 当前用户目录或 `C:\Program Files\md2wechat\md2wechat.exe`

如果脚本执行完后当前终端仍然提示 `command not found: md2wechat`，先执行：

```bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
```

Windows PowerShell：

```powershell
$env:MD2WECHAT_RELEASE_BASE_URL = "https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3"
iex ((New-Object System.Net.WebClient).DownloadString("$env:MD2WECHAT_RELEASE_BASE_URL/install.ps1"))
```

如果你主要在 Claude Code / Codex / OpenCode 等 Coding Agent 中使用，推荐先安装 CLI，再安装 skill。mac 用户优先用 Homebrew：

```bash
brew install geekjourneyx/tap/md2wechat
```

如果你已经有 Go 环境，也可以改成：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
```

然后安装 skill：

```bash
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果你要直接发给 Agent，建议把这句也带上：

```text
如果 install.sh 执行完后当前终端仍然提示 command not found: md2wechat，请继续执行：
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
```

手动下载 release 资产时，请同时校验 `checksums.txt`。主路径应始终指向固定版本的 release，不要使用 `latest` 或 `main` 作为安装入口。

### 第二步：配置微信（只需一次）

```bash
md2wechat config init
```

用记事本打开生成的配置文件（会显示路径），填入两个信息：

| 配置项 | 是什么 | 在哪获取 |
|--------|--------|----------|
| AppID | 公众号唯一标识 | [微信开发者平台](https://developers.weixin.qq.com/platform) → 开发接口管理 |
| Secret | API 密钥 | 同上，需要管理员权限 |

第一次接微信开发，建议直接看：

- [微信凭证与 IP 白名单指南](docs/WECHAT-CREDENTIALS.md)

里面单独讲清楚了：

- AppID / AppSecret 去哪里拿
- IP 白名单怎么配
- 为什么白名单没配时会报错
- 配好后怎么一步步验证

### 第三步：开始使用

```bash
# 1. 用 Markdown 写好文章（假设文件叫 article.md）

# 2. 预览效果
md2wechat convert article.md --preview

# 3. 发送到微信草稿箱
md2wechat convert article.md --draft --cover cover.jpg
```

> **小贴士**：第一次使用时，命令会自动引导你完成配置。

## 这是什么？

**md2wechat** 是一个让你的微信公众号写作更高效的神器。

> **一句话理解**：用 Markdown 写文章 → 一键转换 → 自动发到微信草稿箱

**适合谁用？**

| 你是 | 痛点 | md2wechat 帮你 |
|------|------|---------------|
| 内容创作者 | 微信编辑器太难用，排版花时间 | Markdown 写作，自动排版 |
| 产品经理 | 要发公告，但不会 HTML | 不用学代码，一行命令搞定 |
| 程序员 | 习惯 Markdown，讨厌微信编辑器 | 保持你的写作习惯 |
| AI 用户 | 用 AI 生成内容，但要手动复制粘贴 | AI 生成 → 微信草稿，无缝衔接 |

### 站点入口

- 国际站: [md2wechat.com](https://www.md2wechat.com/)
- 国内站: [md2wechat.cn](https://md2wechat.cn)
- GitHub 项目: [geekjourneyx/md2wechat-skill](https://github.com/geekjourneyx/md2wechat-skill)

### 关于作者

这个项目由 [geekjourneyx](https://github.com/geekjourneyx/geekjourneyx) 维护。

- 大厂后端工程师 / 独立开发者 / AI Builder / AI 科技领域博主
- 持续打磨面向 AI Agent 的 CLI、API 与自动化工作流
- 相关项目和长期方向可见 GitHub 主页: [github.com/geekjourneyx/geekjourneyx](https://github.com/geekjourneyx/geekjourneyx)

---

## 核心功能

```mermaid
flowchart LR
    A[用 Markdown 写文章] --> E[生成 AI request]

    E --> F[由 Claude 等继续生成 HTML]

    F --> G[预览效果]

    G --> H{满意吗}
    H -->|不满意| E
    H -->|满意| I[上传图片]
    I --> J[发送到微信草稿箱]
    J --> K[完成]

    classDef nodeA fill:#e3f2fd,stroke:#2196f3,color:#0d47a1
    classDef nodeE fill:#fff3e0,stroke:#ff9800,color:#e65100
    classDef nodeJ fill:#e8f5e9,stroke:#4caf50,color:#1b5e20
    classDef nodeK fill:#c8e6c9,stroke:#4caf50,color:#1b5e20

    class A nodeA
    class E nodeE
    class J nodeJ
    class K nodeK
```

### 五大核心功能

| 功能 | 命令 | 说明 | 适合谁 |
|------|------|------|--------|
| **Markdown 转换** | `convert` | 将 Markdown 转换为微信格式 HTML | 所有用户 |
| **风格写作** | `write` | 用创作者风格辅助写作，自动生成文章和封面提示词 | 写作小白、内容创作者 |
| **AI 去痕** | `humanize` | 去除 AI 生成痕迹，让文章听起来更自然、像人写的 | AI 写作用户 |
| **小绿书** | `create_image_post` | 创建图片消息（小绿书），最多 20 张图片 | 图片内容创作者 |
| **草稿推送** | `convert --draft` | 一键发送到微信草稿箱 | 需要频繁发布的用户 |

**`write` 与 `convert` 的区别：**

| 对比项 | `write` 命令 | `convert` 命令 |
|--------|-------------|---------------|
| **输入** | 一个想法/观点/片段 | 完整的 Markdown 文件 |
| **输出** | 结构化提示词（AI 处理后生成文章） | 微信格式 HTML |
| **用途** | 从零开始创作 | 格式转换已有内容 |
| **封面** | 自动生成封面提示词 | 需要手动指定封面图 |

**简单理解：**
- `write` = 帮你写文章（从想法到完整文章）
- `convert` = 帮你排版（从 Markdown 到微信格式）

### 完整工作流程

```mermaid
flowchart LR
    A1[Markdown 写作] --> A2[插入图片]
    A2 --> B3[生成 AI request]

    B3 --> B4[由 Claude 等继续生成 HTML]

    B4 --> C1[预览效果]
    C1 --> C2{满意吗}

    C2 -->|调整| A2
    C2 -->|OK| C3[上传图片]
    C3 --> C4[发送草稿]
    C4 --> C5[完成]

    classDef write fill:#e3f2fd,stroke:#2196f3,color:#0d47a1
    classDef ai fill:#fff3e0,stroke:#ff9800,color:#e65100
    classDef done fill:#e8f5e9,stroke:#4caf50,color:#1b5e20
    classDef success fill:#c8e6c9,stroke:#4caf50,color:#1b5e20

    class A1,A2 write
    class B3 ai
    class C4,C5 done
```

## 使用方法

### 基础命令

```bash
# 预览转换效果（不发送）
md2wechat convert article.md --preview

# 转换并保存为 HTML 文件
md2wechat convert article.md -o output.html

# 使用 AI 模式生成精美排版
md2wechat convert article.md --mode ai --theme autumn-warm --preview
```

### Agent 可发现命令

如果你是 Agent、自动化脚本，或者想先确认当前实例支持哪些能力，先执行这些命令：

```bash
# 总览当前 CLI 能力、默认模式、可用资源
md2wechat capabilities --json

# 查看图片 provider
md2wechat providers list --json

# 查看可用主题
md2wechat themes list --json

# 查看内置 Prompt Catalog
md2wechat prompts list --json
md2wechat prompts list --kind image --json
md2wechat prompts list --kind image --archetype cover --json
```

需要查看具体资源时，可继续用：

```bash
md2wechat providers show openrouter --json
md2wechat themes show autumn-warm --json
md2wechat prompts show cover-default --kind image --json
md2wechat prompts show cover-hero --kind image --archetype cover --tag hero --json
md2wechat prompts show infographic-victorian-engraving-banner --kind image --archetype infographic --tag victorian --json
```

Prompt 模板也可以直接渲染：

```bash
md2wechat prompts render cover-default \
  --kind image \
  --var article_title='从 0 到 1 做好公众号封面' \
  --var article_summary='一份关于封面图策略的实战清单' \
  --json
```

详细说明见 [docs/DISCOVERY.md](docs/DISCOVERY.md)。

图片 prompt 的底层统一归类为 `kind=image`。`cover` 和 `infographic` 是主要用途分组，不是两套独立系统。判断某个 preset 能否兼作封面或信息图时，优先看 `prompts show --json` 返回的 `primary_use_case`、`compatible_use_cases`、`recommended_aspect_ratios` 和 `default_aspect_ratio`。

### 风格写作

```bash
# 查看所有可用写作风格
md2wechat write --list

# 用 Dan Koe 风格写文章（交互模式）
md2wechat write

# 用指定风格写文章，从观点生成
md2wechat write --style dan-koe

# 润色现有文章
md2wechat write --style dan-koe --input-type fragment article.md

# 生成匹配的封面提示词
md2wechat write --style dan-koe --cover-only

# 同时生成文章和封面
md2wechat write --style dan-koe --cover
```

**写作风格说明：**

| 风格 | 特点 | 适合内容 |
|------|------|----------|
| **Dan Koe** | 深刻但不晦涩，犀利但不刻薄，有哲学深度但接地气 | 个人成长、观点文章、评论 |

**如何添加自定义风格：**

在 `writers/` 目录下创建 YAML 文件即可，格式参考 `writers/dan-koe.yaml`。

### Prompt Catalog

除了 `themes/` 和 `writers/`，项目现在还提供内置 Prompt Catalog，用于承载：

- `humanize` 的强度模板
- `write` 的润色模板
- 后续扩展的图片 archetype，例如封面图、信息图、配图

当前内置 prompt kind：

- `humanizer`
- `refine`
- `image`

Prompt 覆盖顺序：

- `MD2WECHAT_PROMPTS_DIR`
- `./prompts`
- `~/.config/md2wechat/prompts`
- 内置 prompt 资产

这意味着纯二进制安装也能用默认 prompt，而用户和平台仍然可以覆盖。

### 风格写作工作原理

```
┌─────────────────────────────────────────────────────────────────────┐
│                      风格写作 (write 命令)                           │
│                                                                       │
│  你只需要提供一个想法 → AI 自动生成符合特定创作者风格的文章            │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

**核心概念：**

| 概念 | 说明 | 示例 |
|------|------|------|
| **写作风格** | 特定创作者的写作 DNA，包括语气、结构、用词习惯 | Dan Koe：深刻犀利、接地气 |
| **输入类型** | 你提供的内容类型 | 观点、片段、大纲、标题 |
| **AI 模式** | 返回结构化提示词，由 Claude 等大模型生成内容 | 默认模式 |
| **封面提示词** | 根据文章内容自动生成的配图提示 | 匹配写作风格 |

**完整工作流程：**

```mermaid
flowchart LR
    A[你提供想法] --> B{选择写作风格}
    B --> C[构建结构化提示词]

    C --> D{AI 处理}
    D --> E[生成完整文章]

    E --> F{需要封面?}
    F -->|是| G[生成封面提示词]
    F -->|否| H[输出文章]

    G --> I[用 AI 生成封面图]
    I --> J[上传到微信素材库]
    H --> K[转换为微信格式]
    J --> K
    K --> L[发送到草稿箱]

    classDef idea fill:#e3f2fd,stroke:#2196f3,color:#0d47a1
    classDef creator fill:#fff3e0,stroke:#ff9800,color:#e65100
    classDef ai fill:#f3e5f5,stroke:#9c27b0,color:#4a148c
    classDef output fill:#e8f5e9,stroke:#4caf50,color:#1b5e20

    class A idea
    class B,C creator
    class D,E,G,I ai
    class H,J,K,L output
```

**输入类型说明：**

| 输入类型 | 说明 | 示例 |
|----------|------|------|
| `idea` | 一个观点或想法 | "我觉得自律是个伪命题" |
| `fragment` | 内容片段，需要润色扩展 | 现有的草稿或未完成的文章 |
| `outline` | 文章大纲 | 有结构，需要填充内容 |
| `title` | 仅标题，围绕标题写作 | "自律是个谎言" |

**输出说明：**

```json
{
  "success": true,
  "code": "WRITE_AI_REQUEST_READY",
  "schema_version": "v1",
  "status": "action_required",
  "retryable": false,
  "data": {
    "mode": "ai",
    "action": "ai_write_request",
    "style": "Dan Koe",
    "prompt": "结构化的写作提示词...",
    "cover_prompt": "封面提示词..."
  }
}
```

> **重要说明**：`write` 命令默认使用 **AI 模式**，返回的是结构化提示词，需要由 Claude 等 AI 大模型处理才能生成最终文章。在 Claude Code 中使用时，这个流程是自动的。

**从想法到发布的完整流程：**

```mermaid
flowchart TB
    subgraph Phase1["阶段1：创作"]
        A1[你的想法] --> A2[选择写作风格]
        A2 --> A3[AI 生成文章]
        A3 --> A4[生成封面提示词]
    end

    subgraph Phase2["🎨 阶段2：制作"]
        B1[AI 生成封面图] --> B2[上传到微信素材库]
        A4 --> B1
        A3 --> B3[保存为 Markdown]
    end

    subgraph Phase3["📤 阶段3：发布"]
        C1[转换为微信格式] --> C2[发送到草稿箱]
        B2 --> C1
        B3 --> C1
        C2 --> C3[在微信编辑器中微调]
        C3 --> C4[发布！]
    end

    classDef phase1 fill:#e3f2fd,stroke:#2196f3,color:#0d47a1
    classDef phase2 fill:#fff3e0,stroke:#ff9800,color:#e65100
    classDef phase3 fill:#e8f5e9,stroke:#4caf50,color:#1b5e20

    class A1,A2,A3,A4 phase1
    class B1,B2,B3 phase2
    class C1,C2,C3,C4 phase3
```

**一条命令完成全流程（在 Claude Code 中）：**

```
"用 Dan Koe 风格写一篇关于 AI 时代程序员怎么搞钱的文章，生成封面，并发送到微信草稿箱"
```

Claude 会自动：
1. 调用 `write` 命令生成文章和封面提示词
2. 用 AI 生成封面图并上传
3. 转换为微信格式
4. 发送到草稿箱

### AI 写作去痕

> **致谢**：AI 写作去痕功能基于 [Humanizer-zh](https://github.com/op7418/Humanizer-zh) 项目，感谢作者 **臧师傅** 的分享。

**什么是 AI 去痕？**

AI 生成的文章往往有明显的"AI 味"——公式化的表达、重复的结构、浮夸的词汇。AI 去痕功能可以识别并去除这些痕迹，让文章听起来更像真人写的。

**能检测哪些 AI 痕迹？**

| 类别 | 检测模式 | 示例 |
|------|----------|------|
| **内容模式** | 过度强调意义、宣传语言、模糊归因 | "标志着"、"凸显了"、"行业报告显示" |
| **语言模式** | AI 词汇、否定排比、三段式 | "此外"、"深入探讨"、"不仅…而且…" |
| **风格模式** | 破折号过度、粗体滥用、表情符号 | 过度使用 `——`、大量 `**粗体**` |
| **填充词** | 填充短语、过度限定、通用结论 | "为了实现这一目标"、"未来一片光明" |
| **协作痕迹** | 对话式填充、知识截止免责声明 | "希望这对您有帮助"、"截至 2024 年" |

**三种处理强度：**

| 强度 | 命令 | 适合场景 |
|------|------|----------|
| `gentle` | `--intensity gentle` | 文章已经比较自然，只处理明显问题 |
| `medium` | 默认 | 大多数场景，平衡处理 |
| `aggressive` | `--intensity aggressive` | AI 味很重的文章，深度改写 |

**使用方法：**

```bash
# 独立使用：去除文章的 AI 痕迹
md2wechat humanize article.md

# 指定强度
md2wechat humanize article.md --intensity aggressive

# 显示修改对比和质量评分
md2wechat humanize article.md --show-changes

# 输出到文件
md2wechat humanize article.md -o output.md
```

**与写作风格组合使用：**

```bash
# 写作 + 去痕：生成文章后自动去除 AI 痕迹
md2wechat write --style dan-koe --humanize

# 指定去痕强度
md2wechat write --style dan-koe --humanize --humanize-intensity aggressive
```

> **风格优先原则**：当与写作风格组合时，会保留风格的核心特征（如 Dan Koe 的破折号），只去除无意的 AI 痕迹。

**AI 去痕工作流程：**

```mermaid
flowchart LR
    A[AI 生成的文章] --> B[识别 AI 痕迹模式]
    B --> C{检测到问题}

    C --> D[过度强调意义]
    C --> E[AI 词汇]
    C --> F[三段式结构]
    C --> G[填充短语]
    C --> H[协作痕迹]

    D --> I[重写为自然表达]
    E --> I
    F --> J[打破公式结构]
    G --> I
    H --> K[删除客套话]

    I --> L[注入真实观点]
    J --> L
    K --> L

    L --> M[变化句子节奏]
    M --> N[人性化后的文章]

    N --> O{需要评分?}
    O -->|是| P[5维度质量评分]
    O -->|否| N

    P --> Q[输出结果]

    classDef ai fill:#ffebee,stroke:#f44336,color:#c62828
    classDef problem fill:#fff3e0,stroke:#ff9800,color:#e65100
    classDef fix fill:#e3f2fd,stroke:#2196f3,color:#0d47a1
    classDef result fill:#e8f5e9,stroke:#4caf50,color:#1b5e20

    class A ai
    class B,C,D,E,F,G,H problem
    class I,J,K,L,M fix
    class N,P,Q result
```

**质量评分系统：**

去除 AI 痕迹后，会给出 5 维度评分（总分 50）：

| 维度 | 说明 | 评分标准 |
|------|------|----------|
| **直接性** | 是否直截了当 | 10 分：直击要点；1 分：充满铺垫 |
| **节奏** | 句子长度是否变化 | 10 分：长短交错；1 分：机械重复 |
| **信任度** | 是否尊重读者 | 10 分：简洁明了；1 分：过度解释 |
| **真实性** | 听起来像真人吗 | 10 分：自然流畅；1 分：机械生硬 |
| **精炼度** | 有无冗余内容 | 10 分：无废话；1 分：大量填充 |

**评级标准：**
- **45-50 分**：优秀，已去除 AI 痕迹
- **35-44 分**：良好，仍有改进空间
- **低于 35 分**：需要重新修订

**去痕效果对比：**

| 原文（AI 味） | 去痕后 |
|--------------|--------|
| 在当今快速发展的科技时代，人工智能技术正以前所未有的速度改变着我们的生活方式 | 这几年，AI 变化太快了 |
| 人工智能的重要性不言而喻。它不仅标志着技术进步的新里程碑 | AI 挺重要的，但不是那种玄乎的重要 |
| 此外，AI 技术还在改善我们的日常生活质量 | 顺手提一句，AI 确实让生活方便了不少 |
| 尽管存在挑战，但人工智能的未来依然光明 | 未来会怎样不好说，有好有坏吧 |
| 希望这对您有帮助 | —— |

**自然语言使用（在 Claude Code 中）：**

```
"去除这篇文章的 AI 痕迹"
"让这篇文章听起来更像人写的"
"用温和强度处理这篇文章"
"用 Dan Koe 风格写一篇文章，然后去除 AI 痕迹"
```

### 完整发布流程

```bash
# 一步到位：转换 + 上传图片 + 发送草稿
md2wechat convert article.md --draft --cover cover.jpg

# 流程说明：
# 1. 将 Markdown 转换为微信格式 HTML
# 2. 上传封面图片到微信素材库
# 3. 创建草稿并推送到微信后台
```

### 主题选择

| 主题名 | 命令 | 风格 | 适合内容 |
|--------|------|------|----------|
| **秋日暖光** | `--theme autumn-warm` | 温暖橙色调 | 情感故事、生活随笔 |
| **春日清新** | `--theme spring-fresh` | 清新绿色调 | 旅行日记、自然主题 |
| **深海静谧** | `--theme ocean-calm` | 专业蓝色调 | 技术文章、商业分析 |

### 其他主题分类

精确主题清单以运行时 discovery 输出为准：

```bash
md2wechat themes list --json
```

下面内容主要用于帮助理解主题分组和风格差异。

#### 主题预览
**完整主题预览**: [https://md2wechat.app/theme-gallery](https://md2wechat.app/theme-gallery)

#### 主题分类

**基础主题（6 个）** - v1.0 内置

| 主题 | 风格 | 适合 |
|------|------|------|
| `default` | 微信经典，温暖舒适 | 通用内容 |
| `bytedance` | 科技现代，简洁利落 | 科技资讯 |
| `apple` | 视觉渐变，精致优雅 | 产品评测 |
| `sports` | 活力动感，充满能量 | 体育健康 |
| `chinese` | 古典雅致，书卷气息 | 文化文章 |
| `cyber` | 未来科技，霓虹光影 | 前沿科技 |

**Minimal 系列（8 个）** - 干净克制，纯色文字无装饰

| 颜色 | 主题 | 命令 |
|------|------|------|
| 金色 | minimal-gold | `--theme minimal-gold` |
| 绿色 | minimal-green | `--theme minimal-green` |
| 蓝色 | minimal-blue | `--theme minimal-blue` |
| 橙色 | minimal-orange | `--theme minimal-orange` |
| 红色 | minimal-red | `--theme minimal-red` |
| 藏青 | minimal-navy | `--theme minimal-navy` |
| 灰色 | minimal-gray | `--theme minimal-gray` |
| 天蓝 | minimal-sky | `--theme minimal-sky` |

**Focus 系列（8 个）** - 居中对称，标题上下双横线

| 颜色 | 主题 | 命令 |
|------|------|------|
| 金色 | focus-gold | `--theme focus-gold` |
| 绿色 | focus-green | `--theme focus-green` |
| 蓝色 | focus-blue | `--theme focus-blue` |
| 橙色 | focus-orange | `--theme focus-orange` |
| 红色 | focus-red | `--theme focus-red` |
| 藏青 | focus-navy | `--theme focus-navy` |
| 灰色 | focus-gray | `--theme focus-gray` |
| 天蓝 | focus-sky | `--theme focus-sky` |

**Elegant 系列（8 个）** - 层次丰富，左边框递减 + 渐变背景

| 颜色 | 主题 | 命令 |
|------|------|------|
| 金色 | elegant-gold | `--theme elegant-gold` |
| 绿色 | elegant-green | `--theme elegant-green` |
| 蓝色 | elegant-blue | `--theme elegant-blue` |
| 橙色 | elegant-orange | `--theme elegant-orange` |
| 红色 | elegant-red | `--theme elegant-red` |
| 藏青 | elegant-navy | `--theme elegant-navy` |
| 灰色 | elegant-gray | `--theme elegant-gray` |
| 天蓝 | elegant-sky | `--theme elegant-sky` |

**Bold 系列（8 个）** - 视觉冲击，标题满底色 + 圆角投影

| 颜色 | 主题 | 命令 |
|------|------|------|
| 金色 | bold-gold | `--theme bold-gold` |
| 绿色 | bold-green | `--theme bold-green` |
| 蓝色 | bold-blue | `--theme bold-blue` |
| 橙色 | bold-orange | `--theme bold-orange` |
| 红色 | bold-red | `--theme bold-red` |
| 藏青 | bold-navy | `--theme bold-navy` |
| 灰色 | bold-gray | `--theme bold-gray` |
| 天蓝 | bold-sky | `--theme bold-sky` |

#### 主题命名规则

```
<系列>-<颜色>

系列:
  minimal  - 干净克制
  focus    - 居中对称
  elegant  - 层次丰富
  bold     - 视觉冲击

颜色: gold, green, blue, orange, red, navy, gray, sky
```

#### 使用示例

```bash
# 使用 Elegant 金色主题
md2wechat convert article.md --theme elegant-gold --preview

# 使用 Minimal 蓝色主题
md2wechat convert article.md --theme minimal-blue --draft --cover cover.jpg

# 使用 Bold 红色主题（视觉冲击强）
md2wechat convert article.md --theme bold-red
```

> **提示**: 某些 API 主题能力依赖当前 API 域名与服务版本，先看 `md2wechat themes list --json` 的实际输出。

#### 背景类型选择

除了主题，你还可以自定义背景样式：

| 背景类型 | 命令 | 效果 | 适合场景 |
|----------|------|------|----------|
| `default` | `--background-type default` | 默认背景（纯色或渐变） | 通用内容 |
| `grid` | `--background-type grid` | 网格纹理背景 | 技术文档、笔记类 |
| `none` | `--background-type none` | 无背景（透明） | 嵌入式内容 |

**使用示例：**

```bash
# 使用网格背景
md2wechat convert article.md --theme elegant-gold --background-type grid

# 使用无背景（适合复制到其他编辑器）
md2wechat convert article.md --theme minimal-blue --background-type none

# 组合使用
md2wechat convert article.md --theme focus-green --background-type grid --draft --cover cover.jpg
```

**配置文件设置：**

```yaml
# ~/.config/md2wechat/config.yaml
api:
  background_type: grid  # default/grid/none
```

### 图片处理

```bash
# 上传单张图片到微信素材库
md2wechat upload_image photo.jpg

# 下载网络图片并上传
md2wechat download_and_upload https://example.com/image.jpg

# AI 生成图片并上传（需要配置 IMAGE_API_KEY）
md2wechat generate_image "A cute cat sitting on a windowsill"

# 用内置封面 preset 生成封面图
md2wechat generate_cover --article article.md

# 用内置信息图 preset 生成信息图
md2wechat generate_infographic --article article.md --preset infographic-comparison

# 生成暗黑票券风中文信息图（推荐 21:9）
md2wechat generate_infographic --article article.md --preset infographic-dark-ticket-cn --aspect 21:9

# 生成手绘 sketchnote 信息图
md2wechat generate_infographic --article article.md --preset infographic-handdrawn-sketchnote

# 生成苹果发布会风格信息图（默认 3:4，也可兼作封面）
md2wechat generate_infographic --article article.md --preset infographic-apple-keynote-premium

# 生成维多利亚黑白版画横幅信息图（也可兼作封面，推荐 21:9）
md2wechat generate_infographic --article article.md --preset infographic-victorian-engraving-banner --aspect 21:9

# 生成 16:9 比例的封面图（推荐，适配公众号封面）
md2wechat generate_image --preset cover-hero --article article.md --size 2560x1440

# 单次覆盖本次调用使用的图片模型
md2wechat generate_infographic --article article.md --preset infographic-comparison --model gemini-3-pro-image-preview
```

> **公众号封面图建议**：使用 16:9 横向比例（2560x1440）作为文章封面，在微信 feed 流和文章列表中显示效果更好。方形图片（2048x2048）在预览时会被裁剪。

#### AI 图片生成服务配置

支持多种 AI 图片生成服务：

| 服务 | 配置值 | 说明 | 获取方式 |
|------|--------|------|----------|
| **ModelScope** | `modelscope` 或 `ms` | 阿里 ModelScope，免费额度 | [modelscope.cn](https://modelscope.cn/my/myaccesstoken) |
| **TuZi** | `tuzi` | 国产 API，稳定快速 | [tu-zi.com](https://api.tu-zi.com) |
| **OpenAI** | `openai` | 官方 OpenAI | [platform.openai.com](https://platform.openai.com) |
| **OpenRouter** | `openrouter` 或 `or` | 多模型聚合接口，支持 Gemini / Flux 等 | [openrouter.ai](https://openrouter.ai) |
| **Google Gemini** | `gemini` 或 `google` | 官方 Gemini 图片生成 API | [ai.google.dev](https://ai.google.dev/gemini-api/docs/image-generation) |

**配置方式**（环境变量或配置文件）：

```bash
# 使用 ModelScope（推荐，有免费额度）
export IMAGE_PROVIDER=modelscope
export IMAGE_API_KEY=ms-your-token-here
export IMAGE_API_BASE=https://api-inference.modelscope.cn
export IMAGE_MODEL=Tongyi-MAI/Z-Image-Turbo
```

```yaml
# config.yaml
api:
  image_provider: modelscope
  image_key: ms-your-token-here
  image_base_url: https://api-inference.modelscope.cn
  image_model: Tongyi-MAI/Z-Image-Turbo
  image_size: 1024x1024
```

> `--model` 可覆盖当前命令的图片模型；未传时，优先使用 `IMAGE_MODEL`，再回退到 `api.image_model`，最后才是 provider 默认模型。

**ModelScope 特点**：
- 有免费额度，适合测试
- 国内服务，访问稳定
- 默认模型 `Tongyi-MAI/Z-Image-Turbo` 生成速度快
- 使用异步 API（task_id + 轮询），约 10-30 秒完成

#### Markdown 中生成图片

在 Markdown 中使用特殊语法生成图片：

```markdown
![产品概念图](__generate:现代智能家居设备，白色简约设计，LED指示灯__)
```

**语法格式：** `![描述](__generate:提示词__)`

- 支持中文和英文提示词
- 生成的图片会自动上传到微信素材库
- 需要配置图片生成服务（详见 [图片服务配置文档](docs/IMAGE_PROVISIONERS.md)）

**在 Claude Code 中使用自然语言：**
```
"帮我在文章开头生成一张产品概念图"
"在第三段后添加一张对比图"
"生成一张可爱的猫"（独立生成，不关联文章）
```

---

## AI 模式详解

### 什么是 AI 模式？

**AI 模式**当前会生成用于外部大模型继续处理的排版请求，而不是直接在本地返回最终 HTML。

```
┌─────────────────────────────────────────────────────────────┐
│                     AI 模式工作流程                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   1. 你用 Markdown 写文章                                    │
│              ↓                                               │
│   2. md2wechat 提取文章结构                                  │
│              ↓                                               │
│   3. 构建专业的排版提示词 (Prompt)                           │
│              ↓                                               │
│   4. 输出 AI request / Prompt                              │
│              ↓                                               │
│   5. 在 Claude Code 等环境中继续生成 HTML                    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 在 Claude Code 中使用 AI 模式

如果你使用 **Claude Code**，AI 模式生成的 request / prompt 可以继续由 Claude 自动接力处理：

```bash
# 在 Claude Code 中直接运行
md2wechat convert article.md --mode ai --theme autumn-warm
```

---

## 配置说明

### 配置文件位置

```
~/.config/md2wechat/config.yaml    # 全局配置（推荐）
```

生成示例配置：

```bash
md2wechat config init
```

查看当前实际生效配置：

```bash
md2wechat config show --format json
```

### 配置项说明

```yaml
# 微信公众号配置（必需）
wechat:
  appid: "你的AppID"
  secret: "你的Secret"

# API/通用 配置
api:
  convert_mode: "ai"                        # 默认模式
  default_theme: "default"                  # 默认主题
  http_timeout: 30                          # 超时时间（秒）

# 图片处理配置
image:
  compress: true           # 自动压缩大图
  max_width: 1920         # 最大宽度
  max_size_mb: 5          # 最大文件大小（MB）
```

默认主题和默认写作风格已经内置在二进制里。
如果你要覆盖它们，优先级从高到低是：

- `MD2WECHAT_THEMES_DIR` / `MD2WECHAT_WRITERS_DIR`
- 当前项目目录下的 `themes/` / `writers/`
- `~/.config/md2wechat/themes/` / `~/.config/md2wechat/writers/`
- 二进制内置默认资产

更完整的保姆级说明见：

- [配置指南](docs/CONFIG.md)
- [示例配置](docs/examples/config.yaml.example)
- [真实烟雾测试记录](docs/SMOKE.md)

---

## 📁 项目结构

```
md2wechat-skill/
├── cmd/                    # 命令行工具
│   └── md2wechat/         # 主程序
├── internal/              # 核心功能模块
│   ├── converter/        # 转换器（API/AI）
│   ├── draft/            # 草稿服务
│   ├── image/            # 图片处理
│   ├── wechat/           # 微信 API 封装
│   ├── writer/           # 写作助手
│   ├── humanizer/        # AI 去痕
│   └── config/           # 配置管理
├── writers/              # 写作风格配置
│   ├── dan-koe.yaml      # Dan Koe 风格
│   └── README.md         # 自定义风格指南
├── docs/                 # 详细文档
│   ├── USAGE.md          # 使用教程
│   ├── FAQ.md            # 常见问题
│   └── TROUBLESHOOTING.md # 故障排查
├── examples/             # 示例文章
├── scripts/              # 安装脚本
└── bin/                  # 编译好的二进制文件
```

---

## 高级安装

### 方式一：Homebrew（macOS 优先推荐）

如果你在 macOS 上，并且已经装了 Homebrew，优先执行：

```bash
brew install geekjourneyx/tap/md2wechat
```

升级：

```bash
brew upgrade geekjourneyx/tap/md2wechat
```

### 方式二：一键安装脚本（其他环境推荐）

**Mac/Linux：**
```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
```

**Windows PowerShell：**
```powershell
$env:MD2WECHAT_RELEASE_BASE_URL = "https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3"
iex ((New-Object System.Net.WebClient).DownloadString("$env:MD2WECHAT_RELEASE_BASE_URL/install.ps1"))
```

> 脚本会下载 release 资产并验证 `checksums.txt`，这是当前推荐路径。

### 方式二：手动下载 release 资产

如果你需要固定版本或离线安装，请按 [安装指南](docs/INSTALL.md) 的 release + checksum 步骤操作。

### 方式三：从源码编译

```bash
git clone https://github.com/geekjourneyx/md2wechat-skill.git
cd md2wechat-skill
make build
```

---

## Coding Agent 集成

md2wechat 提供了面向 **Claude Code / Codex / OpenCode** 的 skill 和仓库安装方式，让你在 Coding Agent 中直接使用自然语言完成排版、图片生成和草稿推送。

### 安装方式

#### 方式一：`npx skills add`（推荐，通用）

适合 Claude Code / Codex / OpenCode 等支持 `skills` 安装协议的环境：

```bash
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

安装后，直接在 Agent 中调用 `md2wechat` skill 即可。`npx skills add` 只安装 skill，不会替你自动安装 CLI runtime。

当前 coding-agent skill 的 runtime 逻辑是：

- 直接依赖 `PATH` 里已安装好的 `md2wechat`
- skill 本身不再承担 runtime 查找、缓存 bootstrap 或执行时下载
- 如果 `md2wechat` 不在 `PATH` 里，先安装 CLI，再继续使用 skill

skill 不再在正常执行路径中自动下载并执行远程二进制。

推荐直接发给 Coding Agent 的对话脚本：

```text
请先安装 md2wechat CLI，再安装 md2wechat skill，并验证版本和能力发现都正常。
执行：
1. 如果我是 mac 用户，先运行：brew install geekjourneyx/tap/md2wechat
2. 如果我已经有稳定可用的 Go 环境，也可以改成：go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
3. 如果以上两种都不适合，再运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
4. 运行：npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
5. 如果我是通过 install.sh 安装的，再运行：export PATH="$HOME/.local/bin:$PATH"
6. 运行：md2wechat version --json
7. 运行：md2wechat capabilities --json
8. 运行：md2wechat config init
如果失败，请继续排查，不要只停止在报错信息。
```

#### 方式二：Claude Code Plugin Marketplace

在 Claude Code 中运行以下命令：

```bash
# 添加插件市场
/plugin marketplace add geekjourneyx/md2wechat-skill

# 安装插件
/plugin install md2wechat@geekjourneyx-md2wechat-skill
```

安装后，直接在 Claude Code 中对话即可使用：

```
请用秋日暖光主题将 article.md 转换为微信公众号格式
```

#### 方式三：Claudian / Obsidian

如果你在 Obsidian 里使用 [Claudian](https://github.com/YishenTu/claudian)，也可以直接复用同一套 skill。

Claudian 官方说明里已经明确：
- 它兼容 Claude Code skill 格式
- 会从 `~/.claude/skills/` 或 `{vault}/.claude/skills/` 发现 skill
- 支持 slash commands 和 skills

所以对 `md2wechat` 来说，最稳的路径是：

```bash
brew install geekjourneyx/tap/md2wechat
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果你不用 Homebrew，再改用固定版本安装脚本：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

然后在 Claudian 里：
- 直接输入 `/md2wechat`
- 或直接让 Agent 调用 `md2wechat` skill

如果 Obsidian 图形界面里仍然找不到 `md2wechat`，优先去 Claudian：
- `Settings -> Environment -> Custom variables`
- 添加：
  - `PATH=/Users/你的用户名/.local/bin:...`

保姆级说明见：
- [Obsidian / Claudian 指南](docs/OBSIDIAN.md)

#### 方式四：项目内使用

克隆项目后，Skill 自动可用：

```bash
git clone https://github.com/geekjourneyx/md2wechat-skill.git
cd md2wechat-skill
# 在 Claude Code 中直接使用
```

#### 方式五：全局安装

将 Skill 复制到全局目录：

```bash
# 复制到全局技能目录
cp -r skills/md2wechat ~/.claude/skills/
```

#### 方式六：创建符号链接

```bash
ln -s /path/to/md2wechat-skill/skills/md2wechat ~/.claude/skills/md2wechat
```

### 项目结构

```
md2wechat-skill/
├── .claude-plugin/        # 插件清单
│   └── marketplace.json   # Plugin Marketplace 配置
├── skills/                # coding-agent skill 目录（Claude Code / Codex / OpenCode）
│   └── md2wechat/
│       ├── SKILL.md       # 技能定义
│       ├── references/    # 参考文档
│       │   ├── themes.md      # 主题指南
│       │   ├── html-guide.md  # HTML 规范
│       │   ├── image-syntax.md # 图片语法
│       │   └── wechat-api.md  # API 参考
├── platforms/             # 平台适配层
│   └── openclaw/
│       └── md2wechat/     # OpenClaw 专用 skill 包
│           └── SKILL.md
├── scripts/               # 安装脚本
│   ├── install.sh         # CLI 安装脚本
│   └── install-openclaw.sh # OpenClaw 安装脚本
└── themes/                # AI 主题配置
    ├── autumn-warm.yaml
    ├── spring-fresh.yaml
    └── ocean-calm.yaml
```

---

## Obsidian / Claudian 支持

如果你在 Obsidian 里使用 [Claudian](https://github.com/YishenTu/claudian)，`md2wechat` 现在也能直接接进去。

Claudian 官方 README 说明它：
- 是一个把 Claude Code 嵌入 Obsidian 的插件
- 兼容 Claude Code skill 格式
- 会从 `~/.claude/skills/` 或 `{vault}/.claude/skills/` 发现 skills
- 支持 slash commands

这意味着：
- 先安装 `md2wechat` CLI
- 再执行 `npx skills add ...`
- 之后你可以在 Claudian 里直接用 `/md2wechat` 或自然语言调用 skill

推荐直接发给 Claudian / Obsidian 用户的脚本：

```text
请帮我在当前电脑上安装 md2wechat，并让 Claudian 可以在 Obsidian 里通过 /md2wechat 使用它。按这个顺序执行：
1. 如果我是 mac 用户，先运行：brew install geekjourneyx/tap/md2wechat
2. 如果我已经有稳定可用的 Go 环境，也可以改成：go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
3. 如果以上两种都不适合，再运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
4. 如果我是通过 install.sh 安装的，再运行：export PATH="$HOME/.local/bin:$PATH"
5. 运行：md2wechat version --json
6. 运行：npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
7. 如果我在 Obsidian 里仍然找不到 md2wechat，请提醒我去 Claudian 的 Settings -> Environment -> Custom variables，把 PATH 补上 ~/.local/bin
8. 最后告诉我如何在 Claudian 里输入 /md2wechat 或直接调用 md2wechat skill
如果某一步失败，请继续排查，不要只返回报错原文。
```

适合谁：
- 已经在 Obsidian 里用 Claudian 写作
- 想在笔记库里直接把 Markdown 转公众号格式
- 想在 Obsidian 里直接生成封面图、信息图或推送草稿

单独文档：
- [Obsidian / Claudian 指南](docs/OBSIDIAN.md)

---

## OpenClaw 支持

md2wechat 现已支持 [OpenClaw](https://openclaw.ai/) 平台！

OpenClaw 使用独立的 `platforms/openclaw/md2wechat/` skill 包，和面向 Claude Code / Codex / OpenCode 的 `skills/md2wechat/` 分开维护。OpenClaw 现在也直接依赖已安装到环境路径里的 `md2wechat` CLI，不再保留 skill 内部 runtime wrapper。

OpenClaw 用户也可以直接从技能页面进入：

- [ClawHub 页面：clawhub.ai/geekjourneyx/md2wechat](https://clawhub.ai/geekjourneyx/md2wechat)

### 什么是 OpenClaw？

[OpenClaw](https://openclaw.ai/) 是一个开源的 AI Agent 平台，**在你的设备上运行**，通过你已经在用的聊天应用（WhatsApp、Telegram、Discord、Slack）来操控 AI 助手。

> **The AI that actually does things.** — 你的助手，你的设备，你的规则。

### 安装方式

#### 方式一：ClawHub 安装（仅安装 skill 壳）

```bash
# 安装 OpenClaw 专用 md2wechat skill 包
npx clawhub@latest install md2wechat
```

当前 ClawHub 路径只会安装 skill 壳到 OpenClaw workspace，**不保证自动安装 `md2wechat` CLI**。完整、可验证的安装主线仍建议使用下一种固定版本 installer。

如果你更习惯先从网页进入，再决定是否安装，可以直接打开：

- [ClawHub 页面：clawhub.ai/geekjourneyx/md2wechat](https://clawhub.ai/geekjourneyx/md2wechat)

#### 方式二：先安装 CLI（优先 Homebrew）

如果你已经装好了 OpenClaw skill 壳，或者只需要先把 CLI 装到环境里，可以先执行：

```bash
brew install geekjourneyx/tap/md2wechat
```

这个方式只安装 `md2wechat` CLI，不会自动写入 `~/.openclaw/skills/md2wechat/`。如果你还没装 skill 壳，请继续执行 `npx clawhub@latest install md2wechat`，或者直接用下一种一键脚本。

如果你已经有 Go 环境，也可以执行：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
```

它同样只安装 CLI，不会自动写入 `~/.openclaw/skills/md2wechat/`。

#### 方式三：一键脚本安装

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install-openclaw.sh | bash
```

脚本会按固定版本安装 OpenClaw skill 包与 `md2wechat` CLI，并校验 `checksums.txt`。这是当前最完整、最可验证的 OpenClaw 安装路径。

推荐直接发给 OpenClaw / 通用大模型的对话脚本：

```text
请帮我安装 OpenClaw 版 md2wechat，并验证 skill 和 CLI 都可用。
执行：
1. curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install-openclaw.sh | bash
2. 先执行：export PATH="$HOME/.local/bin:$PATH"
3. md2wechat version --json
4. md2wechat config init
5. md2wechat config validate
6. md2wechat capabilities --json
如果失败，请继续排查 ~/.openclaw/skills/md2wechat/ 和 `command -v md2wechat`，不要只返回错误原文。
```

#### 方式四：手动安装

```bash
VERSION=2.0.3
# 按你的平台选择对应二进制，这里以 Linux amd64 为例
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/v${VERSION}/md2wechat-openclaw-skill.tar.gz
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/v${VERSION}/md2wechat-linux-amd64
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/v${VERSION}/checksums.txt
sha256sum -c checksums.txt --ignore-missing
mkdir -p ~/.openclaw/skills
mkdir -p ~/.local/bin
mkdir -p /tmp/md2wechat-openclaw
tar -xzf md2wechat-openclaw-skill.tar.gz -C /tmp/md2wechat-openclaw
cp -r /tmp/md2wechat-openclaw/skills/md2wechat ~/.openclaw/skills/
install -m 0755 md2wechat-linux-amd64 ~/.local/bin/md2wechat
export PATH="$HOME/.local/bin:$PATH"
```

手动安装时，请以同一版本 release 提供的 OpenClaw 资产为准，确保 skill 包与 `md2wechat` CLI 同版本安装。

### 配置

安装完成后直接初始化 `md2wechat` 配置：

```bash
md2wechat config init
md2wechat config validate
```

默认配置文件：

```text
~/.config/md2wechat/config.yaml
```

### Claude Code vs OpenClaw

| 方面 | Claude Code | OpenClaw |
|------|-------------|----------|
| **定位** | 终端 AI 编程助手 | 聊天应用 AI 助手 |
| **仓库内 skill 路径** | `skills/md2wechat/` | `platforms/openclaw/md2wechat/` |
| **技能目录** | `~/.claude/skills/` | `~/.openclaw/skills/` |
| **安装方式** | `/plugin` 命令 | `clawhub` CLI + `brew/go install`，或 OpenClaw installer |
| **配置方式** | 环境变量 / `config.yaml` | `~/.config/md2wechat/config.yaml` |
| **LLM 支持** | Claude | Claude、GPT、DeepSeek 等 |

> **详细文档**：[OpenClaw 安装指南](docs/OPENCLAW.md)

---

## 使用示例

### 示例 1：技术博主

```bash
# 写好技术文章
vim my-tech-post.md

# 转换并预览
md2wechat convert my-tech-post.md --preview

# 满意后发送草稿
md2wechat convert my-tech-post.md --draft --cover cover.jpg
```

### 示例 2：产品经理发公告

```bash
# 生成 AI 排版请求，再交给 Claude Code 等环境继续处理
md2wechat convert announcement.md --mode ai --theme ocean-calm
```

### 示例 3：生活方式博主

```bash
# 使用春日清新主题
md2wechat convert travel-diary.md --mode ai --theme spring-fresh --preview
```

### 示例 4：写作小白用观点生成文章

```bash
# 交互模式：输入观点，AI 生成文章
md2wechat write

# 指定风格生成
md2wechat write --style dan-koe

# 生成封面提示词
md2wechat write --style dan-koe --cover-only
```

---

## ❓ 常见问题

<details>
<summary><b>Q: 必须要会编程才能用吗？</b></summary>

**A: 不需要！** 只要会用命令行（终端）就可以。如果是 Windows 用户，下载 .exe 文件后，在 CMD 或 PowerShell 中运行命令即可。
</details>

<details>
<summary><b>Q: AI 模式需要付费吗？</b></summary>

**A:** AI 模式使用 Claude 能力：
- 如果你在 **Claude Code** 中使用，直接调用内置 AI
- 如果你想自己接入，需要配置 OpenAI 兼容的 API
</details>

<details>
<summary><b>Q: 支持哪些 Markdown 语法？</b></summary>

**A:** 支持常用语法：
- 标题（# ## ###）
- 列表（无序、有序）
- 粗体、斜体、行内代码
- 代码块（带语法高亮）
- 引用块
- 分割线
- 图片、链接
- 表格
</details>

<details>
<summary><b>Q: 生成的文章可以直接在微信编辑器中编辑吗？</b></summary>

**A:** 可以！草稿发送后，你可以登录微信公众平台，在草稿箱中继续编辑。
</details>

<details>
<summary><b>Q: 如何知道应该下载哪个版本？下载错了怎么办？</b></summary>

**A:**

**Mac 用户：**
- 点击屏幕左上角苹果图标 → 「关于本机」
- 看到 `Apple M1/M2/M3/M4` → 下载 **Apple Silicon (arm64)** 版本
- 看到 `Intel` → 下载 **Intel (amd64)** 版本

**Linux 用户：**
- 运行 `uname -m` 命令
- 输出 `x86_64` → 下载 **amd64** 版本
- 输出 `aarch64` → 下载 **arm64** 版本

**如果下载错了：**
- 删除错误的文件，重新下载正确版本即可
- 也可以使用一键安装脚本，会自动检测系统架构

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
```
</details>

<details>
<summary><b>Q: 写作功能是什么？怎么用？</b></summary>

**A:** 写作功能可以帮你用特定创作者风格生成文章：

```bash
# 查看所有可用风格
md2wechat write --list

# 交互式写作
md2wechat write

# 指定风格写作
md2wechat write --style dan-koe
```

只需输入你的观点或想法，AI 会自动生成符合该风格的文章。
</details>

<details>
<summary><b>Q: 我是写作小白，不会写文章能用吗？</b></summary>

**A:** 完全可以！写作功能专为小白设计：

1. **只需要输入观点**：比如"我觉得自律是个伪命题"
2. **AI 自动扩展**：会自动生成结构完整的文章
3. **多种风格可选**：内置 Dan Koe 等风格，也可以自定义

不需要你会写作技巧，只需要有想法就行。
</details>

<details>
<summary><b>Q: 如何添加我喜欢的作家风格？</b></summary>

**A:** 在 `writers/` 目录下创建 YAML 文件：

```bash
# 参考内置风格
cat writers/dan-koe.yaml

# 创建自己的风格
vim writers/my-style.yaml
```

详细格式参考 `writers/README.md`。
</details>

<details>
<summary><b>Q: AI 去痕是什么？怎么用？</b></summary>

**A:** AI 去痕功能可以去除 AI 生成文章的"AI 味"，让文章听起来更像真人写的：

```bash
# 基本用法
md2wechat humanize article.md

# 指定强度（gentle/medium/aggressive）
md2wechat humanize article.md --intensity aggressive

# 显示修改对比和质量评分
md2wechat humanize article.md --show-changes
```

**与写作风格组合使用：**

```bash
# 写作 + 去痕
md2wechat write --style dan-koe --humanize
```

> **致谢**：AI 去痕功能基于 [Humanizer-zh](https://github.com/op7418/Humanizer-zh) 项目。
</details>

<details>
<summary><b>Q: AI 去痕会改变文章意思吗？</b></summary>

**A:** 不会。AI 去痕只修改表达方式，不改变核心信息。

它会：
- 删除填充短语和过度强调
- 打破公式化结构
- 变化句子节奏
- 注入真实观点

但会保留原文的核心观点和信息。
</details>

<details>
<summary><b>Q: AI 去痕和写作风格会冲突吗？</b></summary>

**A:** 不会。当两者组合使用时，遵循**风格优先原则**：

- 保留写作风格的核心特征（如 Dan Koe 的破折号停顿）
- 只去除无意的 AI 痕迹
- 保持风格的一致性

```bash
# 风格优先：保留 Dan Koe 风格，去除 AI 痕迹
md2wechat write --style dan-koe --humanize
```
</details>

<details>
<summary><b>Q: 发送草稿时报错 "content size out of limit (errcode=45002)"？</b></summary>

**A:** 这是微信 API 的内容大小限制错误。

**微信草稿 API 限制：**
- **字符数**：< 20,000 字符（中文算 1 个字符）
- **大小**：< 1 MB

**解决方案：**
1. 缩短文章内容
2. 减少不必要的格式（inline CSS 会增加内容体积）
3. 拆分为多篇文章发布
4. 使用更简洁的排版主题

**注意：** 生成的 HTML 包含大量 inline CSS，会使内容体积膨胀约 5-10 倍。长文章建议：
- 使用更简洁的 Markdown 写作
- 删除部分图片或使用外部图片链接
- 手动在微信编辑器中复制粘贴（绕过 API 限制）

**来源：** [微信公众号 API 文档](https://developers.weixin.qq.com/doc/subscription/api/draftbox/draftmanage/api_draft_add.html)
</details>

<details>
<summary><b>Q: ModelScope 图片生成需要多久？</b></summary>

**A:** ModelScope 使用异步 API 模式，通常需要 10-30 秒。

**流程：**
1. 发起请求 → 获取 task_id
2. 轮询任务状态（每 5 秒一次）
3. 任务完成 → 返回图片 URL

**超时设置：**
- 默认最大轮询时间：120 秒
- 超时后会返回错误，建议：
  - 简化提示词
  - 重试一次
  - 检查 ModelScope 服务状态
</details>

<details>
<summary><b>Q: 如何配置 ModelScope 图片生成？</b></summary>

**A:** ModelScope 是推荐的图片生成服务，有免费额度。

**配置步骤：**

1. **获取 API Key**
   - 访问 [modelscope.cn](https://modelscope.cn/my/myaccesstoken)
   - 登录后创建 Access Token
   - 格式类似：`ms-your-token-here`

2. **配置环境变量**
   ```bash
   export IMAGE_PROVIDER=modelscope
   export IMAGE_API_KEY=ms-your-token-here
   export IMAGE_API_BASE=https://api-inference.modelscope.cn
   export IMAGE_MODEL=Tongyi-MAI/Z-Image-Turbo
   ```

3. **或在配置文件中设置**
   ```yaml
   # ~/.config/md2wechat/config.yaml
   api:
     image_provider: modelscope
     image_key: ms-your-token-here
     image_base_url: https://api-inference.modelscope.cn
     image_model: Tongyi-MAI/Z-Image-Turbo
     image_size: 1024x1024
   ```

4. **测试**
   ```bash
   md2wechat generate_image "A golden cat"
   md2wechat generate_cover --title "AI 文章封面测试" --summary "测试封面 preset"
   ```
</details>

<details>
<summary><b>Q: write 命令支持管道输入吗？</b></summary>

**A:** 支持！可以通过管道或 heredoc 传递内容。

**使用方式：**

```bash
# 管道输入
echo "你的想法或内容" | md2wechat write --style dan-koe

# heredoc 输入（适合多行内容）
md2wechat write --style dan-koe --title "文章标题" <<EOF
第一段内容
第二段内容
更多观点...
EOF
```

**适用场景：**
- 脚本自动化
- 从其他命令输出传递内容
- 在 CI/CD 流程中使用
</details>

---

## 更多文档

| 文档 | 说明 |
|------|------|
| [文档索引](docs/README.md) | `docs/` 目录导航 |
| [新手入门指南](docs/QUICKSTART.md) | **强烈推荐！** 详细的图文教程 |
| [完整使用说明](docs/USAGE.md) | 所有命令和选项 |
| [架构说明](docs/ARCHITECTURE.md) | 当前代码边界与主线 |
| [OpenClaw 安装指南](docs/OPENCLAW.md) | OpenClaw 平台安装配置指南 |
| [图片服务配置](docs/IMAGE_PROVISIONERS.md) | AI 图片生成服务完整配置指南 |
| [写作功能指南](writers/README.md) | 如何使用和自定义写作风格 |
| [写作功能问答](docs/WRITING_FAQ.md) | 写作小白完整指南 |
| [AI 去痕指南](docs/WRITING_FAQ.md) | AI 写作去痕完整指南 |
| [常见问题](docs/FAQ.md) | 20+ 常见问题解答 |
| [故障排查](docs/TROUBLESHOOTING.md) | 遇到问题看这里 |

---

## 贡献

欢迎提交 Issue 和 Pull Request！

如果你有好的想法或发现了 bug，请随时提 issue。

---

## 许可证

[MIT License](LICENSE)

---

## 打赏 Buy Me A Coffee

如果该项目帮助了您，请作者喝杯咖啡吧。

### WeChat

<img src="https://raw.githubusercontent.com/geekjourneyx/awesome-developer-go-sail/main/docs/assets/wechat-reward-code.jpg" alt="微信打赏码" width="200" />

---

## 作者

- 作者：**geekjourneyx**
- X（Twitter）：https://x.com/seekjourney
- 公众号：极客杰尼

关注公众号，获取更多 AI 编程、AI 工具与 AI 出海建站的实战分享：

<p align="center">
<img src="https://raw.githubusercontent.com/geekjourneyx/awesome-developer-go-sail/main/docs/assets/qrcode.jpg" alt="公众号：极客杰尼" width="180" />
</p>

如需咨询，扫描底部公众号二维码联系，备注「API内测」。

---

<div align="center">

**让公众号写作更简单**

[主页](https://github.com/geekjourneyx/md2wechat-skill) • [文档](docs) • [反馈](https://github.com/geekjourneyx/md2wechat-skill/issues)

Made by [geekjourneyx](https://geekjourney.dev)

</div>
