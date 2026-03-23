# 常见问题（FAQ）

这份 FAQ 只回答一件事：**新手最常卡在哪里，最快怎么排掉。**

如果你是第一次接触 `md2wechat`，推荐先看这些文档：

- [安装指南](INSTALL.md)
- [配置指南](CONFIG.md)
- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)
- [真实烟雾测试记录](SMOKE.md)

---

## 目录

- [安装与启动](#安装与启动)
- [配置与默认行为](#配置与默认行为)
- [转换与排版](#转换与排版)
- [图片与素材](#图片与素材)
- [微信与草稿](#微信与草稿)
- [Agent 与自动化](#agent-与自动化)
- [调试与求助](#调试与求助)

---

## 安装与启动

### Q1：提示 `command not found: md2wechat`

**原因**：程序没有在 `PATH` 里。

**先做这两步：**

```bash
command -v md2wechat
md2wechat --help
```

如果 `command -v` 没输出，说明系统找不到二进制。

**解决方案 A：重新安装 CLI**

如果你在 macOS 上，优先：

```bash
brew install geekjourneyx/tap/md2wechat
```

如果你已经有稳定可用的 Go 环境，也可以：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
```

如果以上都不适合，再走固定版本安装脚本：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
```

安装脚本默认会把 CLI 放到：

- macOS / Linux: `~/.local/bin/md2wechat`
- Windows: 用户级安装目录或 `C:\Program Files\md2wechat\md2wechat.exe`

**解决方案 B：把二进制目录加到 PATH**

```bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
```

如果你不确定装到了哪里，优先重新走安装脚本，不要手猜路径。

---

### Q2：OpenClaw / Claude Code 装了 skill，但命令还是跑不起来

先区分两种路径：

- `skills/md2wechat/`：给 Claude Code / Codex / OpenCode 的 coding-agent skill
- `platforms/openclaw/md2wechat/`：给 OpenClaw / ClawHub 的专用 skill

**OpenClaw 路径**还需要先安装 `md2wechat` CLI，不是只把 `SKILL.md` 放进去就够了。CLI 优先通过 `brew` 安装；如果你已有 Go 环境，也可以用 `go install`；否则再用固定版本 installer。skill 壳则继续通过 `clawhub` 或 OpenClaw installer 安装。优先看：

- [OPENCLAW.md](OPENCLAW.md)

**Claude Code / Codex 路径**如果只是二进制没装好，skill 也无法替你凭空执行 CLI。

对 `skills/md2wechat/` 这条 coding-agent 路径，skill 现在直接依赖 `PATH` 里的 `md2wechat`。如果命令不存在，先安装 CLI，再安装 skill。

推荐先安装 CLI，再安装 skill。mac 用户优先 Homebrew：

```bash
brew install geekjourneyx/tap/md2wechat
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果你已经有 Go 环境，再把第一步改成：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
```

如果以上都不适合，再把第一步改成：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
```

如果你懒得自己操作，也可以直接把下面的话发给 Claude Code / Codex / OpenCode：

```text
请先安装 md2wechat CLI，再安装 md2wechat skill，并验证版本和能力发现都正常。
执行：
1. 如果我是 mac 用户，先运行：brew install geekjourneyx/tap/md2wechat
2. 如果我已经有稳定可用的 Go 环境，也可以改成：go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
3. 如果以上两种都不适合，再运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
4. 运行：npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
5. 如果我是通过 install.sh 安装的，再执行：export PATH="$HOME/.local/bin:$PATH"
6. md2wechat version --json
7. md2wechat capabilities --json
8. md2wechat config init
如果失败，请继续排查，不要只返回错误原文。
```

如果你走的是 OpenClaw 路径，直接发这段：

```text
请帮我安装 OpenClaw 版 md2wechat，并验证 skill 和 CLI 都可用。
执行：
1. curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install-openclaw.sh | bash
2. 先执行：export PATH="$HOME/.local/bin:$PATH"
3. md2wechat version --json
4. md2wechat config init
5. md2wechat config validate
6. md2wechat capabilities --json
如果失败，请继续排查 ~/.openclaw/skills/md2wechat/ 和 `command -v md2wechat`，不要只给我报错。
```

### Q3：我在 Obsidian 的 Claudian 里怎么用 `/md2wechat`？

先做这几步：

```bash
brew install geekjourneyx/tap/md2wechat
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果你已经有 Go 环境，再改成：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.0.3
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果以上都不适合，再改成：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

然后回到 Claudian：

- 直接输入 `/md2wechat`
- 或直接让 Agent 调用 `md2wechat` skill

如果终端里有 `md2wechat`，但 Claudian 里还是找不到，优先去：

- `Settings -> Environment -> Custom variables`

补上你的 CLI 路径，例如：

```text
PATH=/Users/你的用户名/.local/bin:原来的PATH
```

完整说明见：

- [Obsidian / Claudian 指南](OBSIDIAN.md)

---

### Q4：macOS 提示“无法打开，因为无法验证开发者”

这是 macOS 的安全提示，不是 `md2wechat` 特有问题。

可尝试：

```bash
sudo xattr -cr /Applications/md2wechat
```

或者在系统设置里手动允许打开。

---

## 配置与默认行为

### Q5：配置文件到底在哪？

主路径是：

```text
~/.config/md2wechat/config.yaml
```

先执行：

```bash
md2wechat config init
md2wechat config show --format json
```

第二个命令会直接告诉你当前实际生效的是哪份配置。

更完整说明见：

- [CONFIG.md](CONFIG.md)

---

### Q6：提示 `WECHAT_APPID is required`

**原因**：你还没配置微信凭证，或者当前生效的配置文件里没有它们。

最稳做法：

```bash
md2wechat config init
```

然后编辑：

```yaml
wechat:
  appid: "你的公众号 AppID"
  secret: "你的公众号 AppSecret"
```

再执行：

```bash
md2wechat config validate
md2wechat config show --format json
```

如果你不知道去哪里拿 AppID / AppSecret，直接看：

- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)

---

### Q7：我没传 `--mode`，默认到底走 API 还是 AI？

**默认一定是 API。**

也就是：

```bash
md2wechat convert article.md
```

当前等价于：

```bash
md2wechat convert article.md --mode api
```

只有你显式传：

```bash
md2wechat convert article.md --mode ai
```

才会进入 AI 模式。

---

### Q8：我改了配置，但感觉没生效

最常见原因有 3 个：

1. 你改的不是当前生效的配置文件
2. 环境变量覆盖了配置文件
3. 你以为 `api.convert_mode` 会覆盖 `convert` 默认行为

先执行：

```bash
md2wechat config show --format json
```

重点看：

- `config_file`
- `image_provider`
- `default_convert_mode`

注意：

- `api.convert_mode` / `CONVERT_MODE` 当前不会覆盖“`convert` 不传 `--mode` 默认是 API”这个行为

---

### Q8：API 模式提示需要 API Key

API 模式需要 `md2wechat` 排版服务的 API Key。

如果你没配 API Key，有两条路：

**方案 A：配置 API Key**

```bash
```

**方案 B：显式改走 AI 模式**

```bash
md2wechat convert article.md --mode ai --theme autumn-warm
```

---

## 转换与排版

### Q9：AI 模式为什么没有直接产出最终 HTML？

这是当前 CLI 的设计，不是异常。

当前 `convert --mode ai` 的语义是：

- 生成 AI request / prompt
- 返回 `status=action_required`
- 写出 `*.prompt.txt`

它不是“本地自动完成 HTML 排版”的完全闭环。

如果你要稳定、直接的 HTML 结果，优先用：

```bash
md2wechat convert article.md --mode api
```

如果你想理解当前 AI 模式的真实行为，先看：

- [SMOKE.md](SMOKE.md)

---

### Q10：转换结果为空、乱码或者很奇怪

优先排查两件事：

1. 文件编码不是 UTF-8
2. Markdown 内容本身有结构问题

先试：

```bash
file article.md
```

如果不是 UTF-8，可转码：

```bash
iconv -f GBK -t UTF-8 article.md > article-utf8.md
```

如果仍异常，再检查 Markdown 本身。

---

### Q11：我想知道当前支持哪些主题、provider、prompt，不想靠文档猜

直接用发现命令，不要猜：

```bash
md2wechat capabilities --json
md2wechat providers list --json
md2wechat themes list --json
md2wechat prompts list --json
md2wechat prompts list --kind image --archetype cover --json
md2wechat prompts list --kind image --tag editorial --json
```

看具体资源：

```bash
md2wechat providers show openrouter --json
md2wechat themes show autumn-warm --json
md2wechat prompts show cover-default --kind image --json
```

完整说明见：

- [DISCOVERY.md](DISCOVERY.md)

如果你不想自己写图片 prompt，可以直接用内置 preset：

```bash
md2wechat generate_cover --article article.md
md2wechat generate_infographic --article article.md --preset infographic-comparison
md2wechat generate_infographic --article article.md --preset infographic-dark-ticket-cn --aspect 21:9
md2wechat generate_infographic --article article.md --preset infographic-handdrawn-sketchnote
md2wechat generate_infographic --article article.md --preset infographic-apple-keynote-premium
md2wechat generate_infographic --article article.md --preset infographic-victorian-engraving-banner --aspect 21:9
```

如需只在本次调用切换模型，可直接加 `--model`：

```bash
md2wechat generate_cover --article article.md --model gemini-3-pro-image-preview
```

如果你不确定某个图片 preset 更偏封面还是信息图，先运行：

```bash
md2wechat prompts show <preset-name> --kind image --json
```

优先看输出里的 `primary_use_case`、`compatible_use_cases` 和 `default_aspect_ratio`。有些信息图 preset 也可以兼作封面，不需要复制成两份模板。

---

## 图片与素材

### Q12：图片上传失败 `upload material failed`

先按这个顺序排：

1. 图片格式是否支持
2. 图片是否太小或太异常
3. 微信凭证是否有效
4. IP 白名单是否已配置

支持的常见格式：

- `jpg`
- `png`
- `gif`
- `bmp`
- `webp`

真实 smoke 里还发现一个现象：

- 极小测试图（例如 1x1 PNG）可能被微信拒绝为 `unsupported file type`

所以调试时优先用正常尺寸图片，不要用极小占位图。

---

### Q13：为什么图片链接没有被替换成微信素材地址？

通常是因为你没走上传链。

例如你需要：

```bash
md2wechat convert article.md --upload -o output.html
```

如果你只是：

```bash
md2wechat convert article.md -o output.html
```

那它不会帮你上传图片，也不会把文内图片替换成微信素材地址。

---

### Q14：AI 生成图片失败

最常见原因：

1. `IMAGE_API_KEY` 没配
2. 当前 provider 配置不完整
3. 你选的 provider 模型或 base URL 不对

先执行：

```bash
md2wechat providers list --json
md2wechat config show --format json
```

然后再试最小命令：

```bash
md2wechat generate_image "test prompt"
```

---

## 微信与草稿

### Q15：第一次调用微信接口就报 `ip not in whitelist`

这是微信接口的前置限制，不是代码 bug。

最常见报错类似：

```text
ip xxx.xxx.xxx.xxx not in whitelist
```

解决步骤：

1. 在实际执行 `md2wechat` 的机器上查公网 IP
2. 去微信开发者平台的 `开发接口管理`
3. 把这个公网 IP 加到 `IP 白名单`
4. 等几分钟后再重试

完整新手说明见：

- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)

---

### Q16：草稿创建失败 `create draft failed`

先排这几类：

1. 公众号权限不足
2. 白名单没配
3. 封面图上传失败
4. 内容包含敏感词

最稳的调试顺序不是直接跑完整链，而是：

```bash
md2wechat config validate
md2wechat upload_image cover.png --json
md2wechat test-draft --json draft.html cover.png
```

前两步都过了，再测：

```bash
md2wechat convert article.md --upload --draft --cover cover.png --json
```

---

### Q17：`access_token expired` 是不是凭证坏了？

不一定。

微信的 `access_token` 本来就会过期。通常程序会自己刷新。
如果你持续失败，再排：

1. `AppID` / `AppSecret` 是否真的填对
2. 你是不是刚重置过 `AppSecret`
3. 当前生效配置是不是你以为的那份

先看：

```bash
md2wechat config show --format json
```

---

## Agent 与自动化

### Q18：Agent 应该先看哪份配置、先跑什么命令？

默认先看：

```text
~/.config/md2wechat/config.yaml
```

然后建议按这个顺序：

```bash
md2wechat config show --format json
md2wechat capabilities --json
md2wechat providers list --json
md2wechat themes list --json
md2wechat prompts list --json
```

这样 Agent 才知道：

- 当前配置用了哪份文件
- 默认 provider 是什么
- 当前有哪些 theme / prompt 真正可用

---

### Q19：CI / GitHub Actions 里能直接调微信吗？

可以，但前提是你解决了**白名单和固定出口 IP**问题。

如果你的运行环境公网 IP 会频繁变化，最容易出问题的不是配置，而是微信白名单。

所以更推荐：

- 用固定公网 IP 的服务器
- 或固定出口网关

而不是直接依赖动态 IP 的 CI 环境去调用微信接口。

---

## 调试与求助

### Q20：遇到问题时，最推荐的排查顺序是什么？

按这个顺序最稳：

```bash
md2wechat config validate --json
md2wechat config show --format json
md2wechat upload_image --json cover.png
md2wechat test-draft --json draft.html cover.png
md2wechat convert article.md --mode api --upload --draft --cover cover.png --json
```

如果你还要测 AI：

```bash
md2wechat convert article.md --mode ai --json
```

这个顺序的好处是：

- 先把配置问题排掉
- 再把图片上传问题排掉
- 再把草稿问题排掉
- 最后才测完整转换链

---

### Q21：如何获取帮助？

先看文档：

- [INSTALL.md](INSTALL.md)
- [CONFIG.md](CONFIG.md)
- [WECHAT-CREDENTIALS.md](WECHAT-CREDENTIALS.md)
- [DISCOVERY.md](DISCOVERY.md)
- [SMOKE.md](SMOKE.md)

再看命令帮助：

```bash
md2wechat --help
md2wechat convert --help
md2wechat create_image_post --help
```

如果还解决不了，再提 Issue。

---

## 仍然无法解决？

提 Issue 时，建议一并提供：

### 1. 版本信息

```bash
md2wechat --version
go version
```

### 2. 当前配置摘要

```bash
md2wechat config show --format json
```

### 3. 失败命令和完整错误输出

```bash
md2wechat convert article.md 2>&1
```

### 4. 系统信息

```bash
uname -a
```

或 Windows：

```powershell
systeminfo
```

提交到：

- [GitHub Issues](https://github.com/geekjourneyx/md2wechat-skill/issues)
