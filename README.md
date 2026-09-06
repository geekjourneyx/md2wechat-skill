<div align="center">

<h1>
  <img src="assets/favicon.ico" alt="md2wechat logo" width="28" />
  md2wechat
</h1>

<img src="assets/readme-header.gif" alt="md2wechat：原稿中的标题、配图与正文依次编排成文章的 3D 动画" width="720" />

**面向 AI Agent 的微信公众号创作与发布 CLI**

写 Markdown，生成公众号排版，制作封面和文章配图，预览校验后推送草稿箱。

支持 Claude Code、Codex、WorkBuddy、Kimi Work、Hermes Agent、OpenClaw 等 Agent 通过 JSON discovery 稳定调用。

[![Go Version](https://img.shields.io/badge/Go-1.26.1+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-Source%20Available-orange)](LICENSE)
[![GitHub Release](https://img.shields.io/badge/download-latest-green)](https://github.com/geekjourneyx/md2wechat-skill/releases)
[![Agent Ready](https://img.shields.io/badge/Agent-Ready-00b0aa)](#agent-工作流)
[![API](https://img.shields.io/badge/API-Professional-blue)](#专业-api)

[快速开始](#快速开始) · [专业 API ¥199/永久](#专业-api) · [获取 API Key](https://www.md2wechat.cn/api-docs?utm_source=github&utm_medium=readme&utm_campaign=api) · [Agent 工作流](#agent-工作流) · [文档](#文档)

</div>

---

## 这个项目解决什么问题

md2wechat 把公众号发布流程拆成一组可验证的 CLI 命令：

| 场景 | md2wechat 提供 |
|---|---|
| Markdown 转微信 HTML | `convert`，支持预览、上传图片、创建草稿 |
| 发布前检查 | `inspect --json` 输出标题、摘要、图片、cover、draft readiness |
| 稳定排版 | API 模式成功时返回最终 HTML，覆盖 77 个主推高级排版场景条目和 56 个主推 `:::` 语法名 |
| Agent 自动化 | `capabilities`、`doctor`、`themes`、`layout`、`providers` 等 discovery 命令 |
| 内容生产 | `write`、`humanize`、`title suggest`、`generate_cover`、`generate_infographic` |
| 多账号发布 | 命名公众号账号，本地只读发现，不输出 Secret |
| 微信白名单 | 高级 API 服务可提供微信接口固定出口能力 |

---

## 快速开始

```bash
npm install -g @geekjourneyx/md2wechat
md2wechat version --json
md2wechat config init --json
md2wechat config validate --json
```

API 模式预览和转换需要 md2wechat API Key。专业 API 统一版 ¥199/永久；可直接[查看价格与获取方式](https://www.md2wechat.cn/api-docs?utm_source=github&utm_medium=readme&utm_campaign=api)。完成凭证配置后，先检查文章，再把 HTML 明确写入本地文件：

```bash
md2wechat inspect article.md --json
md2wechat preview article.md --output preview.html
md2wechat convert article.md --output article.html
```

以上命令不会上传图片或创建草稿。需要创建微信草稿时，先按同一目标检查，再显式执行副作用：

```bash
md2wechat inspect article.md --draft --cover cover.jpg --json
md2wechat convert article.md --draft --cover cover.jpg
```

如果使用可选的 `--wechat-account`，必须在 `inspect` 和 `convert` 两条命令中传入同一个账号名。

安装方式、微信凭证和 IP 白名单配置见：

- [安装指南](docs/INSTALL.md)
- [微信凭证与 IP 白名单指南](docs/WECHAT-CREDENTIALS.md)
- [配置保姆级指南](docs/CONFIG-WALKTHROUGH.md)

---

## 专业 API

API 模式适合需要稳定输出、多人协作、批量发布或 Agent 自动化的场景。

**统一版 ¥199/永久，一次购买。** 包含稳定转换接口、定期更新和专属创作交流群。

[查看价格与获取 API Key](https://www.md2wechat.cn/api-docs?utm_source=github&utm_medium=readme&utm_campaign=api)

| 能力 | 免费 AI 模式 | 专业 API 模式 |
|---|---|---|
| 输出方式 | 生成 prompt，由外部 LLM 继续处理 | 直接返回微信 HTML |
| 主题 | 3 个基础主题 | 48 个专业主题 |
| 高级排版模块 | 不解析，`:::module` 作为普通段落输出 | API renderer 解析 56 个推荐 `:::module` 语法 |
| 转换结果 | 需要外部 LLM 完成 HTML | converter 成功时返回最终 HTML |
| 发布自动化 | 适合实验 | 适合团队、客户号、矩阵号 |

专业能力包括：

- 48 个微信渲染精调主题：[theme-gallery](https://md2wechat.app/theme-gallery)
- 77 个主推高级排版场景条目，对应 56 个主推 `:::` 语法名：[docs/LAYOUT.md](docs/LAYOUT.md)
- 3 个兼容模块只用于旧稿迁移；加上 4 个基础增强能力，共计 63 项渲染层语法能力
- 多公众号账号：[docs/CONFIG.md](docs/CONFIG.md)
- 微信接口固定出口：[docs/WECHAT-CREDENTIALS.md](docs/WECHAT-CREDENTIALS.md)
- 发布前 readiness 检查：[docs/DISCOVERY.md](docs/DISCOVERY.md)

获取 API Key：

- 正式使用：[查看价格与获取方式](https://www.md2wechat.cn/api-docs?utm_source=github&utm_medium=readme&utm_campaign=api)
- 微信咨询：关注公众号「极客杰尼」，备注「API咨询」

<p align="center">
<img src="assets/wechat.png" alt="公众号：极客杰尼" width="160" />
</p>

---

## Agent 工作流

md2wechat 给 Agent 提供可机读接口，减少猜测和误操作。

```bash
md2wechat capabilities --json
md2wechat doctor --json
md2wechat inspect article.md --json
md2wechat themes list --json
md2wechat layout list --json
md2wechat title suggest article.md --json
md2wechat title suggest article.md --json --hook-level 2
md2wechat skills list --json
md2wechat skills read md2wechat --json
```

按任务选择 discovery：用 `capabilities` 获取聚合路由事实，用资源的 `list` 做选择，用 `show` 查看单个资源的完整定义，仅在该资源支持时使用 `render`。JSON stdout 是单行紧凑对象并以换行结束；人类阅读时可在命令后加 `| jq`，不要要求 CLI 改成缩进输出。

文章命令边界固定为：`inspect` 返回结构化 metadata、checks、readiness targets 和 blockers；`preview` 只把成功的 API converter 最终 HTML 原样写入文件；`convert` 执行转换，并且只在用户明确请求时执行 upload/draft 副作用。AI preview 返回 `PREVIEW_ACTION_REQUIRED` 且不创建输出文件，需要 readiness 时使用 `inspect --json`。

这些命令适合 Claude Code、Codex、WorkBuddy、Kimi Work、Hermes Agent、OpenClaw 以及其他能调用本地 CLI 的 Agent 使用。

Agent 可以据此判断：

- 当前 CLI 支持哪些命令
- API、草稿、上传是否具备执行条件
- 某篇文章能不能发草稿
- 当前主题和排版模块是否可用
- 标题建议是否应交给宿主 Agent / 外部模型完成
- 当前二进制内置的 Agent SOP 是什么

Brand Profile 支持把长期风格偏好写入 `~/.config/md2wechat/brand.md`，由 Agent 在写作和排版时读取。详见 [docs/BRAND-PROFILE.md](docs/BRAND-PROFILE.md)。

---

## 图片生成

md2wechat 支持两条图片路径。

先从当前二进制发现可用 preset，再调用图片 provider：

```bash
md2wechat prompts list --kind image --archetype cover --json
md2wechat prompts list --kind image --archetype infographic --json

md2wechat generate_cover --article article.md
md2wechat generate_cover --article article.md --preset cover-semantic-concept
md2wechat generate_infographic --article article.md --preset infographic-claude-warm
```

完整 preset 清单、用途和默认画幅以 `prompts list/show --json` 为准，文档只保留代表性示例。

支持 Volcengine、ModelScope、OpenRouter、OpenAI、Gemini、MiniMax 等服务。配置见 [docs/IMAGE_PROVISIONERS.md](docs/IMAGE_PROVISIONERS.md)。

需要在生成结果中保持同一人物形象时，可以用 MiniMax 的主体参考（图生图）：

```bash
md2wechat generate_image "保持同一人物形象的秋日封面" \
  --subject-reference "https://cdn.example.com/portrait.png"
```

`--subject-reference` 只支持 `minimax` provider 的 `image-01` 模型，参考图必须是可公开访问的 `http(s)` 图片 URL。

使用宿主 Agent 的 Image Gen：

```bash
md2wechat generate_cover --article article.md --plan --json
md2wechat generate_infographic --article article.md --plan --json
```

计划模式返回 `IMAGE_PLAN_READY`，不请求图片 provider，不要求 `IMAGE_API_KEY`，也不会上传到微信。仅当当前宿主运行时实际暴露 Image Gen 工具时，Agent 才能继续执行图片生成。详见 [docs/AGENT_IMAGE_GEN.md](docs/AGENT_IMAGE_GEN.md)。

---

## 高级排版

API 模式支持 `:::module` 语法，用 Markdown 写结构化公众号排版。

```markdown
:::hero
eyebrow: 深度观察
title: AI 时代的公众号写作
subtitle: 为什么读者愿意继续读下去
:::

:::callout
type: info
body: 高级排版模块只在 API 模式渲染。
:::
```

查看和验证模块：

```bash
md2wechat layout list --json
md2wechat layout show hero --json
md2wechat layout validate --file article.md --json
```

本地 `layout validate` 只验证语法，不能证明远端 renderer 已部署；需要通过 API `preview` 或 `convert` 验证实际渲染。

<p align="center">
<img src="assets/theme-showcase/theme-showcase-default.png" alt="default 主题效果" width="180" />
<img src="assets/theme-showcase/theme-showcase-bytedance.png" alt="bytedance 主题效果" width="180" />
<img src="assets/theme-showcase/theme-showcase-elegant-gold.png" alt="elegant-gold 主题效果" width="180" />
</p>

完整教程见 [docs/LAYOUT.md](docs/LAYOUT.md)。

这里的计数不是同一维度：77 是上游使用场景条目，一个语法名可以覆盖多个结构变体；56 是 `layout list --json` 默认返回的推荐语法名。兼容模块默认不混入推荐列表。完整计数契约见 [docs/LAYOUT.md](docs/LAYOUT.md)。

---

## 常用命令

| 命令 | 用途 |
|---|---|
| `inspect` | 返回结构化 metadata、checks、readiness targets 和 blockers |
| `advise` | 为已有文章推荐可选的最小增强动作 |
| `preview` | 只写入成功 API 转换的最终 HTML；失败或 AI handoff 不新建或覆盖 |
| `convert` | 转换 Markdown，并按显式请求执行 upload/draft 副作用 |
| `write` | 从想法生成文章 |
| `humanize` | 重写 AI 文章，支持 `authentic` 强度 |
| `title suggest` | 生成公众号标题建议的 AI 请求 |
| `generate_cover` | 生成封面图或图片计划 |
| `generate_infographic` | 生成信息图或图片计划 |
| `upload_image` | 上传图片到微信素材库 |
| `create_image_post` | 创建微信图片消息（小绿书/newspic） |
| `config wechat-accounts` | 查看本地多公众号账号配置 |
| `doctor` | 本地配置体检 |

---

## 文档

| 文档 | 内容 |
|---|---|
| [QUICKSTART](docs/QUICKSTART.md) | 新手主路径 |
| [USAGE](docs/USAGE.md) | 命令完整说明 |
| [DISCOVERY](docs/DISCOVERY.md) | Agent discovery 契约 |
| [WORKBUDDY](docs/WORKBUDDY.md) | WorkBuddy 安装、检查、预览与草稿确认流程 |
| [ADVISE](docs/ADVISE.md) | 已有文章的可选增强建议 |
| [LAYOUT](docs/LAYOUT.md) | 高级排版模块教程与 discovery 用法 |
| [HUMANIZE](docs/HUMANIZE.md) | AI 去痕与 authentic 写作 |
| [AGENT_IMAGE_GEN](docs/AGENT_IMAGE_GEN.md) | 宿主 Agent Image Gen 工作流 |
| [CONFIG](docs/CONFIG.md) | 配置字段和环境变量 |
| [FAQ](docs/FAQ.md) | 常见问题 |
| [TROUBLESHOOTING](docs/TROUBLESHOOTING.md) | 故障排查 |

---

## 许可与商业使用

本项目采用 Source Available License。个人使用、学习、评估、非营利使用免费。商业使用、SaaS、客户交付、白标、再分发和训练数据用途需要商业授权。

专业 API：[查看价格与获取 API Key](https://www.md2wechat.cn/api-docs?utm_source=github&utm_medium=readme&utm_campaign=api)。商业授权（SaaS、客户交付、白标、再分发和训练数据）请联系 `skrphper@gmail.com`。

---

<div align="center">

[文档](docs) · [Issues](https://github.com/geekjourneyx/md2wechat-skill/issues) · [Commercial licensing](mailto:skrphper@gmail.com)

Made by [geekjourneyx](https://jieni.ai)

</div>
