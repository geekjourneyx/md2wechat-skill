# 高级排版模块完全教程

> **前提**：高级排版模块是 **API 模式**专属功能。`convert` 命令默认即 API 模式，无需额外参数。  
> 如需 API 访问权限，请联系作者咨询。

本教程讲解微信公众号高级排版模块的核心用法。当前数量与完整规格以 `md2wechat capabilities --json`、`layout list --json` 和 `layout show` 为准；默认 discovery 返回 53 个推荐语法。

---

## 目录

- [一、什么是高级排版模块](#一什么是高级排版模块)
- [二、3 步快速上手](#二3-步快速上手)
- [三、9 大类模块详解](#三9-大类模块详解)
  - [opening 开场类](#opening-开场类)
  - [infographic 信息图类](#infographic-信息图类)
  - [judgment 判断类](#judgment-判断类)
  - [evidence 证据类](#evidence-证据类)
  - [conversion 行动类](#conversion-行动类)
  - [brand 品牌类](#brand-品牌类)
  - [sprint4 精选增强类](#sprint4-精选增强类)
  - [free-layout 自由布局类](#free-layout-自由布局类)
  - [interactive 交互类](#interactive-交互类)
- [四、一篇完整文章示例](#四一篇完整文章示例)
- [五、Agent 工作流](#五agent-工作流)
- [六、常见错误排查](#六常见错误排查)

---

## 一、什么是高级排版模块

### 问题先行

你有没有遇到过这些情况：

- 写了一篇好文章，但读者在手机上扫一眼就划走了
- 文章里有核心判断，但读者记不住你的观点
- 想做品牌感，但每篇文章风格都不一样
- 靠 AI 生成的 HTML，每次都长得不一样

高级排版模块就是为了解决这些问题。

### 核心概念

**高级排版模块** = 一组预定义的视觉卡片组件，用 `:::模块名` 的语法写在 Markdown 里，由 md2wechat API 渲染成精准的微信 HTML。

```
:::hero
eyebrow: 深度观察
title: 公众号排版的真问题
subtitle: 不是好不好看，是读者读不读得完
:::
```

转换后变成一个有结构、有视觉层级的开篇卡片。

### 4 件事原则

每个模块只服务这 4 件事之一：

| 目的 | 解决什么 | 代表模块 |
|------|---------|---------|
| **attention** | 让读者先知道值不值得读 | hero, cards, verdict |
| **readability** | 让手机窄屏阅读不累 | toc, steps, part |
| **memorability** | 让读者记住一个判断或品牌 | verdict, manifesto, author-card |
| **conversion** | 让读者愿意收藏/关注/咨询/转发/购买 | cta, faq, checklist |

**核心原则**：选最少的模块，每件事做好一个。一篇文章 hero 只有一个，verdict 只有一个，cta 只有一个。不要堆模块。

### 语法规则

每个模块的 `body_format` 决定正文写法。先用 `layout list --json` 粗看，再用 `layout show <name> --json` 查看完整规格。Schema 定义什么输入合法；canonical `Example` 是经过契约测试的可执行 witness，结构不同的分支才会额外提供 `Variants[].Example`。复用 witness，不要只凭模块名猜语法。

`body_format: fields`：

```
:::模块名
字段名: 字段值
:::
```

`body_format: rows`，可带标题：

```
:::模块名[卡片标题]
行1 | 列2 | 列3
:::
```

`body_format: json_object` 或 `body_format: json_array`：

```
:::模块名
{"key": "value"}
:::
```

完整的九种正文格式如下：

| `body_format` | 正文形态 | 典型模块 |
|---|---|---|
| `fields` | 每行 `key: value` | `hero`、`verdict` |
| `rows` | 每行一条、列用 `|` 分隔 | `toc`、`metrics` |
| `json_object` | 一个 JSON 对象 | `definition`、`tweet` |
| `json_array` | 一个 JSON 数组 | `stat-row`、`resource-list` |
| `markdown_images` | Markdown 图片列表，可夹带允许的文本 | `gallery-grid`、`svg-swipe-gallery` |
| `markdown_fields` | 重复字段组中允许 Markdown 图片 | `image-steps`、`figure-caption` |
| `split` | 两段正文由模块 schema 指定的分隔线隔开 | `split` |
| `lines` | 逐行条目，分隔符或前缀由 schema 约束 | `flow`、`callout` |
| `dialogue` | 成对前缀或具名说话人行 | `question`、`dialogue-pair` |

`compatible_body_formats` 是旧正文仍可通过校验的兼容入口，不改变主推格式。例如 `question` 的主格式是 `dialogue`，同时接受旧的 `json_array`。

### Catalog 计数与 lifecycle

<!-- layout-count-contract: recommended_scenarios=68 recommended_syntaxes=53 compatibility_modules=3 base_enhancements=4 render_syntaxes=60 -->

- 68 个主推高级排版场景条目是上游使用场景维度。
- 53 个主推 `:::` 语法名是 `layout list --json` 默认返回的 CLI 对象。
- 3 个 compatibility 模块是 `dialogue`、`gallery`、`longimage`，只用于旧稿迁移。
- 4 个基础增强能力与上述语法合计为 60 项渲染层语法能力。

68 与 53 不能互换：一个语法名可以承载多个场景或结构变体。场景映射只用于维护测试，不会通过 CLI discovery 输出。

本批新增的 10 个推荐语法名分属三个新/扩展类别：`figure-caption`、`gallery-grid`、`gallery-story`、`image-phone-shot`、`svg-reveal`、`svg-swipe-gallery`、`split`、`flow`、`matrix`、`dialogue-pair`。具体 opener、body、schema、canonical witness 和结构 variant 以 `layout show <name> --json` 为准。

2026-07-11 针对 `https://md2wechat.app` 的真实语义 conformance 复验覆盖 80 个 canonical、variant 与 compatibility witness，结果 80 pass / 0 fail。远端身份是 target + UTC timestamp 的 `non_commit_evidence`，不是可追溯到远端 commit 的 build identity。

---

## 二、3 步快速上手

### 第 1 步：发现有哪些模块

```bash
# 列出全部推荐模块
md2wechat layout list --json

# 按目的筛选（最常用）
md2wechat layout list --serves attention --json
md2wechat layout list --serves readability --json
md2wechat layout list --serves memorability --json
md2wechat layout list --serves conversion --json

# 按类别筛选
md2wechat layout list --category opening --json
md2wechat layout list --category sprint4 --json
```

输出例子：

```json
{
  "data": {
    "modules": [
      {
        "name": "hero",
        "category": "opening",
        "serves": ["attention", "readability"],
        "when_to_use": "文章开篇第一屏..."
      }
    ]
  }
}
```

### 第 2 步：查看某个模块的完整规格

```bash
# 查看 hero 模块的字段、用法、示例
md2wechat layout show hero --json
```

返回的关键字段：

- `WhenToUse`：什么时候用这个模块
- `WhenNotToUse`：什么时候不该用
- `Fields.Required`：必填字段列表
- `Fields.Optional`：可选字段列表
- `AntiPattern`：常见错误
- `Example`：可直接复制的语法示例

### 第 3 步：生成语法块

方式 A（直接复制 Example）：

```bash
md2wechat layout show hero --json | python3 -c "
import json,sys
d = json.load(sys.stdin)
print(d['data']['spec']['Example'])
"
```

方式 B（用 render 命令生成）：

```bash
md2wechat layout render hero \
  --var eyebrow=深度观察 \
  --var title="你写的文章，读者为什么不读" \
  --var subtitle="排版的本质是降低阅读决策成本" \
  --json
```

输出：

```
:::hero
eyebrow: 深度观察
title: 你写的文章，读者为什么不读
subtitle: 排版的本质是降低阅读决策成本
:::
```

把这段代码粘贴到 Markdown 文章对应位置，转换时就会渲染出来。

复杂正文不要压成大量 `--var`。把正文写入临时文件或通过 stdin 传入；opener 参数与方括号 caption 分开传递：

```bash
md2wechat layout render gallery-grid \
  --param columns=2 \
  --body-file /tmp/gallery-body.md \
  --json

printf '%s\n' '草稿输入 → 结构判断 → 模块选择' | \
  md2wechat layout render flow --caption "Agent 发布流程" --body-file - --json

printf '%s\n' '左侧内容' '---' '右侧内容' | \
  md2wechat layout render split --body-file - --json
```

### 验证语法

写完文章后，先验证再转换：

```bash
md2wechat layout validate --file article.md --json
```

- 返回 `code: LAYOUT_VALIDATED` 且没有 errors → 本地语法可进入转换
- 返回 `code: LAYOUT_VALIDATE_HAS_ERRORS` → 检查 errors，修复后再转

这里的“可以转换”只表示本地 catalog/schema 已接受语法。真正的 API HTML 渲染能力仍需远端 conformance 或一次真实转换证明；两者不是同一层验证。

---

## 三、9 大类模块详解

### opening 开场类

**目的**：在读者决定读还是划走的 3 秒内，先给出判断。

---

#### hero — 开篇主视觉

**什么时候用**：文章开头第一屏，替代普通 H1 标题。适合观点文、产品发布、重大宣布。

**字段**：

| 字段 | 必填 | 说明 |
|------|------|------|
| eyebrow | ✅ | 标签词，如"深度观察"、"行业警告" |
| title | ✅ | 主标题，必须是一句判断或承诺 |
| subtitle | 可选 | 副标题，对主标题补一刀 |
| cta_text | 可选 | 开篇钩子文案，如"↓ 3 分钟读完，给你一个判断" |

**示例**：

```
:::hero
eyebrow: 深度观察
title: 公众号排版的真问题
subtitle: 不是好不好看，是读者读不读得完
cta_text: ↓ 3 分钟，一个判断
:::
```

**不要这样用**：
- title 写成描述性句子（"本文介绍了...）而不是判断
- 在数据报告里用（改用 metrics）
- 一篇文章放两个 hero

---

#### toc — 阅读导航

**什么时候用**：文章超过 1500 字、有 3 个以上章节时，放在 hero 之后。

**格式**：`序号 | 章节名 | 一句话说明`

```
:::toc[阅读导航]
01 | 问题定义 | 为什么现有排版让读者离开
02 | 模块原理 | 推荐模块各自解决什么
03 | 实战示例 | 一篇观点文的完整排版过程
:::
```

---

#### cards — 开篇卡片矩阵

**什么时候用**：文章结构清晰、有 3-4 个并列主题时，替代普通文字目录。

**格式**：`卡片标题 | 副标题 | 说明 | 颜色`（颜色：`accent` 或 `default`）

```
:::cards[本文结构]
PART 01 | 问题 | 读者为什么不读你的文章 | accent
PART 02 | 原理 | 排版如何降低阅读决策成本 | default
PART 03 | 实战 | 推荐模块的选择逻辑 | default
PART 04 | 行动 | 今天就能上手的 3 步方法 | default
:::
```

---

#### part — 章节分隔

**什么时候用**：长文章的每个大章节开头，替代普通 `## 二级标题`。

**字段**：

```
:::part
eyebrow: PART 02
title: 模块选择逻辑
body: 不是每篇文章都需要用遍推荐模块。核心是：每件事做一个，做好一个。
:::
```

---

#### label-title — 标签标题

**什么时候用**：短文或单主题文章的开篇，比 hero 轻量。

```
:::label-title
label: 行业洞察
title: 公众号创作者正在经历什么
:::
```

---

### infographic 信息图类

**目的**：把关键数据和结构用视觉方式呈现，让读者在窄屏里快速扫描。

---

#### metrics — 核心数据行

**什么时候用**：有 2-4 个横向并列的关键指标，比如数据报告、产品参数。

**格式**：`指标名 | 数值 | 说明 | 颜色`（颜色：`accent` 或 `default`）

```
:::metrics[本次结果]
付费转化率 | 23% | 比上月提升 8 个百分点 | accent
平均阅读时长 | 4.2分钟 | 高于行业均值 1.8x | default
:::
```

---

#### compare — 对比行

**什么时候用**：有两种方案/时间点/方法需要横向对比时。

**格式**：`维度 | A方描述 | B方描述 | 颜色`（也可 `维度 | 旧描述 | 新描述`）

```
:::compare[效果对比]
文章打开率 | 旧版排版 3.2% | 新版模块化排版 8.7% | accent
读者完读率 | 41% | 79% | default
制作时间 | 每篇 2小时 | 每篇 35分钟 | default
:::
```

---

#### steps — 步骤卡

**什么时候用**：有 3-6 步的线性流程，替代普通有序列表。

**格式**：`序号 | 步骤名 | 步骤说明`

```
:::steps[落地步骤]
01 | 发现模块 | layout list 列出所有可用模块
02 | 查看规格 | layout show 确认字段和示例
03 | 写进文章 | 直接粘贴 :::module 语法
04 | 验证语法 | layout validate 检查错误
05 | 转换发布 | convert 输出微信 HTML
:::
```

---

#### timeline — 时间轴

**什么时候用**：有时间顺序的里程碑、发展历程、版本更新。

**格式**：`时间点 | 事件标题 | 事件说明`

```
:::timeline[发展历程]
2023.01 | 初版上线 | 支持基础 Markdown 转换
2023.09 | 主题系统 | 推出 48 个专业主题
2024.03 | Prompt Catalog | AI 图片生成集成
2025.01 | Layout Catalog | 高级排版模块目录发布
:::
```

---

#### infographic — 单条信息图

**什么时候用**：需要突出单个数字、比例、或核心结论时。

**字段**：

```
:::infographic
type: data
value: 79%
label: 完读率
note: 使用高级排版模块后的平均表现
:::
```

`type` 可选值：`data`（数字）、`quote`（引语）、`fact`（事实）

---

### judgment 判断类

**目的**：让读者记住作者的核心立场和判断，建立品牌认知。

---

#### verdict — 最终判断卡

**什么时候用**：观点文、复盘、方案结论，把你的核心判断单独拎出来。一篇文章只用一个。

**字段**：

```
:::verdict
eyebrow: 最终判断
title: 真正的护城河不是模块数量，而是品牌表达系统
body: 每个模块必须服务一个真实的阅读任务，否则只是换皮。
note: 适合观点文、复盘、方案结论
:::
```

---

#### audience-fit — 读者匹配卡

**什么时候用**：文章开头明确适合谁读、不适合谁读，帮读者快速判断。

**格式**：`类型 | 描述`（类型：`fit` 或 `not-fit`）

```
:::audience-fit
fit | 想用 AI 工具提升公众号制作效率的创作者
fit | 有固定更新节奏、需要稳定输出的自媒体人
not-fit | 刚开始写公众号、还没有固定内容方向的新手
:::
```

---

#### myth-fact — 认知纠偏

**什么时候用**：有需要打破的错误认知时，用"误区 vs 真相"的对比格式。

**格式**：`类型 | 内容`（类型：`myth` 或 `fact`）

```
:::myth-fact
myth | 排版好看就是配色丰富
fact | 排版的本质是让读者更快做出阅读决策
myth | 模块越多，文章越专业
fact | 只用最少的模块，每件事做好一个
:::
```

---

#### manifesto — 宣言卡

**什么时候用**：品牌宣言、价值观声明、重大立场时。比 verdict 更有力量感。

**字段**：

```
:::manifesto
eyebrow: 我们相信
title: 内容的价值不在于看起来多专业，而在于读者读完后想做点什么
:::
```

---

#### bridge — 转场

**什么时候用**：两个章节之间需要过渡，承上启下。

**字段**：

```
:::bridge
from: 我们看完了问题是什么
to: 现在来看怎么解决
:::
```

---

### evidence 证据类

**目的**：用数据、案例、图片支撑你的判断，让读者相信你说的是真的。

---

#### quote — 引用卡

**什么时候用**：引用他人观点、用户反馈、书中金句时，给出来源。

**格式**：`引用内容` + 可选 `来源 | 作者`

```
:::quote
一句话能让读者决定读不读，一段话能让读者决定收不收藏。
| 极客旅程 | 内容设计原则
:::
```

---

#### image-annotate — 图片标注

**什么时候用**：需要对图片的特定区域做说明时，如截图分析、海报拆解。

**字段**：

```
:::image-annotate
src: https://example.com/screenshot.png
title: 公众号后台截图解读
point: 01 | 12 | 15 | 标题区域 | 读者扫到的第一眼
point: 02 | 45 | 60 | 封面图 | 决定打开率的关键元素
note: 标注坐标为百分比（0-100），可添加多个 point
:::
```

`point` 格式：`序号 | X坐标(0-100) | Y坐标(0-100) | 标签 | 说明`（可多个）

---

#### image-compare — 图片对比

**什么时候用**：需要展示前后对比、A/B 测试结果时。

**字段**：

```
:::image-compare
before: https://example.com/before.png
after: https://example.com/after.png
label_before: 旧版排版
label_after: 新版模块化排版
:::
```

---

#### image-steps — 图片步骤

**什么时候用**：操作教程，每步配一张截图。

**格式**：`body_format: markdown_fields`。每组使用 `step` 和 `desc`，中间可放一张 Markdown 图片；图片可省略。可用 `{columns=2 caption_style=numbered}` 等参数。

```
:::image-steps{columns=2 caption_style=numbered}
step: 打开编辑器
![打开](https://example.com/open.jpg)
desc: 选择主题和文章结构。
note: 先确认文章目标
step: 复制到微信
![复制](https://example.com/copy.jpg)
desc: 检查预览后复制。
:::
```

---

#### image-text — 图文并排

**什么时候用**：需要图片配合文字说明时，图文左右排列。

**字段**：

```
:::image-text
src: https://example.com/photo.png
title: 模块化排版的效果
body: 用固定结构替代手工堆砌，每篇文章都有一致的品牌气质。
:::
```

---

### conversion 行动类

**目的**：文章读完之后，让读者做一件事（收藏、关注、咨询、转发、购买）。

---

#### cta — 行动召唤

**什么时候用**：文章结尾，引导读者采取行动。一篇文章只用一个。

**字段**：

```
:::cta
title: 如果你想把公众号做成稳定可复用的结构，可以从这套模块开始。
note: 联系作者咨询 API 服务
:::
```

---

#### faq — 常见问题

**什么时候用**：有 3-8 个读者经常问的问题，或者需要处理潜在疑虑时。

**格式**：`问题 | 回答`

```
:::faq[常见问题]
这些模块只能在某个主题里用吗？ | 不是，48 个专业主题都支持高级排版模块。
API 模式和 AI 模式有什么区别？ | API 模式直接转换输出 HTML，AI 模式生成提示词给外部 AI。
我的文章需要用几个模块？ | 按 4 件事原则选，hero 1 个，verdict 1 个，cta 1 个，不要堆。
:::
```

---

#### checklist — 清单

**什么时候用**：有操作性清单、检查事项时，比普通列表更有视觉重量。

**格式**：`描述 | 状态`（状态：`done`、`todo`、`na`）

```
:::checklist[发布前检查]
md2wechat layout validate 通过 | done
封面图已准备好（比例 3:4，≥ 300px） | todo
摘要已填写（不超过 128 字） | todo
:::
```

---

#### cases — 案例卡

**什么时候用**：有 2-4 个真实案例或客户背书时。

**格式**：`案例名 | 行业 | 结果描述`

```
:::cases[使用案例]
某科技公众号 | 科技媒体 | 使用模块化排版后，平均完读率从 41% 提升到 79%
某企业内刊 | 金融行业 | 标准化模板让制作时间从 2小时降至 35分钟
:::
```

---

#### summary — 文章总结

**什么时候用**：章节复盘或文章结尾。根据读者需要选择一句话、三点、决策或收藏清单。

**必填规则**：默认/`legacy` 需要 `highlight`；`three` 和 `save` 需要 `items`；`decision` 需要 `fit` 或 `recommendation`。

```
:::summary
variant: three
title: 发布前带走三点
items: 结构先于风格 | 模块服务阅读 | 主题定义气质
note: 适合章节末尾和观点文复盘
:::
```

---

#### notice — 重要通知

**什么时候用**：有重要通知、政策变更、限时活动时。

**字段**：

```
:::notice
title: 重要提醒
body: 高级排版模块需要 API Key 才能使用。如需开通，请联系作者。
:::
```

---

### brand 品牌类

**目的**：让读者记住"谁写的"，建立作者品牌和订阅关系。

---

#### author-card — 作者卡片

**什么时候用**：文章开头或结尾，展示作者信息和定位。

**字段**：

```
:::author-card
name: 极客旅程
bio: 研究内容创作工具和 AI 工作流，专注公众号效率提升。
avatar: https://example.com/avatar.jpg
:::
```

---

#### subscribe — 关注引导

**什么时候用**：文章结尾，引导读者关注公众号。通常配合 cta 一起用。

**字段**：

```
:::subscribe
title: 关注极客旅程
body: 每周一篇，分享 AI 工具和内容创作方法论。
:::
```

---

#### people — 人物卡

**什么时候用**：介绍特定人物、专家访谈嘉宾、团队成员时。

**格式**：`姓名 | 职位 | 简介`

```
:::people[本期嘉宾]
张明 | 内容策略总监 | 10年媒体经验，主导过多个千万级公众号的内容体系建设
李华 | AI产品经理 | 专注 AI 写作工具研发，服务超过 500 个创作团队
:::
```

---

#### series — 系列说明

**什么时候用**：系列文章的开头，说明本文属于哪个系列、本篇位置。

**字段**：

```
:::series
name: 公众号排版进阶系列
episode: 第 3 篇，共 5 篇
topic: 高级排版模块实战指南
:::
```

---

### sprint4 精选增强类

**背景**：部分 sprint4 模块使用 JSON 格式写内容（不是 `key: value` 格式），适合更复杂的数据展示场景。

> **注意**：JSON 模式的关键字段名必须完全正确，否则渲染为空。先用 `layout show <name> --json` 查看 `body_format` 和字段名。

---

#### callout — 提示框

**什么时候用**：需要突出提示、警告、成功确认或危险说明时，支持 4 种样式。

**格式**：`:::callout 类型`（类型：`info` 默认、`warning`、`success`、`danger`）

```
:::callout
这是默认 info 样式，适合一般说明。
:::

:::callout warning
⚠️ 注意：高级排版模块仅在 API 模式下渲染，AI 模式不支持。
:::

:::callout success
✅ 成功：layout validate 返回 0 errors，可以转换了。
:::

:::callout danger
❌ 错误：不要用 --mode ai 时期望 :::module 模块渲染。
:::
```

---

#### definition — 术语定义

**什么时候用**：文章中有需要解释的专业术语时，嵌入行内定义卡片。

**格式**：单行 JSON，key 是 `term`、`def`（注意不是 `definition`）

```
:::definition
{"term":"OKR","def":"目标与关键结果","termLabel":"术语"}
:::
```

---

#### quote-card — 金句卡

**什么时候用**：需要单独突出一句话时，比普通引用更有视觉冲击力。

**格式**：单行 JSON，key 是 `text`（注意不是 `quote` 或 `content`）

```
:::quote-card
{"text":"结构先于风格，骨架决定气质","source":"内容设计原则"}
:::
```

---

#### tweet — 推文引用

**什么时候用**：引用社交媒体内容或呈现用户反馈时，用推文卡片格式。

**格式**：单行 JSON，key 是 `text`（不是 `content`）、`name`（不是 `author`）、`timestamp`（不是 `date`）

```
:::tweet
{"name":"内容创作者","handle":"@creator","text":"这套排版模块真的让我的制作效率提升了不止一倍。","timestamp":"2026-01-01","likes":"1.2K"}
:::
```

---

#### stat-row — 内联数据行

**什么时候用**：在正文段落中横向插入 2-4 个小指标时（比 metrics 更轻量）。

**格式**：JSON 数组，每项包含 `label`、`value`，可选 `unit`、`note`

```
:::stat-row
[{"label":"完读率","value":"79%"},{"label":"制作时间","value":"35","unit":"分钟"},{"label":"主题可选","value":"40","unit":"个"}]
:::
```

---

#### question — 问答

**什么时候用**：文章中有问答对，比 faq 更简洁。

**主格式**：按顺序成对书写 `Q:` / `A:`，每个问题必须紧跟一个回答。

```
:::question
Q: 为什么要用高级排版模块？
A: 因为普通 Markdown 在微信里没有视觉层级。

Q: 需要懂设计吗？
A: 不需要，照着字段填写就行。
:::
```

兼容旧稿的 JSON 数组格式；每一项都必须同时包含 `q` 和 `a`：

```
:::question
[{"q":"问题？","a":"答案。"}]
:::
```

---

#### resource-list — 资源列表

**什么时候用**：推荐工具、书单、链接集合时。

**格式**：JSON 数组，key 是 `name`、`url`、`desc`（不是 `description`）、`icon`

```
:::resource-list
[{"icon":"🛠","name":"md2wechat CLI","url":"https://github.com/geekjourneyx/md2wechat-skill","desc":"Markdown 转微信的命令行工具"},{"icon":"📖","name":"Layout 教程","url":"https://github.com/geekjourneyx/md2wechat-skill/blob/main/docs/LAYOUT.md","desc":"高级排版模块核心教程"}]
:::
```

---

#### comparison-table — 对比表格

**什么时候用**：两列对比（优点/缺点、方案A/方案B），比 compare 更结构化。

**格式**：JSON 对象，`left` 和 `right` 各含 `title` 和 `items`（字符串数组）

```
:::comparison-table
{"left":{"title":"AI 模式","items":["灵活度高","支持多种风格","不需要 API Key"]},"right":{"title":"API 模式","items":["稳定一致","支持高级排版模块","支持 48 个专业主题"]}}
:::
```

---

#### changelog — 版本日志

**什么时候用**：产品更新日志、版本说明、迭代记录。

**格式**：JSON 对象，`version` 必填，其余为字符串数组

```
:::changelog
{"version":"v2.3.1","date":"2026-05-28","added":["精选主题 wechat-native","高级排版主题展示图","API 主题集合展开"],"changed":["主题发现与文档口径校准"],"fixed":["api.yaml 中的主题无法被 CLI 发现"]}
:::
```

---

### free-layout 自由布局类

这组模块表达无法可靠压缩成 `key: value` 或表格行的正文结构：

| 模块 | 主格式 | 用途 |
|---|---|---|
| `split` | `split` | 用 schema 指定分隔线组织左右/前后两段内容 |
| `flow` | `lines` | 用逐行节点表达流程 |
| `matrix` | `rows` | 用稳定列数表达二维矩阵 |
| `dialogue-pair` | `dialogue` | 用成对说话人内容表达对话 |

先运行 `layout show <name> --json` 读取 separator、列数、允许前缀、canonical example 等约束。复杂正文使用 `--body-file`。

### interactive 交互类

| 模块 | 主格式 | 用途 |
|---|---|---|
| `svg-reveal` | `fields` | 单图揭示式交互 |
| `svg-swipe-gallery` | `markdown_images` | 多图滑动浏览 |

这两个语法只在 API 模式由远端 renderer 转成 HTML。`layout validate` 只证明本地 schema 接受输入，不证明远端已部署对应 renderer。

### compatibility 旧稿迁移

默认列表不会返回 `dialogue`、`gallery`、`longimage`。只在迁移已有文章时发现和检查它们：

```bash
md2wechat layout list --lifecycle compatibility --json
md2wechat layout show gallery --json
```

新稿分别使用 `dialogue-pair`、推荐图片模块以及其他当前 recommended 语法。

---

## 四、一篇完整文章示例

以下是一篇观点文的完整排版骨架，涵盖 opening → body → conversion → brand 的完整流程。

```markdown
---
title: 公众号排版的真问题
author: 极客旅程
digest: 不是好不好看，是读者有没有理由读下去
---

:::hero
eyebrow: 深度观察
title: 公众号排版的真问题
subtitle: 不是好不好看，是读者有没有理由读下去
kicker: 先给读者一个判断
:::

:::toc[阅读导航]
01 | 问题 | 读者为什么没有理由读你的文章
02 | 原理 | 排版要解决的 4 件事
03 | 实战 | 用最少的模块完成一篇文章
04 | 行动 | 今天就能上手
:::

---

## 01 问题：读者没有理由读你的文章

:::audience-fit
fit | 每周更新公众号、希望提升完读率的创作者
fit | 正在用 AI 写作、想要稳定排版输出的自媒体人
not-fit | 刚开始写公众号、还没有固定内容方向的新手
:::

在手机上，读者决定读还是划走只需要 **3 秒钟**。

:::callout warning
大多数文章失败不是因为内容差，而是因为前 3 秒没有给读者理由继续读。
:::

:::myth-fact
myth | 排版好看 = 配色丰富、字体花哨
fact | 排版的本质是降低读者的阅读决策成本
myth | 模块越多越专业
fact | 选最少的模块，每件事做好一个
:::

---

## 02 原理：排版要解决的 4 件事

:::metrics[排版的 4 个目标]
让读者知道值不值得读 | attention | hero / cards / verdict | accent
让手机阅读不累 | readability | toc / steps / part | default
让读者记住一个判断 | memorability | verdict / manifesto | default
让读者读完愿意行动 | conversion | cta / faq / checklist | default
:::

每个模块只服务其中一件事。一篇文章不需要用遍推荐模块，只需要每件事做对一个。

:::steps[选模块的方法]
01 | 判断文章类型 | 观点文 / 数据报告 / 教程 / 产品发布
02 | 按需选 1-2 个 | 每个目标最多选一个模块
03 | 用 render 生成 | md2wechat layout render <name> --var ...
04 | validate 校验 | md2wechat layout validate --file article.md --json
:::

---

## 03 实战：选最少的模块

:::compare[模块化前 vs 后]
制作时间 | 每篇约 2 小时，手工堆样式 | 每篇约 35 分钟，用模块填内容 | accent
完读率 | 行业平均 41% | 使用模块后平均 79% | default
品牌识别度 | 每篇风格不一样 | 固定骨架，风格稳定 | default
:::

:::verdict
eyebrow: 核心结论
title: 公众号排版的护城河不是审美，而是结构一致性
body: 让读者每次打开你的文章都知道"哦，这是 XX 的风格"，这才是品牌。
:::

---

## 04 行动：今天就能上手

:::checklist[上手清单]
安装 md2wechat CLI | done
配置 API Key | todo
运行 layout list 发现模块 | todo
写一篇文章，用 hero + verdict + cta | todo
:::

:::summary
variant: three
title: 本文要点
items: 3 秒内给出阅读理由 | 每件事只选一个模块 | 先 validate 再 convert
:::

:::cta
title: 想把公众号做成稳定可复用的结构？从这 3 个模块开始：hero + verdict + cta。
note: 联系作者咨询 API 服务
:::

:::author-card
name: 极客旅程
bio: 研究 AI 工作流和内容创作工具，专注公众号效率提升。
:::

:::subscribe
title: 关注极客旅程
body: 每周一篇，分享 AI 工具和内容创作方法论。
:::
```

---

## 五、Agent 工作流

如果你在用 Claude Code 或 OpenClaw 等 AI 助手，可以这样让 Agent 帮你排版：

### 让 Agent 发现并选择模块

```
请帮我分析这篇文章，选择合适的高级排版模块，并生成完整的排版 Markdown。

文章类型：观点文
文章文件：article.md
```

Agent 会自动执行：

```bash
# 1. 发现可用模块
md2wechat layout list --json

# 2. 按目标筛选
md2wechat layout list --serves attention --json

# 3. 查看具体模块规格
md2wechat layout show hero --json

# 4. 生成语法块
md2wechat layout render hero --var eyebrow=... --var title=... --json

# 5. 验证语法
md2wechat layout validate --file article.md --json

# 6. 转换
md2wechat convert article.md --output article.html
```

### Agent 的选模块原则

Agent 在选模块时遵循以下顺序：

1. **判断内容类型**：观点文 / 数据报告 / 教程 / 产品发布 / 综合
2. **4 件事各选一**：
   - attention → hero（开场）或 cards（结构）
   - readability → toc（导航）或 part（分隔）或 steps（步骤）
   - memorability → verdict（结论）或 manifesto（宣言）
   - conversion → cta（行动）或 faq（疑问消除）
3. **不要堆模块**：每类最多 1 个，合计通常 3-5 个模块

---

## 六、常见错误排查

### 错误 1：`:::module` 没有渲染，原样输出

**原因**：使用了 `--mode ai`，AI 模式不渲染 `:::module` 语法。

**解决**：去掉 `--mode ai`，直接用默认 API 模式：

```bash
# ❌ 错误
md2wechat convert article.md --mode ai

# ✅ 正确
md2wechat convert article.md
```

### 错误 2：`layout validate` 报错 "missing required field"

**原因**：某个必填字段没有填写，或字段名写错。

**解决**：

```bash
# 查看该模块的必填字段
md2wechat layout show <name> --json

# 看 Fields.Required 列表，补全缺失字段
```

### 错误 3：sprint4 JSON 模块字段名写错（最常见）

正确的字段名（容易写错的）：

| 模块 | ❌ 错误写法 | ✅ 正确写法 |
|------|-----------|-----------|
| quote-card | `quote` 或 `content` | `text` |
| tweet | `content` | `text` |
| tweet | `author` | `name` |
| tweet | `date` | `timestamp` |
| definition | `definition` | `def` |
| question | `question` / `answer` | `q` / `a` |
| resource-list | `description` | `desc` |

**解决**：

```bash
# 查看正确的字段结构
md2wechat layout show tweet --json
# 看 Fields.Required 和 Fields.Optional 的 Name 字段
```

### 错误 4：validate 通过但转换后看不到模块效果

**原因**：本地没有连接到 API 服务，或 API Key 未配置。

**解决**：

```bash
# 检查配置
md2wechat config validate --json

# 检查 API Key 是否已设置
md2wechat config show --format json | grep api_key
```

### 错误 5：按旧的三种正文格式猜模块语法

不要只在 pipe、JSON object、JSON array 之间猜。当前 `body_format` 有九种，合法输入以 `layout show <name> --json` 返回的 schema 为准：

| `body_format` | 正文结构 | `layout show` 中要检查的 schema |
|---|---|---|
| `fields` | 每行 `key: value` | `Fields`；同时检查字段 required、enum 和 value type |
| `rows` | 每行一项，以 `Rows.Delimiter` 分列 | `Rows` 和 `Body`；检查最少列数、行 schema 和正文约束 |
| `json_object` | 单个 JSON 对象 | `Fields` 和 `Body` |
| `json_array` | JSON 对象数组 | `Fields` 和 `Body` |
| `markdown_images` | Markdown 图片列表及允许的说明文字 | `Body` 的图片数、条目数和分隔约束 |
| `markdown_fields` | 可包含 Markdown 图片的重复字段组 | `Fields` 与 `Body.Group` |
| `split` | 由指定 separator 分开的两段正文 | `Body.Separator` |
| `lines` | 逐行条目 | `Body` 的 separator、allowed prefixes 和 min items |
| `dialogue` | 成对前缀或具名说话人行 | `Body.RequiredPairs`、allowed prefixes 和 named-speaker 约束 |

模块开头的 caption、花括号参数和 token 参数不属于正文格式，检查 `Opener`。优先复用 canonical `Example`；只有结构不同的 renderer 分支才查看 `Variants[].Example`。

`question` 的 primary `dialogue` 使用成对 `Q:` / `A:` 行；其 legacy-compatible `json_array` 仅用于兼容旧稿，不应作为新稿推荐写法。

---

## 延伸阅读

- [发现命令完整说明](DISCOVERY.md)
- [配置文件说明](CONFIG.md)
- [完整使用教程](USAGE.md)
- [故障排查](TROUBLESHOOTING.md)
