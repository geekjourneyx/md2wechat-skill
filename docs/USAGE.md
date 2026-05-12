# 使用教程

本文档详细说明 md2wechat 的各种使用方式。

## 目录

- [Claude Code 集成](#claude-code-集成)
- [基础使用](#基础使用)
- [转换模式](#转换模式)
- [图片处理](#图片处理)
- [主题定制](#主题定制)
- [草稿管理](#草稿管理)
- [完整示例](#完整示例)

---

## Claude Code 集成

### 安装（最简单）

在 Claude Code 中运行：

```bash
/plugin marketplace add geekjourneyx/md2wechat-skill
/plugin install md2wechat@geekjourneyx-md2wechat-skill
```

### 使用方式

安装后，直接与 Claude 对话即可：

```
请用秋日暖光主题将 article.md 转换为微信公众号格式，并上传到草稿箱
```

```
帮我把这篇技术文章用深海静谧主题转换，预览效果给我看
```

Claude 会自动：
1. 调用 md2wechat 转换 Markdown
2. 应用你选择的主题
3. 上传图片到微信
4. 创建草稿或显示预览

---

## 基础使用

### 最简单的例子

```bash
# 预览转换结果（不上传图片）
md2wechat convert article.md --preview
```

### 常用命令组合

```bash
# 1. 预览模式 - 快速查看效果
md2wechat convert article.md --preview

# 2. 保存到文件
md2wechat convert article.md -o output.html

# 3. 上传图片并输出 HTML
md2wechat convert article.md --upload -o output.html

# 4. 完整流程 - 上传图片 + 创建草稿
md2wechat convert article.md --upload --draft
```

---

## 转换模式

### API 模式（推荐新手）

使用 md2wechat.cn API 进行转换，稳定可靠。

```bash
md2wechat convert article.md --mode api --api-key "your_key"
```

**特点**：
- 转换速度快
- 结果稳定一致
- 需要注册 API Key

**可用主题**：
- `default` - 默认主题
- `bytedance` - 字节跳动风格
- `apple` - Apple 极简风格
- `sports` - 运动活力风格
- `chinese` - 中国传统文化风格
- `cyber` - 赛博朋克风格

### AI 模式（适合定制）

使用 AI 生成 HTML，更加灵活。

```bash
md2wechat convert article.md --mode ai --theme autumn-warm
```

**特点**：
- 高度可定制
- 主题更精美
- 需要 AI API Key

**可用主题**：
- `autumn-warm` - 秋日暖光
- `spring-fresh` - 春日清新
- `ocean-calm` - 深海静谧
- `custom` - 自定义

### 模式对比

| 特性 | API 模式 | AI 模式 |
|------|---------|---------|
| 速度 | 快 | 较慢 |
| 稳定性 | 高 | 中 |
| 主题选择 | 基础 | 丰富 |
| 成本 | 需要 API Key | 需要 AI Key |
| 适用场景 | 日常使用 | 追求美观 |

---

## 图片处理

### 图片语法

在 Markdown 中使用标准图片语法：

```markdown
<!-- 本地图片：会上传到微信 -->
![图片描述](./images/photo.jpg)

<!-- 在线图片：会先下载再上传 -->
![图片描述](https://example.com/image.jpg)

<!-- AI 生成图片：会调用 API 生成 -->
![图片描述](__generate:A cute orange cat__)
```

### 自动上传

```bash
# 自动上传所有图片
md2wechat convert article.md --upload

# 上传并替换 HTML 中的图片链接
md2wechat convert article.md --upload -o output.html
```

### 手动上传单个图片

```bash
# 上传本地图片
md2wechat upload_image ./photo.jpg

# 下载并上传在线图片
md2wechat download_and_upload https://example.com/image.jpg
```

### AI 生成图片

```bash
# 生成图片并上传
md2wechat generate_image "A beautiful sunset over mountains"

# 用内置封面模板生成封面图
md2wechat generate_cover --article article.md

# 用内置信息图模板生成信息图
md2wechat generate_infographic --article article.md --preset infographic-timeline

# 通用入口也支持 preset 模式
md2wechat generate_image --preset cover-hero --article article.md

# 单次覆盖图片模型
md2wechat generate_image --preset cover-hero --article article.md --model doubao-seedream-5.0-lite
```

输出示例：

```json
{
  "success": true,
  "prompt": "A beautiful sunset over mountains",
  "media_id": "12345***6789",
  "wechat_url": "http://mmbiz.qpic.cn/..."
}
```

### 图片压缩

程序会自动压缩超过限制的图片：

- 宽度超过 1920px → 等比缩放到 1920px
- 大小超过 5MB → 压缩质量
- 格式转换 → PNG → JPEG（可选）

配置压缩参数：

```yaml
# md2wechat.yaml
image:
  compress: true
  max_width: 1920      # 最大宽度
  max_size_mb: 5       # 最大大小（MB）
```

---

## 主题定制

### 使用内置主题

```bash
# 秋日暖光
md2wechat convert article.md --mode ai --theme autumn-warm

# 春日清新
md2wechat convert article.md --mode ai --theme spring-fresh

# 深海静谧
md2wechat convert article.md --mode ai --theme ocean-calm
```

### 主题预览

| 主题 | 色调 | 风格 |
|------|------|------|
| autumn-warm | 橙色 | 温暖治愈 |
| spring-fresh | 绿色 | 生机盎然 |
| ocean-calm | 蓝色 | 理性专业 |

### 自定义提示词

```bash
md2wechat convert article.md --mode ai --custom-prompt "
请使用蓝色配色方案，创建专业的技术博客风格。
标题使用深蓝色 #1a365d，正文使用 #2d3748。
"
```

### 设置默认主题

在配置文件中设置：

```yaml
api:
  default_theme: "autumn-warm"  # 设置默认主题
```

---

## 草稿管理

### 创建微信草稿

```bash
# 直接创建草稿
md2wechat convert article.md --draft

# 先上传图片再创建草稿
md2wechat convert article.md --upload --draft
```

### 保存草稿 JSON

```bash
# 保存草稿到文件（不提交到微信）
md2wechat convert article.md --save-draft draft.json

# 查看草稿文件
cat draft.json
```

草稿 JSON 格式：

```json
{
  "articles": [
    {
      "title": "文章标题",
      "content": "<section>...</section>",
      "digest": "文章摘要..."
    }
  ]
}
```

### 从 HTML 创建草稿

```bash
md2wechat create_draft \
  --file article.html \
  --cover cover.jpg \
  --title "文章标题" \
  --desc "文章摘要"
```

其中：

- `--file`：要发布的 HTML 文件
- `--cover`：封面图路径，会先上传到微信素材库
- `--title`：草稿标题
- `--desc`：草稿摘要，不传时会根据 HTML 内容自动生成

---

## 完整示例

### 示例 1：新手入门

```bash
# 1. 首次使用，初始化配置
md2wechat config init
# 编辑 ~/.config/md2wechat/config.yaml，填入微信 AppID、Secret 和 API Key

# 2. 验证配置
md2wechat config validate

# 3. 预览转换
md2wechat convert my-article.md --preview

# 4. 创建草稿
md2wechat convert my-article.md --draft
```

### 示例 2：使用精美主题

```bash
# 1. 使用 AI 模式 + 秋日暖光主题
md2wechat convert my-article.md \
  --mode ai \
  --theme autumn-warm \
  --preview

# 2. 满意后，上传图片并创建草稿
md2wechat convert my-article.md \
  --mode ai \
  --theme autumn-warm \
  --upload \
  --draft
```

### 示例 3：批量处理

```bash
#!/bin/bash
# batch-convert.sh

for file in articles/*.md; do
  echo "Converting $file..."
  md2wechat convert "$file" \
    --mode ai \
    --theme autumn-warm \
    --upload \
    --draft
done
```

### 示例 4：CI/CD 集成

```bash
#!/bin/bash
# .github/workflows/publish.yml

# 设置环境变量
export WECHAT_APPID="${{ secrets.WECHAT_APPID }}"
export WECHAT_SECRET="${{ secrets.WECHAT_SECRET }}"

# 转换并创建草稿
md2wechat convert article.md \
  --upload \
  --draft \
  --save-draft /outputs/draft.json
```

---

## 高级技巧

### 组合使用模式

```bash
# 使用 API 模式转换，但用 AI 模式的主题提示词
md2wechat convert article.md \
  --mode api \
  --custom-prompt "参考 autumn-warm 主题的配色"
```

### 仅处理图片

```bash
# 提取所有图片链接
md2wechat convert article.md --preview | grep IMG

# 上传所有图片并保存 URL
md2wechat convert article.md --upload -o temp.html
```

### 调试模式

```bash
# 查看详细日志
md2wechat convert article.md --preview 2>&1 | tee debug.log
```

---

## 故障排除

### 问题：转换结果为空

**原因**：Markdown 内容为空或格式错误

**解决**：
```bash
# 检查文件内容
cat article.md

# 检查文件编码
file article.md
```

### 问题：图片未替换

**原因**：未使用 `--upload` 参数

**解决**：
```bash
md2wechat convert article.md --upload -o output.html
```

### 问题：草稿创建失败

**原因**：微信 API 权限不足或调用频率限制

**解决**：
```bash
# 检查配置
md2wechat config validate

# 先保存 JSON，手动上传
md2wechat convert article.md --save-draft draft.json
```

---

## 下一步

- 查看 [FAQ](FAQ.md) 了解常见问题
- 查看 [示例文件](../examples/) 了解更多用法
