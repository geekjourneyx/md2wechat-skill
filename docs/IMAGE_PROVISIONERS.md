# 图片生成服务配置

md2wechat 支持多种图片生成服务，可以在 Markdown 中使用 AI 生成图片。

## 快速开始

在 Markdown 中使用以下语法生成图片：

```markdown
![图片描述](__generate:一个秋天的森林，阳光透过树叶__)
```

## 配置方式

在配置文件 `~/.config/md2wechat/config.yaml` 中配置图片生成服务：

```yaml
api:
  # 图片服务提供者: openai, tuzi, modelscope, openrouter
  image_provider: "openrouter"

  # API 配置
  image_key: "your-api-key"
  image_base_url: "https://api.tu-zi.com/v1"

  # 生成参数
  image_model: "doubao-seedream-4-5-251128"
  image_size: "2048x2048"  # TuZi 要求最小 3686400 像素

image:
  compress: true
  max_width: 1920
```

## 支持的图片服务

### TuZi

TuZi 是国内可用的图片生成服务，支持多种模型。

#### 配置示例

```yaml
api:
  image_provider: "tuzi"
  image_key: "tuzi-sk-..."
  image_base_url: "https://api.tu-zi.com/v1"
  image_model: "doubao-seedream-4-5-251128"
  image_size: "1024x1024"
```

#### 支持的模型

| 模型 | 说明 |
|------|------|
| `doubao-seedream-4-5-251128` | 豆包 Seedream 4.5（推荐） |
| `gemini-3-pro-image-preview` | Gemini 3 Pro 图片预览版 |

#### 支持的尺寸

| 尺寸 | 比例 | 说明 |
|------|------|------|
| `2048x2048` | 1:1 | 正方形（默认，4.2M 像素）|
| `1920x1920` | 1:1 | 正方形（最小要求，3.7M 像素）|
| `2560x1440` | 16:9 | 横版（3.7M 像素）|
| `1440x2560` | 9:16 | 竖版（3.7M 像素）|
| `3072x2048` | 3:2 | 横版（6.3M 像素）|
| `2048x3072` | 2:3 | 竖版（6.3M 像素）|
| `3840x2160` | 16:9 | 超宽横版（8.3M 像素）|
| `2160x3840` | 9:16 | 超高竖版（8.3M 像素）|

> **注意**：TuZi 要求图片尺寸至少达到 **3,686,400 像素**

#### 获取 API Key

前往 [TuZi 控制台](https://api.tu-zi.com) 注册并获取 API Key。

---

### OpenAI

使用 OpenAI DALL-E 模型生成图片。

#### 配置示例

```yaml
api:
  image_provider: "openai"
  image_key: "sk-..."
  image_base_url: "https://api.openai.com/v1"
  image_model: "dall-e-3"
  image_size: "1024x1024"
```

#### 支持的模型

| 模型 | 说明 |
|------|------|
| `dall-e-3` | DALL-E 3（最新） |
| `dall-e-2` | DALL-E 2 |

#### 支持的尺寸

| 尺寸 | 模型 |
|------|------|
| `1024x1024` | dall-e-2, dall-e-3 |
| `1792x1024` | dall-e-3 |
| `1024x1792` | dall-e-3 |

---

### OpenRouter

OpenRouter 提供统一的 API 接口，支持多种图片生成模型（如 Gemini、Flux 等）。

#### 配置示例

```yaml
api:
  image_provider: "openrouter"
  image_key: "sk-or-v1-..."
  # image_base_url 可选，默认为 https://openrouter.ai/api/v1
  image_model: "google/gemini-3-pro-image-preview"
  image_size: "16:9"  # 支持比例格式或 WIDTHxHEIGHT
```

或使用环境变量：

```bash
export IMAGE_PROVIDER="openrouter"
export IMAGE_API_KEY="sk-or-v1-..."
export IMAGE_MODEL="google/gemini-3-pro-image-preview"
export IMAGE_SIZE="16:9"
```

#### 支持的模型

更多模型请访问：https://openrouter.ai/models?q=image

| 模型 | 说明 |
|------|------|
| [`google/gemini-3-pro-image-preview`](https://openrouter.ai/google/gemini-3-pro-image-preview) | Gemini 3 Pro（默认，推荐） |
| `google/gemini-2.5-flash-image-preview` | Gemini 2.5 Flash |
| `black-forest-labs/flux.2-pro` | Flux 2 Pro（高质量）|
| `black-forest-labs/flux.2-flex` | Flux 2 Flex |
| `sourceful/riverflow-v2-standard-preview` | Riverflow v2 标准版 |
| `sourceful/riverflow-v2-fast` | Riverflow v2 快速版 |
| `sourceful/riverflow-v2-pro` | Riverflow v2 专业版 |

#### 支持的尺寸

OpenRouter 支持两种尺寸配置方式，可在配置文件中设置 `image_size`，也可通过命令行 `--size` 参数覆盖：

```bash
# 使用配置文件中的默认尺寸
md2wechat generate_image "A cute cat"

# 通过命令行指定尺寸（覆盖配置）
md2wechat generate_image --size "16:9" "A landscape photo"
md2wechat generate_image --size "1920x1080" "A landscape photo"
```

**方式一：使用宽高比（推荐）**

| 比例 | 1K 尺寸 | 说明 |
|------|---------|------|
| `1:1` | 1024×1024 | 正方形（默认）|
| `16:9` | 1344×768 | 横版（适合封面）|
| `9:16` | 768×1344 | 竖版（适合手机）|
| `4:3` | 1184×864 | 标准横版 |
| `3:4` | 864×1184 | 标准竖版 |
| `3:2` | 1248×832 | 横版照片 |
| `2:3` | 832×1248 | 竖版照片 |
| `5:4` | 1152×896 | 横版 |
| `4:5` | 896×1152 | 竖版 |
| `21:9` | 1536×672 | 超宽横版 |

**方式二：使用 WIDTHxHEIGHT**

常见尺寸会自动映射到对应的宽高比和分辨率等级：

| 尺寸 | 映射到比例 | 分辨率 |
|------|------------|--------|
| `1024x1024` | 1:1 | 1K |
| `2048x2048` | 1:1 | 2K |
| `4096x4096` | 1:1 | 4K |
| `1344x768` | 16:9 | 1K |
| `1920x1080` | 16:9 | 2K |
| `2560x1440` | 16:9 | 2K |
| `3840x2160` | 16:9 | 4K |
| `768x1344` | 9:16 | 1K |
| `1080x1920` | 9:16 | 2K |
| `1440x2560` | 9:16 | 2K |
| `2160x3840` | 9:16 | 4K |
| `1184x864` | 4:3 | 1K |
| `1600x1200` | 4:3 | 2K |
| `864x1184` | 3:4 | 1K |
| `1248x832` | 3:2 | 1K |
| `1800x1200` | 3:2 | 2K |
| `3072x2048` | 3:2 | 4K |
| `832x1248` | 2:3 | 1K |
| `1200x1800` | 2:3 | 2K |
| `2048x3072` | 2:3 | 4K |

#### 分辨率等级

OpenRouter 支持三种分辨率等级（通过尺寸自动判断）：

| 等级 | 说明 |
|------|------|
| 1K | 标准分辨率 |
| 2K | 较高分辨率（默认）|
| 4K | 最高分辨率 |

> **完整文档**: 更多图片生成配置和使用方法，请参考 [OpenRouter 官方文档](https://openrouter.ai/docs/guides/overview/multimodal/image-generation)。

#### 获取 API Key

前往 [OpenRouter](https://openrouter.ai) 注册并获取 API Key。

---

### Google Gemini

直接调用 Google Gemini API，使用官方 Go SDK，无需通过第三方平台。

#### 配置示例

```yaml
api:
  image_provider: "gemini"  # 或 "google"
  image_key: "AIza..."  # Google API Key
  image_model: "gemini-3-pro-image-preview"
  image_size: "16:9"  # 支持比例格式
```

或使用环境变量：

```bash
export IMAGE_PROVIDER="gemini"
export IMAGE_API_KEY="AIza..."  # 或 GOOGLE_API_KEY
export IMAGE_MODEL="gemini-3-pro-image-preview"
export IMAGE_SIZE="16:9"
```

#### 支持的模型

| 模型 | 说明 |
|------|------|
| [`gemini-3-pro-image-preview`](https://ai.google.dev/gemini-api/docs/image-generation) | Gemini 3 Pro 图片预览版（默认，推荐）|
| `gemini-2.5-flash-preview-image` | Gemini 2.5 Flash 图片版 |
| `gemini-2.0-flash-exp-image-generation` | Gemini 2.0 Flash 实验版 |

#### 支持的尺寸

Gemini 支持以下宽高比，可通过配置文件或 `--size` 参数指定：

| 比例 | 说明 |
|------|------|
| `1:1` | 正方形（默认）|
| `2:3` | 竖版照片 |
| `3:2` | 横版照片 |
| `3:4` | 标准竖版 |
| `4:3` | 标准横版 |
| `4:5` | 竖版 |
| `5:4` | 横版 |
| `9:16` | 竖版（适合手机）|
| `16:9` | 横版（适合封面）|
| `21:9` | 超宽横版 |

也支持 `WIDTHxHEIGHT` 格式（如 `1024x1024`），会自动映射到对应的宽高比和分辨率等级（1K/2K/4K）。

> **完整尺寸列表**: 每个宽高比支持 1K、2K、4K 三种分辨率等级，具体尺寸请参考 [Gemini 图片生成官方文档](https://ai.google.dev/gemini-api/docs/image-generation?hl=zh-cn)。

#### 获取 API Key

前往 [Google AI Studio](https://aistudio.google.com/apikey) 创建 API Key。

#### Gemini vs OpenRouter

| 对比 | Google Gemini 直接调用 | OpenRouter |
|------|------------------------|------------|
| 延迟 | 直连 Google，通常更低 | 经过中转 |
| 计费 | 直接与 Google 结算 | 通过 OpenRouter 结算 |
| 模型 | 仅 Gemini 系列 | 多种模型可选 |
| 配置 | `image_provider: gemini` | `image_provider: openrouter` |

---

## 使用示例

### 在 Markdown 中生成图片

```markdown
# 我的文章

这是一篇文章，里面有一张 AI 生成的封面图：

![封面图](__generate:温暖的秋天森林，阳光透过树叶洒在地面上__)

还可以生成插图：

![插图](__generate:赛博朋克风格的城市夜景，霓虹灯闪烁__)
```

### 命令行使用

```bash
# 转换文章（会自动生成图片并上传到微信）
md2wechat convert article.md --draft

# 只预览（不上传）
md2wechat convert article.md --preview
```

---

## 常见问题

### Q: 提示 "API Key 无效" 怎么办？

**A:** 请检查以下几点：
1. 配置文件中的 `api.image_key` 是否正确填写
2. API Key 是否已过期或被撤销
3. 对于 TuZi，前往控制台确认账户状态正常
4. 对于 OpenAI，确认 API Key 有效且有余额

---

### Q: 提示 "账户余额不足" 怎么办？

**A:**
- **TuZi**: 前往 [TuZi 控制台](https://api.tu-zi.com) 充值
- **OpenAI**: 前往 [OpenAI 控制台](https://platform.openai.com) 充值
- **OpenRouter**: 前往 [OpenRouter 控制台](https://openrouter.ai) 充值

---

### Q: 提示 "请求过于频繁" 怎么办？

**A:** API 服务有速率限制，请：
1. 等待一段时间后再试
2. 考虑升级服务套餐
3. 减少同时生成的图片数量

---

### Q: 提示 "参数配置有误" 怎么办？

**A:** 请检查：
1. `image_provider` 是否为 `openai`、`tuzi`、`modelscope`、`openrouter` 或 `gemini`
2. `image_model` 是否在支持的模型列表中
3. `image_size` 是否在支持的尺寸列表中（OpenRouter 也支持比例格式如 `16:9`）

---

### Q: 生成的图片不符合预期怎么办？

**A:** 尝试优化提示词：
1. 描述更具体：`一只金色的猫坐在红色的沙发上` 比 `猫` 更好
2. 添加风格描述：`油画风格`、`照片级真实`、`卡通风格` 等
3. 指定颜色和光线：`温暖的阳光`、`冷色调`、`高对比度` 等

---

### Q: 图片生成失败但没显示具体错误？

**A:** 运行以下命令查看详细日志：

```bash
# 设置日志级别为 debug
MD2WECHAT_LOG_LEVEL=debug md2wechat convert article.md --preview
```

---

## 环境变量配置

除了配置文件，也可以使用环境变量：

```bash
export IMAGE_PROVIDER="tuzi"
export IMAGE_API_KEY="your-api-key"
export IMAGE_API_BASE="https://api.tu-zi.com/v1"
export IMAGE_MODEL="doubao-seedream-4-5-251128"
export IMAGE_SIZE="2048x2048"  # TuZi 要求最小 3686400 像素
```

环境变量优先级高于配置文件。

---

## 错误代码速查

| 错误代码 | 说明 | 解决方案 |
|----------|------|----------|
| `unauthorized` | API Key 无效 | 检查 API Key 配置 |
| `payment_required` | 余额不足 | 前往控制台充值 |
| `rate_limit` | 请求过于频繁 | 等待后重试 |
| `bad_request` | 参数错误 | 检查模型和尺寸配置 |
| `network_error` | 网络错误 | 检查网络连接和 API 地址 |
| `no_image` | 未生成图片 | 检查提示词是否符合内容政策 |
