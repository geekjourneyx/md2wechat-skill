# 能力发现与 Prompt Catalog

`md2wechat` 提供了一组面向 Agent 和自动化脚本的发现命令，用来在执行前先确认当前 CLI 支持什么能力、有哪些可用资源、当前配置是否就绪。

这组命令的定位不是替代 `--help`，而是提供**可机读、可枚举、可提前探测**的能力接口。

## 推荐顺序

对于 Agent、脚本或 CI，建议按下面的顺序使用：

1. `md2wechat capabilities --json`
2. `md2wechat providers list --json`
3. `md2wechat themes list --json`
4. `md2wechat prompts list --json`
5. 需要具体模板时再用 `show` / `render`

## 能力总览

```bash
md2wechat capabilities --json
```

返回内容包含：

- 已开放的高层命令能力
- `convert` 支持的模式和默认模式
- 当前可枚举的图片 provider
- 当前可枚举的 theme
- 当前可枚举的 prompt catalog

## 图片 Provider

```bash
md2wechat providers list --json
md2wechat providers show openai --json
md2wechat providers show openrouter --json
md2wechat providers show ark --json
```

当前 CLI 会枚举 provider 元数据，包括：

- `name`
- `aliases`
- `description`
- `required_config`
- `optional_config`
- `default_base_url`
- `default_model`
- `supports_size`
- `current`
- `configured`

当前内置支持的图片 provider：

- `openai`
- `tuzi`
- `modelscope` / `ms`
- `openrouter` / `or`
- `gemini` / `google`

## 主题发现

```bash
md2wechat themes list --json
md2wechat themes show default --json
md2wechat themes show autumn-warm --json
```

主题信息来自运行时 ThemeManager，遵循当前加载优先级：

1. `MD2WECHAT_THEMES_DIR`
2. `./themes`
3. `~/.config/md2wechat/themes`
4. 内置 theme 资产

这意味着纯二进制安装也能列出官方默认主题，用户和平台仍可通过目录覆盖内置主题。

## Prompt Catalog

Prompt catalog 是一组内置并可覆盖的 YAML 资产，当前主要用于：

- `humanizer`
- `write` 的润色流程
- 后续扩展的图片 archetype

### 列出 Prompt

```bash
md2wechat prompts list --json
md2wechat prompts list --kind humanizer --json
md2wechat prompts list --kind refine --json
md2wechat prompts list --kind image --json
md2wechat prompts list --kind image --archetype cover --json
md2wechat prompts list --kind image --archetype infographic --json
md2wechat prompts list --kind image --tag editorial --json
```

### 查看 Prompt 定义

```bash
md2wechat prompts show medium --kind humanizer --json
md2wechat prompts show default --kind refine --json
md2wechat prompts show cover-default --kind image --json
md2wechat prompts show cover-hero --kind image --archetype cover --tag hero --json
md2wechat prompts show infographic-victorian-engraving-banner --kind image --archetype infographic --tag victorian --json
```

对于图片 prompt，`archetype` 表示主要分组，不代表只能用于这一种场景。优先查看 `prompts show --json` 返回的：

- `primary_use_case`
- `compatible_use_cases`
- `recommended_aspect_ratios`
- `default_aspect_ratio`

这样 Agent 能判断某些信息图 preset 是否也适合作为封面使用。

### 渲染 Prompt 模板

```bash
md2wechat prompts render cover-default \
  --kind image \
  --var article_title='从 0 到 1 做好公众号封面' \
  --var article_summary='一份关于封面图策略的实战清单' \
  --json
```

### 用 preset 直接生成图片

```bash
md2wechat generate_image --preset cover-hero --article article.md
md2wechat generate_cover --article article.md
md2wechat generate_infographic --article article.md --preset infographic-comparison
md2wechat generate_infographic --article article.md --preset infographic-dark-ticket-cn --aspect 21:9
md2wechat generate_infographic --article article.md --preset infographic-handdrawn-sketchnote
md2wechat generate_infographic --article article.md --preset infographic-apple-keynote-premium
md2wechat generate_infographic --article article.md --preset infographic-victorian-engraving-banner --aspect 21:9
md2wechat generate_image --preset cover-hero --article article.md --model doubao-seedream-5.0-lite
```

高频图片命令和 prompt catalog 的关系是：

- `generate_image`: 通用入口，可直接传 raw prompt，也可用 `--preset`
- `generate_cover`: `cover` archetype 的薄包装命令
- `generate_infographic`: `infographic` archetype 的薄包装命令
- 三个图片命令都支持 `--model`，用于单次覆盖本次调用的图片模型

如果某个图片 preset 的 `compatible_use_cases` 包含 `cover`，那么它也可以被 `generate_cover` 使用；默认画幅优先跟随 prompt 自身声明的 `default_aspect_ratio`。

当前内置 prompt kind：

- `humanizer`
- `refine`
- `image`

当前内置图片 archetype 分组：

- `cover`
- `infographic`

当前内置图片模板示例：

- `cover-default`
- `cover-hero`
- `cover-minimal`
- `cover-metaphor`
- `cover-editorial`
- `cover-illustrated`
- `cover-data-visual`
- `infographic-default`
- `infographic-comparison`
- `infographic-timeline`
- `infographic-dashboard`
- `infographic-hierarchy`
- `infographic-bento`
- `infographic-process`
- `infographic-flat-vector-panorama`
- `infographic-dark-ticket-cn`
- `infographic-handdrawn-sketchnote`
- `infographic-apple-keynote-premium`
- `infographic-victorian-engraving-banner`

## Prompt 资产覆盖顺序

Prompt catalog 的加载优先级为：

1. `MD2WECHAT_PROMPTS_DIR`
2. `./prompts`
3. `~/.config/md2wechat/prompts`
4. 内置 prompt 资产

也就是说，官方默认 prompt 会随二进制一起提供；用户和平台如需自定义，可按上面的顺序覆盖。

## 当前已接入 Prompt Catalog 的能力

目前已经优先使用 prompt catalog 的能力有：

- `humanize`
- `write` 的润色流程

当前仍然主要依赖代码或其他资产的部分：

- `convert` 的 API/AI 调用逻辑
- 更复杂的图片生成流程编排

后续扩展更多封面、信息图、配图 archetype 时，应优先新增 `prompts/image/*.yaml`，而不是直接把大段提示词写进 Go 代码。

## 与配置的关系

发现命令不会替代配置文件，但会帮助 Agent 确认：

- 当前默认 provider 是什么
- 当前 provider 是否已配置
- 当前是否存在某个 theme / prompt

配置主路径仍然是：

- `~/.config/md2wechat/config.yaml`

如需切换 API 域名、图片 provider 或其它默认值，请先看 [CONFIG.md](CONFIG.md)。
