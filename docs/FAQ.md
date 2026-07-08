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

如果你已经有稳定可用的 Node/npm 环境，也可以：

```bash
npm install -g @geekjourneyx/md2wechat
```

如果你已经有稳定可用的 Go 环境，也可以：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.9.0
```

如果以上都不适合，再走固定版本安装脚本：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.9.0/install.sh | bash
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

### Q1.1：`npm install -g @geekjourneyx/md2wechat` 提示 `npmmirror` tarball `404`

这通常不是包没发布，而是你的 npm 当前走的是：

```text
https://registry.npmmirror.com
```

而镜像上的新版本 tarball 还没同步完成。

先确认当前 registry：

```bash
npm config get registry
```

如果输出是 `https://registry.npmmirror.com/`，直接改用官方源安装：

```bash
npm install -g @geekjourneyx/md2wechat --registry=https://registry.npmjs.org/
```

如果你想把默认源切回官方 npm：

```bash
npm config set registry https://registry.npmjs.org/
```

对于维护者，npm 发布新版本后还需要额外执行一次：

```bash
npx cnpm sync @geekjourneyx/md2wechat
```

这样可以主动触发 `npmmirror` 同步，减少用户在镜像源上的新版本 `404`。

---

### Q2：OpenClaw / Claude Code 装了 skill，但命令还是跑不起来

先区分两种路径：

- `skills/md2wechat/`：给 Claude Code / Codex / OpenCode 的 coding-agent skill
- `platforms/openclaw/md2wechat/`：给 OpenClaw / ClawHub 的专用 skill

**OpenClaw 路径**还需要先安装 `md2wechat` CLI，不是只把 `SKILL.md` 放进去就够了。CLI 优先通过 `brew` 安装；如果你已有 Go 环境，也可以用 `go install`；否则再用固定版本 installer。skill 壳则继续通过 `clawhub` 或 OpenClaw installer 安装。优先看：

- [OPENCLAW.md](OPENCLAW.md)

**Claude Code / Codex 路径**如果只是二进制没装好，skill 也无法替你凭空执行 CLI。

对 `skills/md2wechat/` 这条 coding-agent 路径，skill 现在直接依赖 `PATH` 里的 `md2wechat`。如果命令不存在，先安装 CLI，再安装 skill。

从支持 `skills` 命令的版本开始，CLI 二进制本身也内置了 coding-agent SOP。Agent 可以先运行：

```bash
md2wechat skills list --json
md2wechat skills read md2wechat --json
```

这条路径用于读取当前二进制版本携带的 `skills/md2wechat/SKILL.md`，避免 Agent 读到旧 README、旧本地 skill 或依赖联网拉仓库。外部 skill 安装仍然有用，主要用于让 Claude Code / Codex / OpenCode 自动发现 `md2wechat` 能力。

推荐先安装 CLI，再安装 skill。mac 用户优先 Homebrew：

```bash
brew install geekjourneyx/tap/md2wechat
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果你已经有 Node/npm 环境，也可以把第一步改成：

```bash
npm install -g @geekjourneyx/md2wechat
```

如果你已经有 Go 环境，再把第一步改成：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.9.0
```

如果以上都不适合，再把第一步改成：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.9.0/install.sh | bash
```

如果你懒得自己操作，也可以直接把下面的话发给 Claude Code / Codex / OpenCode：

```text
请先安装 md2wechat CLI，再安装 md2wechat skill，并验证版本和能力发现都正常。
执行：
1. 如果我是 mac 用户，先运行：brew install geekjourneyx/tap/md2wechat
2. 如果我已经有稳定可用的 Go 环境，也可以改成：go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.9.0
3. 如果以上两种都不适合，再运行：curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.9.0/install.sh | bash
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
1. curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.9.0/install-openclaw.sh | bash
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

如果你更习惯 npm，也可以把第一步改成：

```bash
npm install -g @geekjourneyx/md2wechat
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果你已经有 Go 环境，再改成：

```bash
go install github.com/geekjourneyx/md2wechat-skill/cmd/md2wechat@v2.9.0
md2wechat version --json
npx skills add https://github.com/geekjourneyx/md2wechat-skill --skill md2wechat
```

如果以上都不适合，再改成：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.9.0/install.sh | bash
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
- `md2wechat_base_url`
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
export MD2WECHAT_API_KEY="your_key"
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
md2wechat prompts list --kind title --json
md2wechat prompts list --kind image --archetype cover --json
md2wechat prompts list --kind image --tag editorial --json
```

看具体资源：

```bash
md2wechat providers show openrouter --json
md2wechat providers show volcengine --json
md2wechat themes show autumn-warm --json
md2wechat prompts show wechat-title-expert --kind title --json
md2wechat prompts show cover-default --kind image --json
```

完整说明见：

- [DISCOVERY.md](DISCOVERY.md)

主题发现结果里要重点看两个字段：

- `selectable`: `true` 才能作为 `convert --theme` 直接使用
- `type`: 必须和转换模式匹配，API 模式选 `api` 主题，AI 模式选 `ai` 主题

`api-collection` 这类集合描述会出现在发现结果里，但它不是可执行主题，`selectable` 会是 `false`。

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

### Q11.1：怎么根据文章生成公众号标题候选？

使用标题建议命令：

```bash
md2wechat title suggest article.md --json
```

如果已经知道目标读者，可以传入更具体的上下文：

```bash
md2wechat title suggest article.md --target-reader "AI 工具用户" --count 10 --max-title-chars 25 --hook-level 2 --json
```

这个命令返回 `TITLE_SUGGEST_REQUEST_READY`，表示标题生成 prompt 已准备好，需要宿主 Agent 或外部模型继续执行。它不会直接调用模型，不会写回文章，也不会创建微信草稿。最终标题应由用户确认，或者由上层 Agent 流程基于返回候选的评分再选择。

`--hook-level 1|2|3` 只控制标题张力：`1` = `restrained`，`2` = `punchy`，`3` = `high_tension`。Level 3 不允许编造事实；如果文章没有证据支撑“刚刚”“全网”“第一”“榜首”“变天”等表达，候选应降级张力并说明原因。

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
4. 当前账号还没开通目标模型（例如 Volcengine `ModelNotOpen`）

先执行：

```bash
md2wechat providers list --json
md2wechat providers show volcengine --json
md2wechat config show --format json
```

如果是 Volcengine 返回 `ModelNotOpen`，去 [豆包大模型](https://www.volcengine.com/product/doubao) 点击“控制台” -> “开通管理”，勾选 `Seedream` 模型完成开通，再重试。

然后再试最小命令：

```bash
md2wechat generate_image "test prompt"
```

### Q14.1：Agent 有 Image Gen 时还需要 `IMAGE_API_KEY` 吗？

看你走哪条路径。

直接 CLI 生成会请求图片 provider，所以需要配置 provider 和 `IMAGE_API_KEY` / `api.image_key`：

```bash
md2wechat generate_cover --article article.md
```

Agent 图片计划模式只返回 `IMAGE_PLAN_READY`，不会请求 provider、不会要求或使用 `IMAGE_API_KEY` 进行图片 provider 调用、不会上传微信。它适合当前 Agent 运行时暴露 Image Gen 工具的场景：

```bash
md2wechat generate_cover --article article.md --plan --json
```

返回里的 `requires_provider:false` 和 `requires_image_api_key:false` 表示 md2wechat 这一侧不需要图片服务配置。真正的图片文件只有在宿主 Agent 调用 Image Gen 并保存后才会出现。完整流程见 [Agent 图片计划模式](AGENT_IMAGE_GEN.md)。

### Q14.2：`advise` 和 `inspect` 有什么区别？

`inspect` 是发布前 readiness 真相源，回答“能不能继续 convert/upload/draft”。`advise` 是可选增强建议，回答“这篇已有文章是否值得最小改动地加标题建议、封面计划、layout 模块或微调”。如果 `inspect` 的目标是 `blocked`，先修 blocker，不要用 `advise` 绕过发布前检查。

---

## 微信与草稿

### Q14.5：`inspect` 和 `preview` 应该什么时候用？

推荐把它们放在真正发布前：

```bash
md2wechat inspect article.md
md2wechat preview article.md
```

区别是：

- `inspect`：解释系统最终会怎么理解你的文章，包括标题/作者/摘要来源、H1 风险；`--json` 会在 `data.readiness.targets/blockers` 输出机器可读的 `convert/upload/draft` 目标状态和 blocker 映射，是发布前能否继续的真相源。
- `preview`：生成一个本地 HTML 预览文件，用来确认当前上下文下能否拿到可信预览。

第一版 `preview` 不是可编辑工作台，也不会触发上传、草稿或写回 Markdown。

### Q14.6：为什么 `preview --mode ai` 不给最终视觉稿？

因为当前 AI 模式返回的是 prompt / request，不是最终 HTML。为了避免误导，`preview --mode ai` 会明确降级成确认页，而不是伪造一个“看起来像最终结果”的假稿。

### Q14.7：为什么我传了 `--title` / `--author` / `--digest`，但正文显示看起来没变？

因为这三个参数控制的是微信草稿 metadata，不等于一定会改正文 HTML 里的可见内容。

- `--title` 会影响最终草稿标题，但正文里的 H1 仍然来自 Markdown 正文。
- `--author` 会影响草稿作者字段，但正文是否单独显示作者，取决于主题和正文结构。
- `--digest` 会影响草稿摘要字段，不保证正文里出现一段“摘要文字”。

先跑：

```bash
md2wechat inspect article.md
```

看清最终 metadata 来源、正文 H1、以及两者是否一致。

### Q14.8：为什么图片没有自动替换成微信 URL？

因为图片上传和替换只发生在发布路径：

- `md2wechat convert article.md --upload`
- `md2wechat convert article.md --draft --cover cover.jpg`
- `md2wechat convert article.md --draft --cover-media-id PERMANENT_MEDIA_ID`

纯：

```bash
md2wechat convert article.md --preview
```

只会预览正文输出，不会把本地图片、远程图片或 AI 图片上传到微信并替换 URL。

### Q14.9：`errcode=45004` 到底应该先查什么？

优先查摘要/描述字段，不要先默认成“正文太长”。

在当前语义下，`45004` 更应该理解为摘要/描述超限。优先检查：

1. `--digest`
2. frontmatter 里的 `digest`
3. frontmatter 里的 `summary`
4. frontmatter 里的 `description`

建议先把摘要压到 128 字以内，再重试草稿创建。

### Q14.10：`inspect` 里那些检查码是什么意思？是不是报错了？

不一定。`inspect` 的 `checks` 里既有 `error`，也有 `warn` 和 `info`。

当前最常见的几类是：

- `TITLE_BODY_MISMATCH`：草稿标题和正文 H1 不一样。系统是在提醒你 metadata 和正文是两层概念，不是说转换失败。
- `DIGEST_METADATA_ONLY`：摘要只会进入草稿 metadata，不保证正文 HTML 里也显示一段摘要。
- `IMAGE_REPLACEMENT_REQUIRES_UPLOAD_OR_DRAFT`：当前文章里有图片，但你现在只是 inspect / preview / plain convert；只有 `--upload` 或 `--draft` 才会真正上传并替换图片 URL。

只有 `error` 级别的检查才意味着当前上下文下不能安全执行下一步。

### Q14.11：`--json` 模式下还能放心让 Agent 直接解析吗？

可以。当前契约是：

- stdout：只输出 JSON
- stderr：只保留诊断信息

也就是说，正常的 Agent / 脚本应该直接读取 stdout，不要把 `2>&1` 混在一起再解析。

如果你只是想确认结果，可以直接运行：

```bash
md2wechat inspect article.md --json
md2wechat preview article.md --json
```

这两条现在都符合 machine-readable contract。

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

### Q15.1：运行环境 IP 会变，能不能给微信请求单独走固定出口？

可以。高级版 API 服务提供稳定的微信接口固定出口 IP，用来解决动态公网 IP 反复改微信白名单的问题。开通后，你会拿到完整的 `proxy_url` 和需要填写到微信后台的固定出口 IP。

```yaml
wechat:
  proxy_url: "https://wechat-egress-url-provided-by-md2wechat.example"
```

或临时环境变量：

```bash
export WECHAT_PROXY_URL="https://wechat-egress-url-provided-by-md2wechat.example"
```

`config show --format json` 里对应字段是 `wechat_proxy_url`，默认会隐藏代理密码。这个代理只影响微信上传、建草稿和图片消息发送；API 排版、图片生成 provider、发现命令和普通转换不走它。

微信后台白名单填高级版 API 服务提供的固定出口 IP。不要自行拼接代理主机、端口或部署形态；以服务侧提供的完整 URL 为准。需要开通固定出口能力或企业私有化方案时，请联系作者进行 `API咨询`。`HTTPS_PROXY` 只适合作为全局代理兜底背景，优先用 `wechat.proxy_url` / `WECHAT_PROXY_URL`。

启用代理模式后，上传、草稿和图片消息副作用前需要有效的 `MD2WECHAT_API_KEY`。

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

如果你已经有可复用的微信永久封面素材，也可以在正式转换时直接传：

```bash
md2wechat convert article.md --draft --cover-media-id PERMANENT_MEDIA_ID --json
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
md2wechat providers show volcengine --json
md2wechat themes list --json
md2wechat prompts list --json
```

这样 Agent 才知道：

- 当前配置用了哪份文件
- 默认 provider 是什么
- 当前 provider 支持哪些模型
- 当前有哪些 theme / prompt 真正可用
- 当前 theme 是否可选择，以及适合 API 模式还是 AI 模式

### Why did `config validate` pass but `convert` still failed?

`config validate` 只说明配置能被加载和解析，不代表每个转换路径都可执行。`convert` 还会检查：

- API 模式是否有 API key
- 选择的主题是否存在
- 主题是否 `selectable: true`
- 主题 `type` 是否匹配当前 `--mode`
- draft/upload 路径是否有 WeChat 凭证

如果看到 `THEME_MODE_MISMATCH`、`THEME_NOT_FOUND` 或 `THEME_NOT_SELECTABLE`，先运行：

```bash
md2wechat themes list --json
```

然后选择一个 `selectable: true` 且 `type` 匹配当前模式的主题。

如果想在执行前做本地体检，运行：

```bash
md2wechat doctor --json
```

`doctor` 不调用远程 API，不验证 live auth，不上传图片，也不创建草稿。它只通过 `data.readiness.format_api` / `data.readiness.advanced_layout` / `data.readiness.draft` 报告本地配置、默认 API 转换、默认主题、layout catalog 和 WeChat 草稿凭证是否具备可尝试性。

---

### How do I discover layout modules supported in API mode?

```bash
md2wechat layout list --json           # all 43 modules
md2wechat layout list --serves attention --json   # attention-grabbing modules
md2wechat layout show hero --json      # full spec with fields and example
```

When the user has not chosen a theme or module, use discovery output as facts and let the Agent decide from the article and Brand Profile. The CLI does not parse Brand Profile; Agents should read `~/.config/md2wechat/brand.md` themselves, choose a compatible theme from `themes list --json`, inspect module schemas with `layout show`, and render only the modules they can fill correctly.

Keep the source Markdown read-only: create a temporary formatted Markdown artifact with rendered `:::module` blocks, run `layout validate` on that generated file, then pass the generated Markdown to `/api/convert`. Saving generated Markdown near the source requires explicit user confirmation.

### What does "unknown layout module" in validate output mean?

`layout validate` warns (does not error) for `:::module-name` blocks it does not recognize. This is intentional — it allows forward-compatible documents. If the module name is a typo, fix it by checking `md2wechat layout list --json`. If it's a new custom module, add a YAML spec to `~/.config/md2wechat/layout/<category>/<name>.yaml`.

### How do I add a custom layout module so the CLI recognizes it?

Create a YAML file following the schema:

```yaml
schema_version: "1"
name: my-module
version: "1.0.0"
category: custom
serves: [attention]
fields:
  required:
    - name: title
      description: "Main heading"
      example: "Hello"
metadata:
  author: yourname
  provenance: custom
  inspired_by: "my-design-system#my-module"
example: |
  :::my-module
  title: Hello
  :::
```

Save it to `~/.config/md2wechat/layout/custom/my-module.yaml`. The CLI picks it up on next run — no restart needed.

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
md2wechat version --json
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
