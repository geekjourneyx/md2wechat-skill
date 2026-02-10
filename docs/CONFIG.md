# 配置指南

本文档详细说明 md2wechat 的各种配置方式。

## 目录

- [快速配置](#快速配置)
- [获取微信凭证](#获取微信凭证)
- [配置方式](#配置方式)
- [配置文件详解](#配置文件详解)
- [环境变量](#环境变量)
- [配置优先级](#配置优先级)

---

## 快速配置

最简单的配置方式是使用交互式命令：

```bash
md2wechat config init
```

这会创建一个 `md2wechat.yaml` 文件，编辑它填入你的凭证：

```yaml
wechat:
  appid: "wx1234567890abcdef"
  secret: "your_secret_here"

api:
  md2wechat_key: ""           # 可选
  convert_mode: "api"
  default_theme: "default"

image:
  compress: true
  max_width: 1920
  max_size_mb: 5
```

---

## 获取微信凭证

### 步骤

1. **登录微信开发者平台**

   访问 [developers.weixin.qq.com/platform](https://developers.weixin.qq.com/platform)

   > 注：开发接口管理已于 2025 年 12 月迁移至微信开发者平台

2. **进入开发接口管理**

   选择你的公众号 → 开发接口管理

3. **获取 AppID 和 AppSecret**

   - AppID：直接复制
   - AppSecret：点击"重置"后获取（请妥善保管）

4. **配置 IP 白名单**（可选）

   如果使用 API 模式，需要将服务器 IP 加入白名单

---

## 配置方式

md2wechat 支持三种配置方式，按优先级从高到低：

1. **命令行参数** - 临时覆盖
2. **环境变量** - 机器级别配置
3. **配置文件** - 项目级别配置

### 配置文件搜索顺序

程序会在以下位置查找配置文件：

```
./md2wechat.yaml          # 当前目录（优先级最高）
./md2wechat.yml
./md2wechat.json
~/.md2wechat.yaml         # 用户主目录
~/.config/md2wechat/config.yaml
```

---

## 配置文件详解

### 完整配置示例

```yaml
# 微信公众号配置
wechat:
  appid: "wx1234567890abcdef"        # 必填：公众号 AppID
  secret: "your_app_secret"           # 必填：公众号 AppSecret

# API 配置
api:
  md2wechat_key: "your_md2wechat_key"  # 可选：md2wechat.cn API Key
  image_key: ""                         # 可选：图片生成 API Key
  image_base_url: "https://api.openai.com/v1"  # 图片 API 地址
  convert_mode: "api"                   # 转换模式：api 或 ai
  default_theme: "default"              # 默认主题
  http_timeout: 30                      # HTTP 超时时间（秒）

# 图片处理配置
image:
  compress: true        # 是否自动压缩图片
  max_width: 1920       # 图片最大宽度（像素）
  max_size_mb: 5        # 图片最大大小（MB）
```

### 配置项说明

#### 微信配置 (wechat)

| 配置项 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `appid` | 是 | 微信公众号 AppID | `wx1234567890abcdef` |
| `secret` | 是 | 微信公众号 AppSecret | `a1b2c3d4e5f6g7h8i9j0` |

#### API 配置 (api)

| 配置项 | 必填 | 说明 | 默认值 |
|--------|------|------|--------|
| `md2wechat_key` | 否* | md2wechat.cn API Key | - |
| `md2wechat_base_url` | 否 | md2wechat.cn API 地址 | `https://www.md2wechat.cn` |
| `image_key` | 否** | 图片生成 API Key | - |
| `image_base_url` | 否 | 图片 API 地址 | `https://api.openai.com/v1` |
| `convert_mode` | 否 | 转换模式 | `api` |
| `default_theme` | 否 | 默认主题 | `default` |
| `http_timeout` | 否 | 超时时间（秒） | `30` |

> **内测地址**：如需参与内测体验新功能，可将 `md2wechat_base_url` 设置为 `https://md2wechat.app`

* API 模式需要
** AI 生成图片时需要

#### 图片配置 (image)

| 配置项 | 必填 | 说明 | 默认值 |
|--------|------|------|--------|
| `compress` | 否 | 自动压缩 | `true` |
| `max_width` | 否 | 最大宽度 | `1920` |
| `max_size_mb` | 否 | 最大大小 | `5` |

---

## 环境变量

### 完整列表

| 环境变量 | 对应配置项 | 说明 |
|----------|-----------|------|
| `WECHAT_APPID` | `wechat.appid` | 微信 AppID |
| `WECHAT_SECRET` | `wechat.secret` | 微信 Secret |
| `MD2WECHAT_API_KEY` | `api.md2wechat_key` | md2wechat API Key |
| `MD2WECHAT_BASE_URL` | `api.md2wechat_base_url` | md2wechat API 地址 |
| `IMAGE_API_KEY` | `api.image_key` | 图片生成 API Key |
| `IMAGE_API_BASE` | `api.image_base_url` | 图片 API 地址 |
| `CONVERT_MODE` | `api.convert_mode` | 转换模式 |
| `DEFAULT_THEME` | `api.default_theme` | 默认主题 |
| `HTTP_TIMEOUT` | `api.http_timeout` | 超时时间 |
| `COMPRESS_IMAGES` | `image.compress` | 是否压缩 |
| `MAX_IMAGE_WIDTH` | `image.max_width` | 最大宽度 |
| `MAX_IMAGE_SIZE` | `image.max_size_mb` | 最大大小 |

### 设置方式

#### Linux/macOS

```bash
# 临时设置（当前终端有效）
export WECHAT_APPID="your_appid"
export WECHAT_SECRET="your_secret"

# 永久设置（添加到 ~/.bashrc 或 ~/.zshrc）
echo 'export WECHAT_APPID="your_appid"' >> ~/.bashrc
echo 'export WECHAT_SECRET="your_secret"' >> ~/.bashrc
source ~/.bashrc
```

#### Windows

```powershell
# 临时设置
$env:WECHAT_APPID="your_appid"
$env:WECHAT_SECRET="your_secret"

# 永久设置（系统环境变量）
# 控制面板 > 系统 > 高级系统设置 > 环境变量
```

---

## 配置优先级

当同一个配置项在多处定义时，优先级如下：

```
命令行参数 > 环境变量 > 配置文件 > 默认值
```

### 示例

假设有以下配置：

1. **配置文件** (`md2wechat.yaml`)：
   ```yaml
   api:
     default_theme: "default"
   ```

2. **环境变量**：
   ```bash
   export DEFAULT_THEME="autumn-warm"
   ```

3. **命令行参数**：
   ```bash
   md2wechat convert article.md --theme ocean-calm
   ```

结果：使用 `ocean-calm` 主题（命令行参数优先级最高）

---

## 配置管理命令

### 查看当前配置

```bash
md2wechat config show
```

输出示例：

```json
{
  "success": true,
  "config": {
    "wechat_appid": "wx123***def",
    "default_convert_mode": "api",
    "default_theme": "default"
  }
}
```

### 查看完整配置（包含密钥）

```bash
md2wechat config show --show-secret
```

### 验证配置

```bash
md2wechat config validate
```

### 初始化配置文件

```bash
# 在当前目录创建
md2wechat config init

# 指定路径
md2wechat config init ~/.config/md2wechat/config.yaml
```

---

## 安全建议

1. **不要提交配置文件到版本控制**

   ```bash
   # 添加到 .gitignore
   echo "md2wechat.yaml" >> .gitignore
   echo ".md2wechat.yaml" >> .gitignore
   ```

2. **使用环境变量存储敏感信息**

   ```bash
   export WECHAT_SECRET="your_secret"
   ```

3. **限制配置文件权限**

   ```bash
   chmod 600 md2wechat.yaml
   ```

---

## 下一步

配置完成后，请阅读 [使用教程](USAGE.md) 了解如何使用。
