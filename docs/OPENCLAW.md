# OpenClaw 安装指南

> 本文档对应 OpenClaw 专用 skill 包 `platforms/openclaw/md2wechat/`。
>
> `skills/md2wechat/` 是给 Claude Code / Codex / OpenCode 的 coding-agent skill，两条路径独立维护。
>
> OpenClaw 路径由两部分组成：OpenClaw skill 壳 + 已安装的 `md2wechat` CLI。当前不再保留 skill 内部 `run.sh` wrapper，也不承担首跑动态下载。

---

## 目录

- [什么是 OpenClaw](#什么是-openclaw)
- [安装方式](#安装方式)
  - [方式一：ClawHub 安装（仅安装 skill 壳）](#方式一clawhub-安装仅安装-skill-壳)
  - [方式二：先安装 CLI（优先 Homebrew）](#方式二先安装-cli优先-homebrew)
  - [方式三：一键脚本安装](#方式三一键脚本安装)
  - [方式四：手动安装](#方式四手动安装)
  - [发给 Agent 的对话脚本](#发给-agent-的对话脚本)
- [配置说明](#配置说明)
- [验证安装](#验证安装)
- [常见问题](#常见问题)
- [与 Claude Code 的区别](#与-claude-code-的区别)

---

## 什么是 OpenClaw

[OpenClaw](https://openclaw.ai/) 是一个开源的 AI Agent 平台，**在你的设备上运行**，通过你已经在用的聊天应用（WhatsApp、Telegram、Discord、Slack、Teams）来操控 AI 助手。

> **The AI that actually does things.**
>
> 清理收件箱、发送邮件、管理日历、航班值机——全部通过你熟悉的聊天应用完成。

**OpenClaw 核心理念：**

```
Your assistant. Your machine. Your rules.
你的助手。你的设备。你的规则。
```

**与 SaaS 助手的区别：** OpenClaw 运行在你选择的地方——笔记本、家用服务器或 VPS。你的基础设施，你的密钥，你的数据。

**OpenClaw 特点：**
- 🦞 **开源免费** - 100,000+ GitHub Stars
- 🏠 **本地运行** - 数据留在你的设备上
- 💬 **多渠道支持** - WhatsApp、Telegram、Discord、Slack、Teams、Twitch、Google Chat
- 🤖 **多模型支持** - Claude、GPT、DeepSeek、KIMI K2.5、Xiaomi MiMo 等
- 🔌 **ClawHub 技能市场** - 安装和分享 AgentSkills

**官方链接：**
- 官网：[openclaw.ai](https://openclaw.ai/)
- 文档：[docs.openclaw.ai](https://docs.openclaw.ai/)
- 技能市场：[clawhub.ai](https://clawhub.ai/)
- 当前技能页面：[clawhub.ai/geekjourneyx/md2wechat](https://clawhub.ai/geekjourneyx/md2wechat)
- GitHub：[github.com/openclaw/openclaw](https://github.com/openclaw/openclaw)

---

## 安装方式

### 方式一：ClawHub 安装（仅安装 skill 壳）

这是当前最直接的官方 skill 壳安装方式：

```bash
# 安装 OpenClaw 专用 md2wechat skill 包
npx clawhub@latest install md2wechat
```

当前 ClawHub 路径只会安装 skill 壳到 OpenClaw workspace，**不保证自动安装 `md2wechat` CLI**。完整、可验证的安装主线仍建议使用下面的固定版本 installer。

如果你更习惯从网页进入当前技能页面，可以直接打开：

- [clawhub.ai/geekjourneyx/md2wechat](https://clawhub.ai/geekjourneyx/md2wechat)

---

### 方式二：先安装 CLI（优先 Homebrew）

如果你已经装好了 OpenClaw skill 壳，或者你的 Agent / 审核系统只关心 CLI 安装方式，可以先安装 `md2wechat` CLI：

```bash
brew install geekjourneyx/tap/md2wechat
```

或者：

```bash
npm install -g @geekjourneyx/md2wechat
```

或者：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v3.0.0
```

这三种方式只安装 CLI，**不会自动把 OpenClaw skill 壳写入 `~/.openclaw/skills/md2wechat/`**。如果你还没装 skill 壳，请继续使用上面的 `npx clawhub@latest install md2wechat`，或者直接使用下面的一键脚本安装。

---

### 方式三：一键脚本安装

适合没有安装 clawhub 的用户：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.0.0/install-openclaw.sh | bash
```

**脚本功能：**
- 按固定版本安装 OpenClaw skill 包与 `md2wechat` CLI
- 自动校验 `checksums.txt`
- 安装到 `~/.openclaw/skills/md2wechat/`
- 安装 CLI 到用户级环境路径，默认是 `~/.local/bin/md2wechat`
- 提示后续直接执行 `md2wechat config init`

### 发给 Agent 的对话脚本

如果你不想自己一步步敲命令，可以直接把下面的话发给 OpenClaw、Claude、GPT 或其他 Agent：

```text
请帮我安装 OpenClaw 版 md2wechat，并验证 skill 和 CLI 都可用。
按这个顺序执行：
1. curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.0.0/install-openclaw.sh | bash
2. 先执行：export PATH="$HOME/.local/bin:$PATH"
3. md2wechat version --json
4. md2wechat config init
5. md2wechat config validate
6. md2wechat capabilities --json
7. md2wechat skills read md2wechat --json
8. 如果我之前装过 skill，再检查 ~/.openclaw/skills/md2wechat/ 是否存在，并确认 `command -v md2wechat` 有输出
如果某一步失败，请继续排查并给我下一条修复命令，不要只返回报错原文。
```

---

### 方式四：手动安装

```bash
# 1. 下载固定版本 release 资产
VERSION=3.0.0
# 按你的平台选择对应二进制，这里以 Linux amd64 为例
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/v${VERSION}/md2wechat-openclaw-skill.tar.gz
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/v${VERSION}/md2wechat-linux-amd64
curl -LO https://github.com/geekjourneyx/md2wechat-skill/releases/download/v${VERSION}/checksums.txt
sha256sum -c checksums.txt --ignore-missing

# 2. 解压并复制技能目录
mkdir -p /tmp/md2wechat-openclaw
tar -xzf md2wechat-openclaw-skill.tar.gz -C /tmp/md2wechat-openclaw
mkdir -p ~/.openclaw/skills
cp -r /tmp/md2wechat-openclaw/skills/md2wechat ~/.openclaw/skills/

# 3. 安装 CLI 到用户级环境路径
mkdir -p ~/.local/bin
install -m 0755 md2wechat-linux-amd64 ~/.local/bin/md2wechat
export PATH="$HOME/.local/bin:$PATH"
```

手动安装时，请以同一版本 release 提供的 OpenClaw 资产为准，确保 skill 包与 `md2wechat` CLI 同版本安装。

---

## 配置说明

安装完成后，直接初始化 `md2wechat` 自己的配置文件即可。

### 初始化配置文件

```bash
md2wechat config init
md2wechat config validate
```

默认配置文件路径：

```text
~/.config/md2wechat/config.yaml
```

推荐在这里配置微信公众号凭证和图片服务，而不是继续维护 `openclaw.json` 里的 skill 环境变量。

最小示例：

```yaml
wechat:
  appid: "你的AppID"
  secret: "你的Secret"
```

### 配置项说明

| 环境变量 | 必需 | 说明 | 获取方式 |
|---------|------|------|---------|
| `WECHAT_APPID` | 草稿上传时 | 微信公众号 AppID | [微信开发者平台](https://developers.weixin.qq.com/platform) → 开发接口管理 |
| `WECHAT_SECRET` | 草稿上传时 | 微信公众号 Secret | 同上，点击"重置"获取 |
| `IMAGE_API_KEY` | AI 图片时 | 图片生成 API Key | 见 [图片服务配置](IMAGE_PROVISIONERS.md) |

### 可选：图片生成配置

如果需要 AI 图片生成功能，在同一份 `config.yaml` 中继续添加：

```yaml
wechat:
  appid: "你的AppID"
  secret: "你的Secret"
api:
  image_provider: "volcengine"
  image_key: "your-ark-api-key"
  image_model: "seedream-3-0"
  image_size: "2K"
```

---

## 验证安装

### 检查技能目录

```bash
ls ~/.openclaw/skills/md2wechat/
```

至少应看到：
```
SKILL.md
```

### 测试运行

```bash
md2wechat --help
```

如果当前 shell 还找不到 `md2wechat`，先执行：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

建议再执行一轮发现命令，确认当前 CLI 和资源都可见：

```bash
md2wechat capabilities --json
md2wechat skills list --json
md2wechat skills read md2wechat --json
md2wechat providers list --json
md2wechat providers show volcengine --json
md2wechat themes list --json
md2wechat prompts list --json
```

`skills read` 读取当前 CLI 二进制内置的核心 coding-agent SOP，用来确认运行时协议和 CLI 版本一致。OpenClaw 专用安装 metadata、ClawHub 安装资源和平台差异仍以 `~/.openclaw/skills/md2wechat/SKILL.md` 或仓库内 `platforms/openclaw/md2wechat/SKILL.md` 为准。

如果你要选图片模型，优先看 `providers show <name> --json` 返回的 `supported_models`，不要凭记忆写死。

### 在 OpenClaw 中使用

启动 OpenClaw 后，直接用自然语言交互：

```
请用秋日暖光主题将 article.md 转换为微信公众号格式
```

---

## 常见问题

### Q: 安装后找不到技能？

**A:** 确认技能目录位置正确：

```bash
# 检查目录结构
tree ~/.openclaw/skills/md2wechat/ -L 1
```

如果目录不存在，重新运行安装脚本。

### Q: 运行时报错 "command not found"？

**A:** 先确认 `md2wechat` CLI 是否已经通过 `brew` 或 OpenClaw 安装器装好，并确认可执行文件可用：

```bash
md2wechat --help
```

如果仍然找不到命令，请重新安装 OpenClaw skill 包与 CLI，然后执行 `export PATH="$HOME/.local/bin:$PATH"`。

### Q: 我不想看文档，能不能直接发一句话给大模型？

**A:** 可以，直接复制这一段：

```text
请帮我安装 OpenClaw 版 md2wechat，并验证 CLI、配置初始化和能力发现都正常。
执行：
1. curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.0.0/install-openclaw.sh | bash
2. 先执行：export PATH="$HOME/.local/bin:$PATH"
3. md2wechat version --json
4. md2wechat config init
5. md2wechat config validate
6. md2wechat capabilities --json
7. md2wechat skills read md2wechat --json
如果失败，请继续排查 skill 目录、PATH 和版本，不要只给我错误信息。
```

### Q: 如何更新技能？

**A:**

```bash
# 如果 CLI 是通过 Homebrew 安装的
brew upgrade geekjourneyx/tap/md2wechat

# 如果 CLI 是通过 go install 安装的
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v3.0.0

# ClawHub 方式
clawhub update md2wechat

# 脚本方式（会覆盖安装）
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.0.0/install-openclaw.sh | bash
```

### Q: 配置没生效？

**A:** 先确认你改的是 `md2wechat` 自己的配置文件，而不是旧的 OpenClaw skill 环境配置。优先检查：

```bash
md2wechat config show --format json
md2wechat config validate
```

### Q: 和 Claude Code 安装冲突吗？

**A:** 不冲突。两个平台使用不同的目录：

| 平台 | 技能目录 |
|------|---------|
| Claude Code | `~/.claude/skills/` |
| OpenClaw | `~/.openclaw/skills/` |

可以同时安装在两个平台。

---

## 与 Claude Code 的区别

| 方面 | Claude Code | OpenClaw |
|------|-------------|----------|
| **定位** | 终端 AI 编程助手 | 聊天应用 AI 助手（WhatsApp/Telegram 等） |
| **运行方式** | 本地终端 | 本地运行，通过聊天应用操控 |
| **仓库内 skill 路径** | `skills/md2wechat/` | `platforms/openclaw/md2wechat/` |
| **技能目录** | `~/.claude/skills/` | `~/.openclaw/skills/` |
| **安装方式** | `/plugin` 命令 | `clawhub` CLI + `brew`，或 OpenClaw installer |
| **配置文件** | 环境变量 / `~/.config/md2wechat/config.yaml` | `~/.config/md2wechat/config.yaml` |
| **LLM 支持** | Claude | Claude、GPT、DeepSeek、KIMI 等 |
| **市场** | Plugin Marketplace | [ClawHub](https://clawhub.ai/) |

**说明：** 两个平台共享同一个 CLI 内核，但 skill 包和安装链分开维护。

---

## 相关链接

- [OpenClaw 官网](https://openclaw.ai/)
- [OpenClaw 文档](https://docs.openclaw.ai/)
- [ClawHub 技能市场](https://clawhub.ai/)
- [OpenClaw GitHub](https://github.com/openclaw/openclaw)
- [md2wechat 主仓库](https://github.com/geekjourneyx/md2wechat-skill)
- [问题反馈](https://github.com/geekjourneyx/md2wechat-skill/issues)

---

<div align="center">

**让公众号写作更简单**

</div>
