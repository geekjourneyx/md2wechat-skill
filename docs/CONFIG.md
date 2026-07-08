# 配置指南

如果你要从零配置、迁移多公众号、或让 Agent 按步骤排查配置，先看 [配置保姆级指南](CONFIG-WALKTHROUGH.md)。本文更偏参考手册，解释配置字段、优先级和高级选项。

这份文档解决 4 个最常见的问题：

1. 配置文件在哪里
2. 默认 API 域名在哪里改
3. Agent 应该先看哪里
4. 哪些功能分别需要哪些凭证

如果你现在卡在：

- 不知道 AppID / AppSecret 去哪拿
- 不知道微信 IP 白名单在哪配
- 明明配了凭证但还是 `ip not in whitelist`

先看：

- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)

如果你只想先跑通主路径，先看下面这 3 步。

## 3 步完成基础配置

### 1. 生成示例配置

```bash
md2wechat config init
```

默认会生成到：

```text
~/.config/md2wechat/config.yaml
```

你也可以显式指定输出位置：

```bash
md2wechat config init ./md2wechat.yaml
```

### 2. 打开配置文件，先填最小必需项

```yaml
wechat:
  appid: "你的微信公众号 AppID"
  secret: "你的微信公众号 Secret"

api:
  md2wechat_key: "你的 md2wechat API Key"
  md2wechat_base_url: "https://www.md2wechat.cn"
  convert_mode: "api"
  default_theme: "default"
```

### 3. 验证当前配置

```bash
md2wechat config validate
md2wechat config show --format json
md2wechat doctor --json
```

`config validate` 只验证配置能否加载和解析。`doctor` 是本地只读体检，会继续检查默认 API 转换是否就绪、默认主题是否兼容、layout catalog 是否可用，以及草稿凭证是否存在；它不做 live auth、不上传、不创建草稿。

---

## 多公众号配置

单账号仍然是默认主路径：

```yaml
wechat:
  appid: "你的微信公众号 AppID"
  secret: "你的微信公众号 Secret"
```

如果你购买了高级 API 服务并需要管理多个公众号，可以在同一份配置里增加命名账号：

```yaml
wechat:
  default_account: main
  accounts:
    main:
      appid: "wx..."
      secret: "..."
    client-a:
      appid: "wx..."
      secret: "..."
```

命名账号名称只支持小写字母、数字、`_` 和 `-`，例如 `main`、`client-a`、`brand_2026`。

会调用微信接口的命令按下面顺序选择账号：

1. `--wechat-account`
2. `WECHAT_ACCOUNT`
3. `wechat.default_account`
4. 直接配置的 `wechat.appid` / `wechat.secret`
5. 唯一的命名账号

命名账号执行上传、生成并上传图片、创建草稿或图片消息时，需要有效的 `MD2WECHAT_API_KEY`。CLI 会在副作用发生前调用 `HEAD /api/auth/validate` 校验 API key。`config show`、`config validate`、`doctor` 和 `config wechat-accounts` 仍然是本地只读命令，不做网络校验。

查看本地已配置的公众号账号：

```bash
md2wechat config wechat-accounts --json
```

该命令不会输出 secret，连掩码后的 secret 也不会输出。

---

## Agent 和用户应该先看哪里

如果你不知道去哪改配置，按这个顺序找：

1. `~/.config/md2wechat/config.yaml`
2. 环境变量
3. 当前目录下的 `md2wechat.yaml` / `md2wechat.yml` / `md2wechat.json`

对 Agent 来说，**默认应该优先检查 `~/.config/md2wechat/config.yaml`**。
如果用户说“把 API 域名改成备用域名”“切换图片服务”“检查当前配置”，先运行：

```bash
md2wechat config show --format json
```

这样可以直接看到当前生效的：

- `config_file`
- `md2wechat_base_url`
- `image_provider`
- `image_api_base`
- `default_convert_mode`

注意这里看到的是 **`config show --format json` 的扁平输出字段名**，不是配置文件里的嵌套 YAML 键名。

例如：

- 配置文件里写的是 `api.image_base_url`
- `config show --format json` 里看到的是 `image_api_base`

---

## 默认 API 域名在哪里改

项目当前默认值是：

```text
https://www.md2wechat.cn
```

它**不是写死不可改**。你有两种常用改法。

### 方式一：改配置文件

编辑 `~/.config/md2wechat/config.yaml`：

```yaml
api:
  md2wechat_base_url: "https://www.md2wechat.cn"
```

如果你要切到备用域名：

```yaml
api:
  md2wechat_base_url: "https://md2wechat.app"
```

### 方式二：用环境变量临时覆盖

```bash
export MD2WECHAT_BASE_URL="https://md2wechat.app"
```

环境变量优先级高于配置文件，适合：

- 临时切换备用域名
- CI / Agent 自动化
- 不想修改全局配置文件的场景

### 关于默认转换模式

当前 CLI 的默认行为是固定的：

- 不传 `--mode` 时，`md2wechat convert ...` 始终默认走 `api`
- 只有显式传入 `--mode ai` 时，才会走 AI 模式

也就是说，下面这个命令：

```bash
md2wechat convert article.md
```

当前一定等价于：

```bash
md2wechat convert article.md --mode api
```

所以如果用户没有填写配置，或者没有显式传 `--mode`，默认也是 `api`。

`api.convert_mode` / `CONVERT_MODE` 当前主要用于配置展示、校验和兼容字段；**不会覆盖 `convert` 命令在未传 `--mode` 时的默认行为**。

---

## 内置资产

当前仓库把官方默认 `themes` 和默认 `writer style` 随二进制一起提供。
这意味着即使 Agent 服务器上没有仓库目录，默认主题和默认写作风格也应该可用。

### 主题加载顺序

`themes` 的优先级从高到低如下：

1. `MD2WECHAT_THEMES_DIR`
2. 当前项目目录下的 `themes/`
3. `~/.config/md2wechat/themes/`
4. 二进制内置的官方默认 themes

同名主题以前面的来源覆盖后面的来源。

### 写作风格加载顺序

`writers` 的优先级从高到低如下：

1. `MD2WECHAT_WRITERS_DIR`
2. 当前项目目录下的 `writers/`
3. `~/.config/md2wechat/writers/`
4. `~/.md2wechat-writers/`
5. 二进制内置的默认 writer style

同名写作风格同样以前面的来源覆盖后面的来源。

### 什么时候改哪里

如果你想：

- 仅当前项目生效，放到项目目录
- 所有项目都生效，放到 `~/.config/md2wechat/...`
- Agent 服务器显式指定，设置 `MD2WECHAT_THEMES_DIR` 或 `MD2WECHAT_WRITERS_DIR`
- 保持官方默认不变，直接用内置资产

---

## 配置文件搜索顺序

程序会按以下顺序查找配置文件：

1. `~/.config/md2wechat/config.yaml`
2. `~/.md2wechat.yaml`
3. `~/.md2wechat.yml`
4. `./md2wechat.yaml`
5. `./md2wechat.yml`
6. `./md2wechat.json`
7. `./.md2wechat.yaml`
8. `./.md2wechat.yml`
9. `./.md2wechat.json`

实践上建议：

- 全局默认配置放 `~/.config/md2wechat/config.yaml`
- 项目特殊配置再放当前目录

---

## 完整示例配置

仓库里提供了一份可直接参考的示例：

- [config.yaml.example](examples/config.yaml.example)

完整示例：

```yaml
wechat:
  appid: "your_wechat_appid"
  secret: "your_wechat_secret"
  # Advanced API service only. Paste the full proxy URL provided by md2wechat.
  # proxy_url: "https://wechat-egress-url-provided-by-md2wechat.example"

api:
  md2wechat_key: "your_md2wechat_api_key"
  md2wechat_base_url: "https://www.md2wechat.cn"
  image_key: "your_image_api_key"
  image_base_url: "https://ark.cn-beijing.volces.com/api/v3"
  image_provider: "volcengine"
  image_model: "doubao-seedream-5-0-pro-260628"
  image_size: "2K"
  convert_mode: "api"
  default_theme: "default"
  background_type: "none"
  http_timeout: 30

image:
  compress: true
  max_width: 1920
  max_size_mb: 5
```

## 三套命名要分清

当前最容易混淆的是：同一个配置项会同时出现在 3 个地方，但名字不完全一样。

### 1. 配置文件字段名

这是你在 `config.yaml` 里实际填写的名字，例如：

- `wechat.appid`
- `api.md2wechat_key`
- `api.image_base_url`
- `api.background_type`

### 2. 环境变量名

这是终端或 CI 里覆盖配置时使用的名字，例如：

- `WECHAT_APPID`
- `MD2WECHAT_API_KEY`
- `IMAGE_API_BASE`
- `DEFAULT_BACKGROUND_TYPE`

### 3. `config show --format json` 输出字段名

这是 CLI 为了更稳定的 machine-readable 输出而提供的扁平字段，例如：

- `wechat_appid`
- `md2wechat_api_key`
- `image_api_base`
- `default_background_type`

所以如果你是在：

- 改配置文件：用 `api.image_base_url`
- 查环境变量：看 `IMAGE_API_BASE`
- 解析 `config show --format json`：看 `image_api_base`

不要把这三套名字混成一个层次。

---

## 配置项说明

### 微信配置

| 配置项 | 必需 | 说明 |
|--------|------|------|
| `wechat.appid` | 创建草稿、上传图片时需要 | 微信公众号 AppID |
| `wechat.secret` | 创建草稿、上传图片时需要 | 微信公众号 Secret |
| `wechat.proxy_url` | 否 | 高级版 API 固定出口能力：仅微信上传、草稿和图片消息副作用使用的 HTTP/HTTPS 前向代理 |

`wechat.proxy_url` 是高级版 API 服务的固定出口能力，用来解决运行环境公网 IP 动态变化导致微信白名单反复失效的问题。开通后，服务侧会提供两项信息：

- 完整的 `proxy_url`，直接粘贴到配置文件或 `WECHAT_PROXY_URL`
- 稳定的微信接口出口 IP，填写到微信后台 `IP 白名单`

`wechat.proxy_url` 只影响微信 API 副作用，不影响 API 排版、图片生成 provider、主题/提示词发现或普通转换。启用后，上传、建草稿和图片消息发送前需要有效的 `MD2WECHAT_API_KEY`。

不要自行拼接代理主机、端口或部署形态；以高级版 API 服务提供的完整 URL 为准。公开配置文档不约定代理端口。需要固定出口能力或企业私有化方案时，请联系作者进行 `API咨询`。`HTTPS_PROXY` 只作为全局代理兜底背景理解，优先使用 `wechat.proxy_url` / `WECHAT_PROXY_URL`，避免把非微信流量一起代理。

### API 转换配置

| 配置项 | 必需 | 说明 | 默认值 |
|--------|------|------|--------|
| `api.md2wechat_key` | API 模式需要 | md2wechat API Key | - |
| `api.md2wechat_base_url` | 否 | 排版 API 域名 | `https://www.md2wechat.cn` |
| `api.convert_mode` | 否 | 默认转换模式 | `api` |
| `api.default_theme` | 否 | 默认主题 | `default` |
| `api.background_type` | 否 | 背景类型 | `none` |
| `api.http_timeout` | 否 | HTTP 超时秒数 | `30` |

### 图片生成配置

| 配置项 | 必需 | 说明 | 默认值 |
|--------|------|------|--------|
| `api.image_key` | AI 图片时需要 | 图片生成 API Key | - |
| `api.image_provider` | 否 | 图片服务提供方 | `openai` |
| `api.image_base_url` | 否 | 图片服务地址 | `https://api.openai.com/v1` |
| `api.image_model` | 否 | 图片模型 | `gpt-image-2` |
| `api.image_size` | 否 | 默认图片执行尺寸/宽高比 | 跟随当前 provider，例如 `openai=auto`、`volcengine=2K` |

当前内置 provider：`openai`、`tuzi`、`modelscope` (`ms`)、`openrouter` (`or`)、`gemini` (`google`)、`volcengine` (`volc`)。

### 图片处理配置

| 配置项 | 必需 | 说明 | 默认值 |
|--------|------|------|--------|
| `image.compress` | 否 | 是否自动压缩 | `true` |
| `image.max_width` | 否 | 最大宽度 | `1920` |
| `image.max_size_mb` | 否 | 最大大小（MB） | `5` |

---

## 环境变量对照表

| 环境变量 | 对应配置项 |
|----------|------------|
| `WECHAT_APPID` | `wechat.appid` |
| `WECHAT_SECRET` | `wechat.secret` |
| `WECHAT_ACCOUNT` | 命名账号选择 |
| `WECHAT_PROXY_URL` | `wechat.proxy_url` |
| `MD2WECHAT_API_KEY` | `api.md2wechat_key` |
| `MD2WECHAT_BASE_URL` | `api.md2wechat_base_url` |
| `IMAGE_API_KEY` | `api.image_key` |
| `IMAGE_API_BASE` | `api.image_base_url` |
| `IMAGE_PROVIDER` | `api.image_provider` |
| `IMAGE_MODEL` | `api.image_model` |
| `IMAGE_SIZE` | `api.image_size` |
| `CONVERT_MODE` | `api.convert_mode` |
| `DEFAULT_THEME` | `api.default_theme` |
| `DEFAULT_BACKGROUND_TYPE` | `api.background_type` |
| `HTTP_TIMEOUT` | `api.http_timeout` |
| `COMPRESS_IMAGES` | `image.compress` |
| `MAX_IMAGE_WIDTH` | `image.max_width` |
| `MAX_IMAGE_SIZE` | `image.max_size_mb` |
| `MD2WECHAT_THEMES_DIR` | `themes` 覆盖目录 |
| `MD2WECHAT_WRITERS_DIR` | `writers` 覆盖目录 |

图片生成相关命令还支持 `--model`，用于单次覆盖当前调用的图片模型。优先级顺序为：

1. `--model`
2. `IMAGE_MODEL`
3. `api.image_model`
4. provider 默认模型

## `config show --format json` 常见字段对照

如果你是在排查 Agent / 脚本实际读到的配置，最常见的不是 YAML 字段，而是下面这些扁平 key：

| `config show --format json` 字段 | 对应配置文件字段 |
|---|---|
| `wechat_appid` | `wechat.appid` |
| `wechat_secret` | `wechat.secret` |
| `wechat_proxy_url` | `wechat.proxy_url` |
| `wechat_account` | 当前命名账号，直接账号为空字符串 |
| `md2wechat_api_key` | `api.md2wechat_key` |
| `md2wechat_base_url` | `api.md2wechat_base_url` |
| `image_api_key` | `api.image_key` |
| `image_api_base` | `api.image_base_url` |
| `image_provider` | `api.image_provider` |
| `image_model` | `api.image_model` |
| `image_size` | `api.image_size` |
| `default_convert_mode` | `api.convert_mode` |
| `default_theme` | `api.default_theme` |
| `default_background_type` | `api.background_type` |
| `compress_images` | `image.compress` |
| `max_image_width` | `image.max_width` |
| `max_image_size_mb` | `image.max_size_mb` |
| `http_timeout` | `api.http_timeout` |
| `config_file` | 当前实际命中的配置文件路径 |

---

## 常见场景怎么配

### 只预览，不创建草稿

最小需要：

```yaml
api:
  md2wechat_key: "your_md2wechat_api_key"
  md2wechat_base_url: "https://www.md2wechat.cn"
  convert_mode: "api"
```

### 需要上传图片和创建草稿

最小需要：

```yaml
wechat:
  appid: "your_wechat_appid"
  secret: "your_wechat_secret"

api:
  md2wechat_key: "your_md2wechat_api_key"
```

### 需要 AI 图片生成

最小需要：

```yaml
wechat:
  appid: "your_wechat_appid"
  secret: "your_wechat_secret"

api:
  image_key: "your-ark-api-key"
  image_provider: "volcengine"
  image_model: "seedream-3-0"
  image_size: "2K"
```

补充说明：

- `api.image_size` / `IMAGE_SIZE` 控制的是实际发给图片 provider 的默认执行尺寸
- `generate_image --size ...` 会覆盖配置文件里的 `api.image_size`
- 图片 prompt 里的 `default_aspect_ratio` 是 preset 的语义默认画幅，用于渲染 prompt 与默认视觉比例
- 对于 Gemini / OpenRouter 这类支持比例格式的 provider，`api.image_size` 可以直接写成 `16:9`、`3:4`、`21:9`
- 对于 Volcengine Ark 当前接入，`api.image_size` 使用尺寸等级，例如 `2K`、`3K`；如果省略，当前默认值是 `2K`
- `api.image_base_url` 对 OpenAI、TuZi、ModelScope、OpenRouter、Volcengine 生效；Gemini 直连模式当前固定走官方 Go SDK backend，不读取该配置

---

## 配置优先级

优先级从高到低：

```text
命令行参数 > 环境变量 > 配置文件 > 默认值
```

举例：

1. 配置文件里写了：

```yaml
api:
  md2wechat_base_url: "https://www.md2wechat.cn"
```

2. 当前终端又执行了：

```bash
export MD2WECHAT_BASE_URL="https://md2wechat.app"
```

最终生效的是：

```text
https://md2wechat.app
```

---

## 自检命令

```bash
md2wechat config init
md2wechat config show --format json
md2wechat config validate
```

推荐排查顺序：

1. 先看 `config_file` 指向哪个文件
2. 再看 `md2wechat_base_url` 是否真是你想要的域名
3. 再看 `image_provider` / `image_api_base` 是否匹配
   这里的 `image_api_base` 是 `config show --format json` 的输出字段；配置文件里对应的是 `api.image_base_url`
4. 最后检查环境变量是否把文件里的值覆盖掉了

---

---

## Brand Profile

Brand Profile 是 Agent 读取的品牌与风格提示文件，**CLI 不解析此文件**。

与 CLI 运行时配置（`~/.config/md2wechat/config.yaml`）不同，Brand Profile 专门为 Agent 设计，用于记录内容生成的风格偏好和品牌上下文。

### 快速开始

```bash
# 初始化 Brand Profile（幂等操作，文件存在时不覆盖）
md2wechat brand init

# 查看当前 Brand Profile
md2wechat brand show
md2wechat brand show --json
```

Brand Profile 位置：

```text
~/.config/md2wechat/brand.md
```

### Markdown 格式说明

Brand Profile 使用 **Markdown 格式**，而不是 YAML。这让你可以用完全自然的语言书写品牌风格和偏好。

以下是一份 Markdown 模板示例。它是自然语言 prompt，不是 CLI schema；字段名可以修改、删除或扩展。

```markdown
# md2wechat Brand Profile

## 基本信息

**名字 / 品牌名**：极客杰尼

**简介**：AI 应用开发者，记录 AI 工具、内容系统和独立产品实践。

---

## 语气与风格

**我的风格**：
犀利实用，第一人称。直接说结论，然后给证据。
像在和朋友聊干货，不废话，不升华，不说"希望对你有帮助"。

**我要避免的表达**：
- 过多 emoji（最多 1-2 个）
- 空泛鸡汤（"在这个充满变化的时代..."）
- 过度营销词汇（"革命性"、"颠覆性"）
- 被动语态（"被认为"、"据悉"）

---

## 文章开头偏好

**我的偏好**：verdict_first（先结论）

我喜欢开门见山，第一段就给出核心判断。
例如："这个工具我用了三个月，值得推荐，原因有三。"

---

## 排版偏好

- 模块少而准，不堆装饰
- 通常只放一个 CTA
- 观点文章可以用金句，但不要连续堆引用
- 除非文章特别长，否则不要用 TOC

---

## 默认 CTA（行动引导）

**标题**：如果这篇对你有启发
**正文**：欢迎关注，我在持续记录 AI 工具和独立开发实践。每周更新，不灌水。
**行动**：关注 / 转发给有需要的朋友

---

## 作者卡片

**名字**：极客杰尼
**头衔**：AI 应用开发者 / 独立开发者
**简介**：记录 AI 工具、内容系统和独立产品实践。关注从 idea 到 MVP 的完整路径。

---

## 风格参考（可选）

我喜欢的表达方式：
- 先给结论，再给证据
- 多写具体使用细节，少写抽象判断

如果你写了本地路径，Agent 可以在用户允许且路径可读时参考；这不是 CLI 解析功能。
```

### 为什么是 Markdown 而不是 YAML？

品牌风格本质上是语言性的，而不是结构化的。Markdown 允许你用**完全自然的语言**描述风格偏好，Agent 可以直接理解这些自然语言描述。

**越具体，Agent 越能准确还原你的风格。**

### Agent 读取方式

Agent 读取 Brand Profile 时：

```python
import os

brand_path = os.path.expanduser("~/.config/md2wechat/brand.md")
brand_content = ""
if os.path.exists(brand_path):
    with open(brand_path) as f:
        brand_content = f.read()

# brand_content contains the full Markdown prompt.
# The agent uses it as context for layout decisions.
```

### JSON 响应格式

`md2wechat brand show --json` 返回：

```json
{
  "success": true,
  "code": "BRAND_SHOWN",
  "data": {
    "path": "~/.config/md2wechat/brand.md",
    "content": "# md2wechat Brand Profile\n\n## 基本信息\n..."
  }
}
```

注意：`data.content` 是完整的 Markdown 文本，而不是解析后的结构。

### 降级行为与容错

1. **文件不存在**：Agent 继续工作，不报错；任务开始前最多提示一次，然后使用系统默认风格。

2. **文件不可读（权限问题）**：
   - `md2wechat brand show` 返回 `BRAND_READ_FAILED`
   - Agent 应使用默认风格并通知用户

3. **Markdown 无语法错误**：Markdown 是自由格式文本，不存在"格式错误"。只要文件可读，Agent 就能使用。

### 与 CLI 运行时配置的区别

| 配置文件 | 位置 | 用途 | 解析方 | 必需 |
|---------|------|------|--------|------|
| CLI 运行时配置 | `~/.config/md2wechat/config.yaml` | API Keys、Provider、主题 | CLI | API 转换、图片生成或创建草稿时按需使用 |
| Brand Profile | `~/.config/md2wechat/brand.md` | 内容风格、排版偏好、品牌上下文 | Agent | 可选（无则使用默认） |

**CLI 运行时配置** 典型场景：切换图片 Provider、配置 WeChat AppID、选择主题。

**Brand Profile** 典型场景：Agent 生成内容时遵守品牌约束、追踪作者信息、统一语气风格。

### 常见场景

#### 只初始化，保持最小配置

```bash
md2wechat brand init
# 编辑 ~/.config/md2wechat/brand.md，填入基本信息和语气风格即可
```

结果：Agent 会尊重你的品牌名和语气，但使用所有其他默认值。

#### Agent 读取并应用 Brand Profile

Agent 应该：

```bash
# 1. 检查 Brand Profile 是否存在
md2wechat brand show --json

# 2. 如果存在，读取 data.content 作为完整上下文
# 3. 生成内容时应用其中的风格偏好、约束、CTA 和作者信息
```

---

## 相关文档

- [新手快速开始](QUICKSTART.md)
- [安装指南](INSTALL.md)
- [图片服务配置](IMAGE_PROVISIONERS.md)
- [真实烟雾测试记录](SMOKE.md)
- [内置资产](#内置资产)
