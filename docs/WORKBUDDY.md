# 在 WorkBuddy 中使用 md2wechat

这是一份从零开始的纯文字教程。完成后，你可以让 WorkBuddy 调用 `md2wechat` 检查 Markdown、生成本地 HTML 预览，并在你明确确认后创建微信公众号草稿。

先记住两条边界：

- WorkBuddy Skill 和 `md2wechat` CLI 是两层，二者都要安装。Skill 告诉 Agent 怎样安全调用，CLI 才是真正执行命令的程序。
- HTML 转换、微信草稿、图片生成使用三类不同凭证。只配置当前任务需要的凭证，不要把真实密钥粘贴到对话、文章或 Git 仓库。

## 一、下载并登录 WorkBuddy

1. 打开 [WorkBuddy 官方下载页](https://www.codebuddy.cn/work/)，只从官网的“下载 WorkBuddy”入口选择安装包。
2. macOS 按芯片选择 Apple 芯片或 Intel 版本；Windows 选择 Windows x64 版本。系统要求和安装细节以官网当前提示为准。
3. 安装并启动 WorkBuddy，点击“登录”，同意服务条款与隐私协议后使用微信扫码登录。
4. 登录成功后，新建任务，并选择一个专门存放文章的本地工作目录。后续的 `article.md`、`cover.jpg` 和 HTML 都放在这个目录里，最容易检查。

如果安装或登录界面与这里不同，可分别查看 [Windows 官方安装指南](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Installation-Win-Guide) 与 [Mac 官方安装指南](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Installation-Mac-Guide)。不要从非官网镜像下载客户端。

## 二、安装 md2wechat Skill

按当前官方文档，常见入口是：

1. 打开左侧“技能”。
2. 点击“添加技能”，选择“查找技能”或进入“技能市场”。
3. 搜索 `md2wechat`。
4. 核对项目来源是 `geekjourneyx/md2wechat-skill`，再点击安装。
5. 在“已安装”中确认 `md2wechat` 已启用。

WorkBuddy 正在持续更新 SkillHub 和插件界面，所以不同版本可能出现这些差异：

- “技能”可能写成 `Skills`，“技能市场”可能写成 `SkillHub`。
- 较旧版本可能只有“插件”入口，或把 Skill 放在插件市场里。
- “添加技能”下可能显示“查找技能”“上传技能”或直接显示搜索框。

如果找不到上述入口，先更新 WorkBuddy，再按 [WorkBuddy 官方技能说明](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Skills-Market) 操作。也可以打开 [腾讯 SkillHub](https://skillhub.tencent.com/)，搜索 `md2wechat` 后把页面提供的安装指令粘贴到 WorkBuddy 对话框。安装第三方 Skill 前始终核对来源和权限。

> 只装 Skill 还不能运行 `md2wechat`。下一节还要安装 CLI。

## 三、用 npm 安装 CLI

NPM 安装要求 Node.js 18 或更高。可以把下面这段话直接发给 WorkBuddy：

```text
请在当前电脑检查 node --version 和 npm --version。Node.js 版本不低于 18 时，执行：
npm install -g @geekjourneyx/md2wechat
然后执行 md2wechat version --json 验证安装。不要初始化或修改任何配置，也不要上传图片、创建草稿或发布内容。请把每条命令的结果告诉我。
```

如果 `md2wechat version --json` 提示找不到命令，关闭并重新启动 WorkBuddy，让它重新读取 `PATH`，然后再试。更多平台和安装方式见 [安装指南](INSTALL.md)。

### 初始化配置不会覆盖旧文件

首次使用时可以执行：

```bash
md2wechat config init --json
```

默认文件是 `~/.config/md2wechat/config.yaml`。如果文件已经存在，命令会返回 `CONFIG_WRITE_FAILED`，不会覆盖旧配置。看到这个结果时应保留原文件，先阅读或备份，不要删除后重建。

下面这段是推荐的首次检查指令，可以直接发给 WorkBuddy：

```text
请只做 md2wechat 的本地安装与配置检查，按顺序执行：
1. 运行 md2wechat version --json。
2. 检查 ~/.config/md2wechat/config.yaml 是否存在；仅当文件不存在时运行 md2wechat config init --json。文件已存在时不得运行 init，也不得覆盖、删除或重写它。
3. 运行 md2wechat config validate --json。
4. 运行 md2wechat doctor --json。
5. 运行 md2wechat capabilities --json。
逐条汇报 JSON 的 code、status，以及返回的 readiness 或 blockers。到此停止，不要运行 upload_image、download_and_upload、create_draft、test-draft、convert --upload、convert --draft 或任何发布命令。
```

`config validate` 只证明配置可以加载和解析，`doctor` 也是本地只读体检；它们不会验证远端密钥、微信 IP 白名单或单篇文章是否可创建草稿。

## 四、认识三类凭证

| 凭证类别 | 什么时候需要 | 配置文件字段 | 常用环境变量 |
|----------|--------------|--------------|----------------|
| md2wechat API Key | API 模式预览、转换和高级排版 | `api.md2wechat_key` | `MD2WECHAT_API_KEY` |
| 微信公众号凭证 | 上传微信素材、创建微信草稿或图片消息 | `wechat.appid`、`wechat.secret` | `WECHAT_APPID`、`WECHAT_SECRET` |
| 图片 provider 凭证 | 让 CLI 直接请求 Seedream 等图片服务 | `api.image_key`，并配套 provider/base/model | `IMAGE_API_KEY`，并配套 `IMAGE_PROVIDER` 等 |

这三类凭证不能互相替代。只把 Markdown 转为 API 模式 HTML 时，不需要微信公众号 `AppID` / `AppSecret`；创建微信草稿时需要微信凭证和封面；使用 WorkBuddy 自带的 Image Gen 工具时，也不等于已经给 CLI 配置了图片 provider。

完整字段、优先级和多公众号说明见 [配置保姆级指南](CONFIG-WALKTHROUGH.md)。

## 五、配置微信凭证和 IP 白名单

只有准备上传微信素材或创建草稿时才做这一节。

1. 登录 [微信开发者平台](https://developers.weixin.qq.com/platform)，选择目标公众号，在“开发接口管理”中取得 `AppID` 和 `AppSecret`。
2. 把它们写入 `~/.config/md2wechat/config.yaml` 的 `wechat.appid` 和 `wechat.secret`。不要把 `AppSecret` 发到 WorkBuddy 对话里，也不要写进文章文件。
3. 在实际运行 `md2wechat` 的电脑上查询公网 IP。WorkBuddy 桌面端调用本地 CLI 时，白名单要放行这台电脑真正访问微信接口时使用的公网出口。
4. 回到微信开发者平台的“开发接口管理”，把该公网 IP 加入“IP 白名单”，保存后等待几分钟。
5. 重新运行 `md2wechat config validate --json` 和 `md2wechat doctor --json`。它们仍然只是本地检查；真正调用微信前还要对文章执行 `inspect --draft`。

家庭宽带、公司网络或 VPN 的公网 IP 可能变化。固定出口、高级 API 代理、多账号和微信错误的完整处理步骤见 [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)，不要在本页自行拼接代理地址。

## 六、可选：配置 Seedream 5.0 Pro

如果你希望 `md2wechat` CLI 直接请求火山引擎生成图片，先在火山引擎控制台开通 Seedream 模型，再把最小配置写入现有配置文件：

```yaml
api:
  image_provider: "volcengine"
  image_key: "你的火山引擎 API Key"
  image_base_url: "https://ark.cn-beijing.volces.com/api/v3"
  image_model: "doubao-seedream-5-0-pro-260628"
  image_size: "2K"
```

先用 discovery 确认当前 CLI 仍支持该 provider 和模型：

```bash
md2wechat providers show volcengine --json
```

如果返回 `ModelNotOpen`，说明火山引擎账号还没有开通目标模型，不是文章或提示词错误。模型、尺寸、开通入口和其他 provider 见 [图片服务配置](IMAGE_PROVISIONERS.md)。

如果 WorkBuddy 当前会话本身提供 Image Gen 工具，你也可以不配置图片 provider，改用 `generate_cover --plan --json` 让 CLI 只生成计划，再由 WorkBuddy 生成本地图片。这个流程的边界见 [Agent 图片计划模式](AGENT_IMAGE_GEN.md)。

## 七、第一次转换：只生成本地 HTML

先在工作目录准备 `article.md`。第一次不要上传图片、不要创建草稿，也不要发布。API 模式需要已经配置 `MD2WECHAT_API_KEY` 或 `api.md2wechat_key`。

把下面这段话直接发给 WorkBuddy：

```text
请在当前工作目录对 article.md 做第一次无发布转换，并严格按顺序执行：
1. md2wechat inspect article.md --mode api --json
2. 只有 data.readiness.targets.convert 显示可执行，且 data.readiness.blockers 没有阻断 convert 时，运行 md2wechat preview article.md --mode api --output workbuddy-preview.html --json
3. 只有 preview 成功并返回本次输出文件时，运行 md2wechat convert article.md --mode api --output article.html --json
请汇报每一步的 code、status、readiness、blockers 和实际输出路径，然后停止。不要添加 --upload、--draft，不要上传图片，不要创建微信草稿，不要发布，也不要改写 article.md。
```

这里的 `workbuddy-preview.html` 用来查看效果，`article.html` 是转换产物。`preview` 和 `convert` 会调用 md2wechat 的 API 转换服务，但不会因为这组命令而调用微信发布接口。

这次成功只证明 API 模式转换链可用，**不会验证微信公众号 `AppID` / `AppSecret`，也不会验证微信 IP 白名单**。

## 八、明确确认后创建微信草稿

创建草稿会调用微信接口并改变远端状态。先检查，看到结果后再由你明确确认，不能把检查和执行合成一步。

先让 WorkBuddy 只运行：

```bash
md2wechat inspect article.md --draft --cover cover.jpg --json
```

确认 `data.readiness.targets.draft` 显示可执行，且 `data.readiness.blockers` 没有阻断 draft，并检查标题、摘要、正文图片和封面。然后你可以回复：

```text
我已检查 readiness，同意现在创建微信草稿。只执行下面这一条命令，执行后汇报 JSON 结果，不要继续发布：
md2wechat convert article.md --draft --cover cover.jpg --json
```

如果使用命名公众号账号，检查和执行必须带完全相同的账号参数，例如：

```bash
md2wechat inspect article.md --draft --cover cover.jpg --wechat-account main --json
md2wechat convert article.md --draft --cover cover.jpg --wechat-account main --json
```

不要用 `main` 检查后再用另一个账号执行。创建微信草稿不等于群发或正式发布；本教程不包含群发操作。

## 九、常见问题

### Skill 已安装，但提示 `md2wechat: command not found`

Skill 不包含可执行 CLI。回到第三节执行 npm 全局安装；安装后重启 WorkBuddy，让新的全局 npm 命令路径生效。

### npm 安装出现权限错误或镜像 `404`

不要让 WorkBuddy 用不明脚本提升系统权限。先按 [安装指南](INSTALL.md) 修复 npm 全局目录；如果 `npmmirror` 的新版本尚未同步，使用文档中的 npm 官方 registry 命令。

### `config init` 报文件已经存在

这是防覆盖保护，不是配置丢失。保留 `~/.config/md2wechat/config.yaml`，用 `config validate --json` 和 `doctor --json` 检查。

### API 预览或转换提示缺少 API Key

API 模式需要 md2wechat API Key。它不是微信 `AppSecret`，也不是图片 provider key。按 [配置保姆级指南](CONFIG-WALKTHROUGH.md) 配置 `api.md2wechat_key` 后重试。

### 草稿提示缺封面

检查命令和执行命令都要带同一份 `--cover cover.jpg`，并确认文件在当前工作目录。也可以按命令帮助使用已有的永久 `--cover-media-id`；此时 `inspect` 与 `convert` 必须使用同一个 `--cover-media-id` 值，不要在检查与执行之间切换目标。

### 微信返回 `ip not in whitelist`

白名单里必须是实际执行 CLI 的机器访问微信时使用的公网出口 IP。VPN、公司网关或家庭宽带变化后要重新确认；详见 [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)。

### Seedream 返回 `ModelNotOpen`

去火山引擎控制台开通 `doubao-seedream-5-0-pro-260628` 对应的 Seedream 服务，或从 `providers show volcengine --json` 中选择账号已经开通的模型。

### WorkBuddy 里找不到“技能”或“添加技能”

不同版本可能显示“技能”“Skills”“SkillHub”或“插件”。如果当前版本提供“检查更新”入口，先更新，再按官方技能说明核对当前入口，不要凭相似名称安装来源不明的 Skill。

## 十、相关文档

- [安装指南](INSTALL.md)
- [配置保姆级指南](CONFIG-WALKTHROUGH.md)
- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)
- [图片服务配置](IMAGE_PROVISIONERS.md)
- [Agent 图片计划模式](AGENT_IMAGE_GEN.md)
- [能力发现与 Prompt Catalog](DISCOVERY.md)
- [常见问题](FAQ.md)
- [WorkBuddy 官方下载页](https://www.codebuddy.cn/work/)
- [WorkBuddy 官方技能说明](https://www.codebuddy.cn/docs/workbuddy/From-Beginner-to-Expert-Guide/Function-Description/Skills-Market)
