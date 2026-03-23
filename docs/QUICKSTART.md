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
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3/install.sh | bash
```

Windows PowerShell：

```powershell
$env:MD2WECHAT_RELEASE_BASE_URL = "https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.0.3"
iex ((New-Object System.Net.WebClient).DownloadString("$env:MD2WECHAT_RELEASE_BASE_URL/install.ps1"))
```

安装后验证：

```bash
md2wechat version --json
```

### 2. 初始化配置

```bash
md2wechat config init
```

默认配置文件位置：

```text
~/.config/md2wechat/config.yaml
```

如果你要创建微信草稿，至少需要配置：

- `wechat.appid`
- `wechat.secret`
- `api.md2wechat_key`

如果你需要切换 API 域名，在这个文件里修改：

```yaml
api:
```

备用域名可改为：

```yaml
api:
```

默认主题和默认写作风格已经随二进制内置，不需要额外拷贝 `themes/` 或 `writers/` 目录。
如果你要自定义它们，按优先级放到项目目录、`~/.config/md2wechat/...`，或者显式设置 `MD2WECHAT_THEMES_DIR` / `MD2WECHAT_WRITERS_DIR`。

### 3. 预览 Markdown

```bash
md2wechat convert article.md --preview
```

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
```

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
3. [故障排查](TROUBLESHOOTING.md)
4. [架构说明](ARCHITECTURE.md)

## 不再作为主路径的内容

以下内容不再作为推荐主路径：

- `latest` 下载链接
- `main` 分支上的原始安装脚本
- 不带 `--cover` 的 `convert --draft`
- 过时的“命令层直接编排所有业务”描述
