# 新手快速开始

这份指南只保留当前仓库已经支持、并且文档路径稳定的主流程。

如果你需要完整安装说明，请先看 [安装指南](INSTALL.md)。

## 5 分钟主路径

### 1. 安装

推荐使用固定版本 release 资产：

mac 用户优先：

```bash
brew install geekjourneyx/tap/md2wechat
```

如果你不用 Homebrew，再执行：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.1.0/install.sh | bash
```

Windows PowerShell：

```powershell
$env:MD2WECHAT_RELEASE_BASE_URL = "https://github.com/geekjourneyx/md2wechat-skill/releases/download/v3.1.0"
iex ((New-Object System.Net.WebClient).DownloadString("$env:MD2WECHAT_RELEASE_BASE_URL/install.ps1"))
```

安装后验证：

```bash
md2wechat version --json
md2wechat skills read md2wechat --json
```

`skills read` 会读取当前二进制内置的 Agent SOP。它不是外部 skill 安装步骤，而是确认当前 CLI 自带的操作协议和可执行文件版本一致。

### 2. 初始化配置

```bash
md2wechat config init
```

默认配置文件位置：

```text
~/.config/md2wechat/config.yaml
```

完整配置流程，包括单公众号、多公众号和常见错误，见 [配置保姆级指南](CONFIG-WALKTHROUGH.md)。

如果你要创建微信草稿，至少需要配置：

- `wechat.appid`
- `wechat.secret`
- `api.md2wechat_key`

如果你需要切换 API 域名，在这个文件里修改：

```yaml
api:
  md2wechat_base_url: "https://www.md2wechat.cn"
```

备用域名可改为：

```yaml
api:
  md2wechat_base_url: "https://md2wechat.app"
```

默认主题和默认写作风格已经随二进制内置，不需要额外拷贝 `themes/` 或 `writers/` 目录。
如果你要自定义它们，按优先级放到项目目录、`~/.config/md2wechat/...`，或者显式设置 `MD2WECHAT_THEMES_DIR` / `MD2WECHAT_WRITERS_DIR`。

### 3. 预览 Markdown

```bash
md2wechat inspect article.md
md2wechat preview article.md
md2wechat convert article.md --preview
```

建议顺序：

1. 先跑 `inspect --json`，确认最终标题、摘要、H1 风险；需要上传或草稿时读取 `data.readiness.targets/blockers`
2. 再跑 `preview`，拿到本地 HTML 预览文件
3. 最后再执行 `convert` / `--draft`

### 4. 创建微信草稿

创建草稿时需要显式提供封面：

```bash
md2wechat convert article.md --draft --cover cover.jpg
```

### 5. 使用 AI 模式

AI 模式会生成可交给外部 AI 的结构化输出：

```bash
md2wechat convert article.md --mode ai --theme autumn-warm --json
```

如果你更关注稳定性和直接转换，优先使用 API 模式。

## 两条常用路径

### 图文文章

```bash
md2wechat convert article.md --preview
md2wechat convert article.md -o article.html
md2wechat convert article.md --draft --cover cover.jpg
md2wechat convert article.md --title "新标题" --author "作者名" --digest "摘要"
```

元数据优先级：

- 标题：`--title` -> `frontmatter.title` -> 正文首个 Markdown 标题 -> `未命名文章`
- 作者：`--author` -> `frontmatter.author`
- 摘要：`--digest` -> `frontmatter.digest` -> `frontmatter.summary` -> `frontmatter.description`

限制：

- 标题最多 32 个字符
- 作者最多 16 个字符
- 摘要最多 128 个字符

### 图片帖子（小绿书 / newspic）

```bash
md2wechat create_image_post --title "Weekend Trip" --images a.jpg,b.jpg
```

预览：

```bash
md2wechat create_image_post --title "Weekend Trip" --images a.jpg,b.jpg --dry-run --json
```

## 建议阅读顺序

1. [安装指南](INSTALL.md)
2. [完整使用说明](USAGE.md)
3. [高级排版模块教程](LAYOUT.md) ← API 模式专属功能
4. [故障排查](TROUBLESHOOTING.md)
5. [架构说明](ARCHITECTURE.md)

## 不再作为主路径的内容

以下内容不再作为推荐主路径：

- `latest` 下载链接
- `main` 分支上的原始安装脚本
- 不带 `--cover` 的 `convert --draft`
- 过时的“命令层直接编排所有业务”描述
