# 故障排查向导

> 遇到问题？按照下面的步骤一步步排查

---

## 问题分类

点击你的问题跳转到解决方案：

- [安装问题](#安装问题)
- [配置问题](#配置问题)
- [转换问题](#转换问题)
- [图片问题](#图片问题)
- [微信问题](#微信问题)

---

## 安装问题

### ❓ 下载后双击没反应

**Windows**：
1. 右键点击 `md2wechat.exe`
2. 选择「属性」
3. 点击「解除锁定」（如果有）
4. 再双击运行

**Mac**：
1. 打开「系统偏好设置」→「安全性与隐私」
2. 点击「仍要打开」

---

### ❓ 提示 "命令不存在" 或 "不是内部或外部命令"

**原因**：程序没有添加到系统 PATH

#### Windows 解决方法：

**推荐方式：重新走固定版本安装器**

```powershell
$env:MD2WECHAT_RELEASE_BASE_URL = "https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.10.1"
iex ((New-Object System.Net.WebClient).DownloadString("$env:MD2WECHAT_RELEASE_BASE_URL/install.ps1"))
```

如果 CLI 已经安装，只是当前终端找不到，再手动补 PATH：

1. 搜索「环境变量」→「编辑系统环境变量」
2. 点击「环境变量」
3. 在「用户变量」中找到 `Path`，点击「编辑」
4. 点击「新建」，输入程序所在目录
5. 点击「确定」保存

#### Mac/Linux 解决方法：

```bash
curl -fsSL https://github.com/geekjourneyx/md2wechat-skill/releases/download/v2.10.1/install.sh | bash
export PATH="$HOME/.local/bin:$PATH"
md2wechat version --json
```

---

### ❓ Windows 提示 "Windows 已保护你的电脑"

1. 点击「更多信息」
2. 点击「仍要运行」

---

## 配置问题

如果你需要从零配置、迁移多公众号，或排查 `WECHAT_ACCOUNT_*` / `API_KEY_*` 错误，先看 [配置保姆级指南](CONFIG-WALKTHROUGH.md)。

### ❓ 提示 "WECHAT_APPID is required"

**原因**：还没有配置微信凭证

**解决步骤**：

更完整的新手版说明见：

- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)

1. **获取微信凭证**
   - 打开 https://developers.weixin.qq.com/platform
   - 登录后选择公众号 → 开发接口管理
   - 复制 AppID
   - 点击「重置」获取 AppSecret

2. **创建配置文件**
   ```bash
   md2wechat config init
   ```

3. **编辑配置文件**
   - 用记事本打开 `~/.config/md2wechat/config.yaml`
   - 填入你的 AppID 和 Secret
   - 保存

4. **验证配置**
   ```bash
   md2wechat config validate
   ```

---

### ❓ 配置文件在哪？

默认配置文件会创建在：

```text
~/.config/md2wechat/config.yaml
```

**查找配置文件**：
- `~/.config/md2wechat/config.yaml`（推荐，全局配置）
- `~/.md2wechat.yaml`
- `./md2wechat.yaml`

你也可以运行下面的命令确认当前实际生效的是哪个配置文件：

```bash
md2wechat config show --format json
```

---

## 转换问题

### ❓ 转换结果是空的

**可能原因 1**：文件路径不对

```bash
# 错误示例（文件不存在）
md2wechat convert 文章.md

# 正确示例（使用正确的文件名）
md2wechat convert 我的文章.md

# 或者使用完整路径
md2wechat convert /Users/你的名字/Documents/文章.md
```

**可能原因 2**：文件编码不是 UTF-8

- 用记事本打开文件
- 点击「另存为」
- 编码选择「UTF-8」
- 保存

---

### ❓ 中文显示乱码

**解决方法**：确保文件是 UTF-8 编码

1. 用 VS Code 或记事本打开文件
2. 点击「另存为」
3. 编码选择「UTF-8」
4. 保存

---

### ❓ AI 模式报错

先分清当前 AI 模式的语义：

- `convert --mode ai` 当前返回的是 AI request / prompt
- 它不是最终 HTML 成品
- 它也不是靠 `api.image_key` 去完成正文排版

**解决方法 A**：使用 API 模式（更简单）
```bash
md2wechat convert 文章.md --mode api
```

**解决方法 B**：先确认你是不是把 AI 模式当成了最终排版

```bash
md2wechat inspect 文章.md
md2wechat preview 文章.md --mode ai
md2wechat convert 文章.md --mode ai --json
```

如果你真正想要的是“直接拿到可发 HTML”，优先改回：

```bash
md2wechat convert 文章.md --mode api
```

---

## 图片问题

### ❓ 图片没有显示

**原因**：需要加上 `--upload` 参数

```bash
# 错误（不会上传图片）
md2wechat convert 文章.md

# 正确（上传图片）
md2wechat convert 文章.md --upload
```

---

### ❓ 图片上传失败

**可能原因 1**：图片格式不支持

**支持的格式**：`.jpg`、`.png`、`.gif`、`.bmp`、`.webp`

**不支持**：`.heic`（iPhone 默认格式）、`.tiff`

**解决方法**：用手机相册打开图片，选择「导出」为 JPEG 格式

---

**可能原因 2**：图片太大

程序会自动压缩，但如果仍然失败：

1. 用图片编辑器缩小图片
2. 或在配置文件中调整：
   ```yaml
   image:
     max_width: 1280  # 缩小最大宽度
     max_size_mb: 2   # 缩小最大文件大小
   ```

---

### ❓ 在线图片下载失败

**原因**：网络问题或图片链接有问题

**解决方法**：
1. 先用浏览器打开图片链接，确认能访问
2. 把图片下载到本地，使用本地图片：
   ```bash
   # 原来：![图片](https://...)
   # 改为：![图片](./images/图片.jpg)
   ```

---

## 微信问题

### ❓ 提示 "access_token expired"

**原因**：微信凭证过期

**解决方法**：
```bash
# 1. 验证配置
md2wechat config validate

# 2. 等待几分钟后重试
md2wechat convert 文章.md --draft
```

---

### ❓ 草稿创建失败

**可能原因 1**：公众号未认证

- 认证后的公众号才能使用草稿 API

**可能原因 2**：内容包含敏感词

- 尝试简化内容后重试
- 或先保存为 JSON：
  ```bash
  md2wechat convert 文章.md --save-draft draft.json
  ```

**可能原因 3**：缺少封面或摘要字段超限

先检查：

```bash
md2wechat inspect 文章.md --draft --cover cover.jpg
```

如果看到 `errcode=45004` / `description size out of limit`，优先缩短：

- `--digest`
- frontmatter 里的 `digest`
- frontmatter 里的 `summary`
- frontmatter 里的 `description`

不要先默认成“正文太长”。

**可能原因 4**：API 调用次数超限

- 等待几分钟后重试
- 或联系微信提高限额

---

## 需要更多帮助？

### 收集诊断信息

遇到问题时，运行以下命令收集信息：

```bash
# 1. 检查版本
md2wechat version --json

# 2. 验证配置
md2wechat config validate

# 3. 查看配置（不显示密码）
md2wechat config show --format json
```

### 获取支持

1. 查看 [常见问题](FAQ.md)
2. 查看 [使用教程](USAGE.md)
3. 提交 Issue：https://github.com/geekjourneyx/md2wechat-skill/issues

提交问题时，请附上：
- 你的操作系统（Windows 10 / macOS 13 / Linux）
- 错误信息的截图
- 运行的完整命令
