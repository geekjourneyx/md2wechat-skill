<div align="center">

<h1>
  <img src="assets/favicon.ico" alt="md2wechat logo" width="28" />
  md2wechat
</h1>

<img src="assets/readme-header.gif" alt="md2wechat — 公众号创作全流程 CLI" width="720" />

**面向 AI Agent 的公众号创作与发布 CLI**

写 Markdown · 43 个高级排版模块 · 40 个专业主题 · AI 配图 · 推送草稿箱<br/>
全流程 CLI，Agent-native — Claude Code · Codex · OpenClaw 原生支持

[![Go Version](https://img.shields.io/badge/Go-1.26.1+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-Source%20Available-orange)](LICENSE)
[![GitHub Release](https://img.shields.io/badge/download-latest-green)](https://github.com/geekjourneyx/md2wechat-skill/releases)
[![Claude Code](https://img.shields.io/badge/Claude%20Code-Skill-purple)](#coding-agent)
[![OpenClaw](https://img.shields.io/badge/OpenClaw-Compatible-00b0aa)](#openclaw)
[![zread](https://img.shields.io/badge/Ask_Zread-_.svg?style=flat&color=00b0aa&labelColor=000000&logo=data%3Aimage%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB3aWR0aD0iMTYiIGhlaWdodD0iMTYiIHZpZXdCb3g9IjAgMCAxNiAxNiIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTQuOTYxNTYgMS42MDAxSDIuMjQxNTZDMS44ODgxIDEuNjAwMSAxLjYwMTU2IDEuODg2NjQgMS42MDE1NiAyLjI0MDFWNC45NjAxQzEuNjAxNTYgNS4zMTM1NiAxLjg4ODEgNS42MDAxIDIuMjQxNTYgNS42MDAxSDQuOTYxNTZDNS4zMTUwMiA1LjYwMDEgNS42MDE1NiA1LjMxMzU2IDUuNjAxNTYgNC45NjAxVjIuMjQwMUM1LjYwMTU2IDEuODg2NjQgNS4zMTUwMiAxLjYwMDEgNC45NjE1NiAxLjYwMDFaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00Ljk2MTU2IDEwLjM5OTlIMi4yNDE1NkMxLjg4ODEgMTAuMzk5OSAxLjYwMTU2IDEwLjY4NjQgMS42MDE1NiAxMS4wMzk5VjEzLjc1OTlDMS42MDE1NiAxNC4xMTM0IDEuODg4MSAxNC4zOTk5IDIuMjQxNTYgMTQuMzk5OUg0Ljk2MTU2QzUuMzE1MDIgMTQuMzk5OSA1LjYwMTU2IDE0LjExMzQgNS42MDE1NiAxMy43NTk5VjExLjAzOTlDNS42MDE1NiAxMC42ODY0IDUuMzE1MDIgMTAuMzk5OSA0Ljk2MTU2IDEwLjM5OTlaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik0xMy43NTg0IDEuNjAwMUgxMS4wMzg0QzEwLjY4NSAxLjYwMDEgMTAuMzk4NCAxLjg4NjY0IDEwLjM5ODQgMi4yNDAxVjQuOTYwMUMxMC4zOTg0IDUuMzEzNTYgMTAuNjg1IDUuNjAwMSAxMS4wMzg0IDUuNjAwMUgxMy43NTg0QzE0LjExMTkgNS42MDAxIDE0LjM5ODQgNS4zMTM1NiAxNC4zOTg0IDQuOTYwMVYyLjI0MDFDMTQuMzk4NCAxLjg4NjY0IDE0LjExMTkgMS42MDAxIDEzLjc1ODQgMS42MDAxWiIgZmlsbD0iI2ZmZiIvPgo8cGF0aCBkPSJNNCAxMkwxMiA0TDQgMTJaIiBmaWxsPSIjZmZmIi8%2BCjxwYXRoIGQ9Ik00IDEyTDEyIDQiIHN0cm9rZT0iI2ZmZiIgc3Ryb2tlLXdpZHRoPSIxLjUiIHN0cm9rZS1saW5lY2FwPSJyb3VuZCIvPgo8L3N2Zz4K&logoColor=ffffff)](https://zread.ai/geekjourneyx/md2wechat-skill)

[快速开始](#quickstart) · [高级排版](#layout) · [API 解锁](#api) · [Agent 支持](#coding-agent) · [常见问题](#faq)

</div>

---

## 为什么不是另一个转换器

市面上有很多 Markdown 排版工具，md2wechat 不一样的地方：

| | 其他工具 | md2wechat |
|---|---|---|
| **输出一致性** | LLM 每次不同 | API 模式确定性输出，同样 Markdown 永远相同 |
| **排版系统** | 靠 prompt 碰运气 | 43 个结构化排版模块（`:::module` 语法），API 专属 |
| **主题数量** | 无 / 寥寥几个 | 40 个专业主题，微信渲染精调 |
| **全流程** | 只做格式转换 | 写作 → 去 AI 痕 → 排版 → AI 配图 → 上传 → 推送草稿 |
| **Agent 集成** | 无结构约定 | JSON envelope、capabilities 端点、discovery 命令 |

---

<a id="api"></a>

## API 模式 — 解锁完整体验

> **AI 模式**（免费）：生成排版 prompt，由你的 Claude / Codex 继续处理，3 个基础主题。
>
> **API 模式**（专业服务）：秒级响应，40 个专业主题，43 个高级排版模块，确定性输出，团队协作与自动化发布首选。

**API 模式专属能力：**

- ✦ **43 个高级排版模块** — `:::hero`、`:::callout`、`:::timeline`… 结构化公众号内容设计语言，详见 [高级排版指南](#layout)
- ✦ **40 个专业主题** — Basic · Minimal · Focus · Elegant · Bold · 精选系列，微信渲染精调，完整预览 [theme-gallery](https://md2wechat.app/theme-gallery)
- ✦ **确定性输出** — 同样 Markdown 每次结果完全一致，适合团队协作和自动化发布
- ✦ **秒级响应** — 无需等待 LLM 生成，适合高频发布场景

**申请 API 服务 / 加入微信交流群：**

扫描下方二维码关注 **极客杰尼** 公众号 → 备注 **「API咨询」** 联系作者；或备注 **「交流群」** 申请加入用户交流群，和同类创作者一起探索 AI 驱动的公众号创作。

**免费公开《杰尼AI实战手册》，我把过去一年做 md2wechat、AI Agent、内容工厂和知识库的实战经验放在这里。不会编程也能看，想写作、做项目、自动化发布，[可以直接来这提问](https://ask.feishu.cn/shared-space/7498802516269236252)。**

<p align="center">
<img src="assets/wechat.png" alt="公众号：极客杰尼" width="160" />
</p>


---

<a id="quickstart"></a>

## 快速开始

### 第一步：安装

```bash
# macOS 优先推荐
brew install geekjourneyx/tap/md2wechat
```

其他安装方式（npm / go install / install.sh / Windows PowerShell）见 [安装指南](docs/INSTALL.md)。

npm 全局安装也可以直接用：

```bash
npm install -g @geekjourneyx/md2wechat
```

### 第二步：配置微信（只需一次）

```bash
md2wechat config init
# 打开生成的配置文件，填入微信公众号 AppID 和 Secret
```

AppID / Secret 获取方式与 IP 白名单配置详见 [微信凭证指南](docs/WECHAT-CREDENTIALS.md)。

### 第三步：开始创作

```bash
# 确认文章解析结果（推荐第一步总是先 inspect）
md2wechat inspect article.md

# 本地预览 HTML（不触发上传或草稿副作用）
md2wechat preview article.md

# 转换并推送微信草稿箱
md2wechat convert article.md --draft --cover cover.jpg
```

如果你想把上面的步骤直接发给 Agent 执行，见 [Agent 安装脚本](docs/INSTALL.md#agent-script)。

<p align="center">
<img src="assets/transform.gif" alt="md2wechat 转换演示" width="720" />
</p>

---

## 核心能力

### 命令速览

| 命令 | 说明 |
|------|------|
| `inspect` | 解析文章元数据与发布就绪状态；`--json` 在 `data.readiness.targets/blockers` 输出可执行目标状态 |
| `preview` | 生成本地预览 HTML，不触发任何上传或草稿副作用 |
| `convert` | Markdown → 微信格式 HTML，可选 `--draft` 直接推送草稿 |
| `write` | 风格写作，从一个想法生成完整文章 + 封面提示词 |
| `humanize` | AI 去痕，让 AI 生成的文章听起来像真人写的 |
| `generate_cover` | AI 生成封面图，内置专业 preset |
| `generate_infographic` | AI 生成信息图，内置 10+ 风格 preset |
| `upload_image` | 上传图片到微信永久素材库 |

### 全流程示例

```bash
# 从一个想法到草稿箱，全流程 4 步
md2wechat write --style dan-koe                          # 1. 生成文章 + 封面提示词
md2wechat humanize article.md                            # 2. 去除 AI 痕迹
md2wechat generate_cover --article article.md            # 3. AI 生成封面图
md2wechat convert article.md --draft --cover cover.jpg  # 4. 推送草稿
```

在 Claude Code 中可以直接发自然语言：

```
"用 Dan Koe 风格写一篇关于 AI 时代独立开发者的文章，生成封面，推送到微信草稿箱"
```

---

<a id="layout"></a>

## 高级排版模块（API 专属）

> **仅 API 模式可用。** 高级排版模块是 md2wechat 独有的能力 — 基于 `:::module` 语法，提供 43 个结构化排版组件，是专为微信公众号设计的内容排版语言。不是 prompt，是一套确定性的设计系统。

### `:::module` 语法示例

在 Markdown 中用 `:::` 包裹排版块：

```markdown
:::hero
eyebrow: 深度观察
title: AI 时代的公众号写作
subtitle: 为什么你需要重新定义「好内容」
:::

:::callout
高级排版模块仅在 API 模式下生效。需要 API Key，扫码联系作者申请。
:::

:::timeline
- 2024：GPT-4 发布，内容生产门槛归零
- 2025：AI 写作工具爆发，同质化严重
- 2026：高质量、有视角的内容成为稀缺品
:::
```

### 五个主题渲染效果

同一篇 Markdown，包含一级标题、二级标题、引用、加粗、列表，以及 `hero` / `cards` / `quote` / `summary` / `cta` 等高级排版模块。

<table>
  <tr>
    <td align="center" width="20%"><strong>default</strong><br><img src="assets/theme-showcase/theme-showcase-default.png" alt="default 主题高级排版效果" width="180"></td>
    <td align="center" width="20%"><strong>bytedance</strong><br><img src="assets/theme-showcase/theme-showcase-bytedance.png" alt="bytedance 主题高级排版效果" width="180"></td>
    <td align="center" width="20%"><strong>elegant-gold</strong><br><img src="assets/theme-showcase/theme-showcase-elegant-gold.png" alt="elegant-gold 主题高级排版效果" width="180"></td>
    <td align="center" width="20%"><strong>elegant-green</strong><br><img src="assets/theme-showcase/theme-showcase-elegant-green.png" alt="elegant-green 主题高级排版效果" width="180"></td>
    <td align="center" width="20%"><strong>sspai-red</strong><br><img src="assets/theme-showcase/theme-showcase-sspai-red.png" alt="sspai-red 主题高级排版效果" width="180"></td>
  </tr>
</table>

### 发现与验证命令

```bash
# 列出全部 43 个模块
md2wechat layout list --json

# 按用途筛选
md2wechat layout list --serves attention --json
md2wechat layout list --serves conversion --json

# 查看模块完整规格
md2wechat layout show hero --json

# 验证文章中的 :::module 用法
md2wechat layout validate --file article.md --json
```

`layout list --json` 和 `layout show --json` 会显示模块的 `body_format`：`fields` / `rows` / `json_object` / `json_array`。按这个字段写模块正文，`example` 只作参考。

保姆级教程（43 个模块全覆盖）见 [docs/LAYOUT.md](docs/LAYOUT.md)。

---

## Agent 发现命令

在 Coding Agent 或自动化脚本中，用 discovery 命令做事实来源，但按任务运行最小必要集合，不要把所有 catalog 都当成固定启动流程：

```bash
md2wechat themes list --json               # 排版选主题时使用，按 selectable/type 判断能否用于当前模式
md2wechat layout list --json               # 高级排版选模块时使用
md2wechat doctor --json                    # API、草稿、上传或配置本地可尝试性排障时使用
md2wechat prompts list --kind image --json # 图片 prompt 选择时使用
md2wechat providers list --json            # 图片生成 provider 选择时使用
md2wechat capabilities --json              # 版本、命令能力或行为边界不确定时使用
md2wechat skills read md2wechat --json     # 读取当前二进制内置的 Agent SOP
```

Agent 排版时应保持原始 Markdown 只读：把文章复制到临时 Markdown，按需用 `layout render` 插入少量模块，先 `layout validate`，再把临时稿交给 `convert`。只有用户明确要求时，才把生成稿保存到源文件旁边。

`skills list/read` 用来发现并读取当前安装版本随二进制发布的 Agent skill，适合在 README、远程仓库或本地 skill 可能过期时先确认运行时 SOP。

所有 discovery 命令加 `--json` 后 stdout 只输出 JSON envelope，适合脚本和 Agent 直接消费。

---

## AI 模式 vs API 模式

| | AI 模式（免费） | API 模式（专业） |
|---|---|---|
| **是否需要 API Key** | 不需要 | 需要（扫码联系作者申请） |
| **输出方式** | 生成 prompt，由外部 LLM 继续处理 HTML | 直接返回最终 HTML |
| **主题数量** | 3 个（autumn-warm / spring-fresh / ocean-calm） | 40 个专业主题 |
| **高级排版模块** | ❌ | ✅ 43 个 |
| **输出一致性** | 每次不同 | 确定性，同样输入同样输出 |
| **响应速度** | 取决于外部 LLM | 秒级 |
| **适合场景** | 实验、偶发写作 | 品牌内容、团队协作、自动化发布 |

```bash
# AI 模式（--mode ai，不需要 API Key）
md2wechat convert article.md --mode ai --theme autumn-warm --preview

# API 模式（默认，需要 API Key）
md2wechat convert article.md --preview
```

---

<a id="coding-agent"></a>

## Coding Agent 支持

md2wechat 是 CLI-first 工具，天然适合集成进 Coding Agent。

```bash
# 安装 CLI（先装这个）
brew install geekjourneyx/tap/md2wechat

# 安装 skill（Claude Code / Codex / OpenCode）
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

安装后在 Claude Code 中直接用自然语言驱动：

```
"把 article.md 转换为微信格式，用 elegant-gold 主题，生成封面图，推送到草稿箱"
"帮我用 inspect --json 检查 data.readiness.targets/blockers，然后预览一下排版效果"
```

### 支持的平台

| 平台 | skill 路径 | 安装文档 |
|------|------------|---------|
| Claude Code / Codex / OpenCode | `skills/md2wechat/` | `npx skills add ...` |
| Obsidian（Claudian 插件） | `~/.claude/skills/` | [docs/OBSIDIAN.md](docs/OBSIDIAN.md) |
| OpenClaw | `platforms/openclaw/md2wechat/` | [docs/OPENCLAW.md](docs/OPENCLAW.md) |

<a id="openclaw"></a>

OpenClaw 用户可以通过 ClawHub 直接安装：[clawhub.ai/geekjourneyx/md2wechat](https://clawhub.ai/geekjourneyx/md2wechat)

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.4.0/install-openclaw.sh | bash
```

---

## 图片生成

支持多种 AI 图片生成服务，用于封面图、信息图和文章配图：

| 服务 | 推荐 | 说明 |
|------|------|------|
| Volcengine Ark | ⭐ 主推荐 | 豆包 Seedream 系列，高质量，国内直连 |
| ModelScope | 次推荐 | 有免费额度，国内访问稳定 |
| OpenRouter | 通用 | 多模型聚合，支持 Gemini / Flux |
| OpenAI | 通用 | 官方 GPT Image / DALL·E |
| Google Gemini | 通用 | 官方 Gemini 图片生成 |

配置方式详见 [图片服务配置指南](docs/IMAGE_PROVISIONERS.md)。

---

## 文档

| 文档 | 说明 |
|------|------|
| [快速入门](docs/QUICKSTART.md) | 详细图文教程，新手优先看这里 |
| [完整使用说明](docs/USAGE.md) | 所有命令和选项 |
| [高级排版模块](docs/LAYOUT.md) | :::module 语法保姆级教程，43 个模块全覆盖 |
| [能力发现](docs/DISCOVERY.md) | discovery 命令与 Prompt Catalog |
| [安装指南](docs/INSTALL.md) | 多平台安装（npm / go / install.sh / Windows） |
| [配置指南](docs/CONFIG.md) | 配置文件与环境变量完整说明 |
| [图片服务配置](docs/IMAGE_PROVISIONERS.md) | AI 图片生成服务配置 |
| [微信凭证指南](docs/WECHAT-CREDENTIALS.md) | AppID / Secret / IP 白名单 |
| [常见问题](docs/FAQ.md) | 20+ 问题解答 |
| [故障排查](docs/TROUBLESHOOTING.md) | 遇到问题看这里 |
| [OpenClaw 指南](docs/OPENCLAW.md) | OpenClaw 平台安装与配置 |
| [Obsidian 指南](docs/OBSIDIAN.md) | Claudian 插件集成 |

---

<a id="faq"></a>

## 常见问题

**Q: 没有 API Key 可以用吗？**

可以。AI 模式不需要 API Key，直接加 `--mode ai` 即可。API 模式需要申请，扫码联系作者。

**Q: 高级排版模块（:::module）只有 API 模式才有？**

是的。43 个结构化排版模块是 API 服务的核心能力，不依赖外部 LLM，输出确定。

**Q: AI 模式和 API 模式有什么本质区别？**

AI 模式返回一个结构化排版 prompt，需要 Claude / Codex 继续处理才能得到 HTML。API 模式直接返回最终 HTML，40 个专业主题，确定性输出，无需额外 LLM。

**Q: 必须会编程才能用吗？**

不需要。会用命令行即可。在 Claude Code / Codex 中可以全程用自然语言驱动，Agent 自动调用 CLI 命令。

**Q: 发送草稿时报错 45002（内容超限）？**

微信草稿 API 限制 < 20,000 字符。API 模式的 inline CSS 会使内容体积膨胀，长文章建议拆分，或使用更简洁的主题。详见 [常见问题](docs/FAQ.md)。

更多问题见 [docs/FAQ.md](docs/FAQ.md)。

---

## md2wechat 生态

围绕 md2wechat 打造的工具和资源矩阵，持续扩展中：

| 仓库 | 说明 |
|------|------|
| [md2wechat](https://github.com/md2wechat) | 品牌主页，所有工具和资源的入口 |
| [md2wechat-guide](https://github.com/md2wechat/md2wechat-guide) | 从入门到 API 的完整使用指南 |
| [awesome-wechat-markdown](https://github.com/md2wechat/awesome-wechat-markdown) | 微信公众号 Markdown 工具精选列表 |
| [md2wechat-templates](https://github.com/md2wechat/md2wechat-templates) | 开箱即用的排版模板库 |

---

## 关于作者

**极客杰尼** — 独立开发者 / AI Builder / AI 科技领域博主

持续打磨面向 AI Agent 的 CLI、API 与公众号自动化工作流。

| | |
|:---|:---|
| 个人主页 | [jieni.ai](https://jieni.ai) |
| GitHub | [geekjourneyx](https://github.com/geekjourneyx) |
| Twitter | [@seekjourney](https://x.com/seekjourney) |
| 公众号 | 微信搜「极客杰尼」 |

**欢迎加入微信交流群** — 扫码关注公众号，备注 **「交流群」** 申请入群；备注 **「API咨询」** 申请 API 服务：

<p align="center">
<img src="assets/wechat.png" alt="公众号：极客杰尼" width="160" />
</p>

---

## 打赏

如果该项目帮助了你，欢迎请作者喝杯咖啡 ☕

<img src="https://raw.githubusercontent.com/geekjourneyx/awesome-developer-go-sail/main/docs/assets/wechat-reward-code.jpg" alt="微信打赏码" width="160" />

---

## 贡献

欢迎提交 Issue 和 Pull Request！有好想法或发现 Bug，随时提 issue。

---

## Star History

<a href="https://www.star-history.com/?repos=geekjourneyx%2Fmd2wechat-skill&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/image?repos=geekjourneyx/md2wechat-skill&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/image?repos=geekjourneyx/md2wechat-skill&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/image?repos=geekjourneyx/md2wechat-skill&type=date&legend=top-left" />
 </picture>
</a>


---

<div align="center">

**让公众号创作回归写作本身**

[主页](https://github.com/geekjourneyx/md2wechat-skill) · [文档](docs) · [生态](https://github.com/md2wechat) · [反馈](https://github.com/geekjourneyx/md2wechat-skill/issues)

Made with ♥ by [geekjourneyx](https://jieni.ai) · [Source Available License](LICENSE) · Personal use free · [Commercial licensing](mailto:skrphper@gmail.com)

</div>
