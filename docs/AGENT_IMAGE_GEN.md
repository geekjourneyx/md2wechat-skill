# Agent 图片计划模式

`generate_image`、`generate_cover` 和 `generate_infographic` 支持 `--plan --json`。这个模式不会直接请求图片服务，而是把 md2wechat 已经整理好的图片意图、prompt 和元数据交给当前 Agent。

如果当前 Agent 运行时暴露了 Image Gen 工具，Agent 可以读取 `data.prompt`，调用宿主提供的图片生成能力，把结果保存为本地文件，再交回 md2wechat 上传或创建草稿。

一句话流程：

1. md2wechat 生成图片计划。
2. Agent 读取 `data.prompt`。
3. Agent 在宿主运行时可用时调用 Image Gen，并保存本地文件。
4. 使用 `upload_image` 或 `convert --draft --cover <file>` 进入微信流程。

## 适用场景

- 当前 Agent 运行时暴露了 Image Gen 工具，但你不想在 md2wechat 里配置图片 provider。
- 你希望先让 md2wechat 根据文章、preset、画幅和用途整理 prompt，再由 Agent 接管执行。
- 你需要封面图、信息图或配图，但图片服务凭证由宿主 Agent 环境管理。
- 你想让 Agent 人工确认 prompt、画幅和用途后再执行图片生成。

## 不适用场景

- 你希望 md2wechat 直接生成并上传图片：请配置 provider，并使用普通 `generate_image` / `generate_cover` / `generate_infographic`。
- 当前 Agent 运行时没有 Image Gen 工具：计划仍可生成，但不会出现图片文件。
- 你需要完全自动的 CLI 闭环：计划模式会停在 `action_required`，等待宿主 Agent 执行。
- 你想把图片 provider 切到名为 `agent` 的特殊后端：没有这个 provider。

## 快速开始：封面图

先让 md2wechat 读取文章并产出封面计划：

```bash
md2wechat generate_cover --article article.md --plan --json
```

典型 JSON envelope 会类似这样：

```json
{
  "success": true,
  "code": "IMAGE_PLAN_READY",
  "message": "Image plan ready; generate the image with a host Agent or configured provider.",
  "schema_version": "v1",
  "status": "action_required",
  "retryable": false,
  "data": {
    "prompt": "Create a WeChat article cover image...",
    "archetype": "cover",
    "primary_use_case": "cover",
    "aspect": "16:9",
    "default_aspect_ratio": "16:9",
    "side_effects": false,
    "requires_provider": false,
    "requires_image_api_key": false,
    "execution_owner": "host_agent"
  }
}
```

然后由 Agent 读取 `data.prompt`。如果当前 Agent 运行时暴露 Image Gen，Agent 调用宿主工具并把图片保存为本地文件，例如 `/tmp/cover.png`。

上传到微信永久素材库：

```bash
md2wechat upload_image /tmp/cover.png --json
```

或者直接作为草稿封面：

```bash
md2wechat convert article.md --draft --cover /tmp/cover.png
```

## 信息图示例

```bash
md2wechat generate_infographic \
  --article article.md \
  --preset infographic-comparison \
  --aspect 21:9 \
  --plan --json
```

Agent 读取返回的 `data.prompt`、`data.archetype`、`data.primary_use_case`、`data.aspect` 和 `data.default_aspect_ratio` 后，再决定是否调用宿主 Image Gen。`data.aspect` 是本次执行提示；`data.default_aspect_ratio` 是 preset 自身声明的默认比例，两者可能不同。

## 自定义配图示例

```bash
md2wechat generate_image \
  "一张适合公众号文章的扁平插画：团队围绕白板讨论产品发布计划" \
  --plan --json
```

也可以继续使用 preset 和文章上下文：

```bash
md2wechat generate_image \
  --preset cover-hero \
  --article article.md \
  --plan --json
```

## 怎么选 provider 路径还是计划路径

选择直接 provider 路径，如果：

- 你已经配置 `IMAGE_API_KEY` 或 `api.image_key`。
- 你希望 md2wechat 直接请求 OpenAI、MiniMax、Volcengine、ModelScope、OpenRouter、Gemini 等 provider。
- 你要让 CLI 完成生成、上传或草稿创建的自动流程。

选择 Agent 图片计划路径，如果：

- 当前 Agent 运行时暴露 Image Gen 工具。
- 你不想把图片 provider key 配进 md2wechat。
- 你希望 Agent 先审阅 prompt，再决定是否执行。
- 你只需要一个稳定的 JSON 计划给宿主 Agent 编排。

## 重要边界

- `--plan --json` 不会请求图片 provider。
- `--plan --json` 不会上传到微信。
- `--plan --json` 不要求或使用 `IMAGE_API_KEY` 进行图片 provider 调用。
- `--plan --json` 不判断宿主 Agent 是否有 Image Gen。
- 没有名为 `agent` 的图片 provider 环境变量路径。
- Codex、WorkBuddy 或其他 Agent 是否可用 Image Gen，取决于当前运行时实际暴露的工具。
- 在宿主工具完成前，本地不存在由这一步产出的图片文件。

## FAQ

### 怎么知道当前 Agent 有没有 Image Gen？

看当前 Agent 界面、工具列表或运行环境说明里是否有 Image Gen / image generation / 生成图片一类工具。不同 Agent、不同版本、不同会话权限可能不一样；md2wechat 只返回计划，不判断宿主能力。

### 为什么 md2wechat 不自动使用 Agent 的 Image Gen？

Agent 工具属于宿主运行时，不是 md2wechat 的 CLI provider。md2wechat 不能假设工具名称、参数格式、权限模型或输出路径都一致，所以计划模式只提供稳定的 JSON 交接点。

### 计划模式产出的图片在哪里？

没有图片。`IMAGE_PLAN_READY` 表示 prompt 和执行元数据已经准备好；只有宿主 Agent 调用 Image Gen 并保存文件后，才会出现例如 `/tmp/cover.png` 这样的本地图片。

### 有 Agent Image Gen 时还需要 `IMAGE_API_KEY` 吗？

计划路径不需要 `IMAGE_API_KEY`，因为 md2wechat 不请求 provider。直接 CLI 生成路径仍然需要 provider 配置和对应 key。
