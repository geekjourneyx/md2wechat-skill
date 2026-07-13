# Obsidian / Claudian 指南

这份文档服务一种非常具体的场景：

- 你在用 Obsidian
- 你装了 [Claudian](https://github.com/YishenTu/claudian)
- 你希望在 Obsidian 里直接通过 `/md2wechat` 或自然语言使用 `md2wechat`

一句话结论：

**先安装 `md2wechat` CLI，再安装 skill。**

`npx skills add ...` 只安装 skill，不会替你安装 CLI。

---

## Claudian 是什么

根据 Claudian 官方 README，它是：

- 一个把 Claude Code 嵌入 Obsidian 的插件
- 兼容 Claude Code skill 格式
- 会从 `~/.claude/skills/` 或 `{vault}/.claude/skills/` 发现 skills
- 支持 slash commands

这就是为什么 `md2wechat` 可以在 Claudian 里工作。

你可以把它理解成：

- Claudian 负责把 Agent 带进 Obsidian
- `md2wechat` skill 负责告诉 Agent 怎么排版、生成图片、发草稿
- `md2wechat` CLI 负责真正执行命令

---

## 最短路径

如果你只想尽快跑通，按这个顺序做：

```bash
brew install geekjourneyx/tap/md2wechat
md2wechat version --json
md2wechat skills read md2wechat --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
md2wechat capabilities --json
```

如果你已经有 Go 环境，再改成：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v3.1.0
md2wechat version --json
md2wechat skills read md2wechat --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
md2wechat capabilities --json
```

如果以上都不适合，再改成：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.1.0/install.sh | bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
md2wechat skills read md2wechat --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
md2wechat capabilities --json
```

做到这里，你已经完成了两件事：

1. 本机有可执行的 `md2wechat` CLI
2. Claudian / Claude Code 格式的 skill 已经装到 `~/.claude/skills/`

---

## 保姆级步骤

### 第一步：确认你已经装好 Claudian

先确认这几件事：

- 你已经在 Obsidian 里启用了 Claudian
- Claudian 自己能正常工作
- Claudian 已经能调用 Claude Code / Claude Agent 能力

如果 Claudian 本身还没装好，先按它自己的官方 README 安装。

---

### 第二步：安装 `md2wechat` CLI

mac 用户优先：

```bash
brew install geekjourneyx/tap/md2wechat
```

如果你已经有 Go 环境，也可以：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v3.1.0
```

如果以上都不适合，再用固定版本安装脚本：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.1.0/install.sh | bash
```

默认安装位置：

- macOS / Linux：`~/.local/bin/md2wechat`
- Windows：用户级安装目录或 `C:\Program Files\md2wechat\md2wechat.exe`

安装后先验证：

```bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
```

如果能看到版本 JSON，说明 CLI 安装成功。

---

### 第三步：安装 skill

```bash
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

这一步只会安装 skill，不会安装 CLI。

Claudian 官方说明它会从这些位置发现 skill：

- `~/.claude/skills/`
- `{vault}/.claude/skills/`

所以装完后，你可以检查：

```bash
ls ~/.claude/skills/md2wechat
```

至少应看到：

```text
SKILL.md
```

---

### 第四步：验证能力发现

```bash
md2wechat capabilities --json
md2wechat skills list --json
md2wechat skills read md2wechat --json
md2wechat providers show volcengine --json
md2wechat themes list --json
md2wechat prompts list --kind image --json
md2wechat providers list --json
```

这是最稳的验证方式。  
不要靠猜当前支持什么，先看 discovery 输出。
如果需要选图片模型，优先查看 `providers show <name> --json` 里的 `supported_models`。

---

### 第五步：在 Claudian 里使用

现在回到 Obsidian / Claudian：

- 直接输入 `/md2wechat`
- 或直接告诉 Agent：
  - “请用 md2wechat 把当前 Markdown 转成公众号格式”
  - “请先用 md2wechat capabilities 看当前支持什么”
  - “请为这篇文章生成封面图”

推荐的起手式：

```text
请先用 md2wechat capabilities --json 看当前支持什么，然后把当前文章转换成微信公众号格式，并给我一个预览版本。
```

---

## 如果 Claudian 里还是找不到 `md2wechat`

这是最常见的问题。

### 原因

Obsidian 是图形界面应用。  
很多情况下，GUI 进程拿到的 `PATH` 和你终端里的 `PATH` 不一样。

所以会出现这种情况：

- 终端里 `md2wechat` 正常
- Claudian 里却找不到

### 最稳的修法

去 Claudian：

- `Settings`
- `Environment`
- `Custom variables`

补一条：

```text
PATH=/Users/你的用户名/.local/bin:原来的PATH
```

Linux 可以改成：

```text
PATH=/home/你的用户名/.local/bin:原来的PATH
```

如果你不确定自己 CLI 安装在哪，先在终端执行：

```bash
command -v md2wechat
```

把输出路径前面的目录加进 Claudian 的 `PATH` 即可。

---

## 推荐直接发给 Agent 的话

如果你不想自己逐步敲命令，可以直接把下面这段发给 Claudian、Claude、GPT 或其他 Agent：

```text
请帮我在当前电脑上安装 md2wechat，并让 Claudian 可以在 Obsidian 里通过 /md2wechat 使用它。按这个顺序执行：
1. 如果我是 mac 用户，先运行：brew install geekjourneyx/tap/md2wechat
2. 如果我已经有稳定可用的 Go 环境，也可以改成：go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v3.1.0
3. 如果以上两种都不适合，再运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.1.0/install.sh | bash
4. 如果我是通过 install.sh 安装的，再运行：export PATH="$HOME/.local/bin:$PATH"
5. 运行：md2wechat version --json
6. 运行：md2wechat skills read md2wechat --json
7. 运行：npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
8. 运行：md2wechat capabilities --json
9. 如果 Obsidian / Claudian 里还是找不到 md2wechat，请提醒我去 Claudian 的 Settings -> Environment -> Custom variables，把 PATH 补上 ~/.local/bin
10. 最后告诉我如何在 Claudian 里输入 /md2wechat 或直接调用 md2wechat skill
如果某一步失败，请继续排查并给我下一条修复命令，不要只返回报错原文。
```

---

## 常见问题

### Q1：为什么要先装 CLI？

因为 `npx skills add ...` 只装 skill。  
`md2wechat` 真正执行转换、图片生成、上传草稿，还是靠本机 CLI。

### Q2：为什么我在终端里能用，Claudian 里不能用？

因为 GUI 进程的 `PATH` 和终端不一样。  
优先去 Claudian 的环境变量设置里补 `PATH`。

### Q3：我可以只装 skill，不装 CLI 吗？

不可以。  
skill 只是知识和工作流说明，不是运行时本体。

### Q4：`/md2wechat` 是不是一定会出现？

Claudian 支持 slash commands 和 skills，但你的最终呈现方式会受插件版本、当前配置和 Agent 行为影响。  
最稳的理解方式是：

- skill 已安装
- Agent 可以调用 `md2wechat`
- 你可以通过 `/md2wechat` 或自然语言触发它

如果你更想确认当前可用能力，先在终端执行：

```bash
md2wechat capabilities --json
```

---

## 相关文档

- [安装指南](INSTALL.md)
- [配置指南](CONFIG.md)
- [OpenClaw 指南](OPENCLAW.md)
- [常见问题](FAQ.md)
