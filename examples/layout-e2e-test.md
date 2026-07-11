---
title: 高级排版模块 E2E 渲染验证
author: md2wechat
digest: 覆盖九类推荐模块的真实 API 渲染测试，每次发布前必须通过
---

:::hero
eyebrow: 排版能力测试
title: 53 个推荐模块，一次看清
subtitle: 真实 API 渲染验证
:::

:::toc[阅读导航]
01 | 判断类
02 | 信息图类
03 | 证据类
04 | 品牌类
05 | 转化类
06 | 图像与互动类
07 | 自由布局类
:::

## 一、判断类（judgment）

:::verdict
eyebrow: 最终判断
title: 高级排版模块显著提升阅读完成率
body: 结构化内容让读者快速定位核心判断，减少认知负担。
:::

:::audience-fit
fit: 内容创作者、运营、自媒体人
not-fit: 只需纯文字输出的场景
:::

:::myth-fact[常见误区]
排版越复杂越好看 | 最少模块、最准服务读者才是核心 | 数量只是覆盖面，准确才决定阅读体验
信息图必须像海报 | 信息图首先是一句话判断 | 图片和编号都不是必要内容
:::

## 二、信息图类（infographic）

:::metrics[核心数据]
推荐语法 | 53 个 | 覆盖九类内容结构 | accent
内容分类 | 9 类 | opening/judgment/infographic/evidence/interactive/brand/conversion/sprint4/free-layout | default
服务目标 | 4 个 | 知道值不值得读、不累、记住、行动 | default
:::

:::compare[旧方式 | 新方式]
手工堆样式 | 每篇都要重搭结构，容易乱 | 模块化排版 | 节奏固定，替换内容就能稳定复用
纯文字堆砌 | 读者要自己找重点 | 结构化视觉层级 | 模块直接引导注意力
:::

:::steps[落地步骤]
01 | 选择模块 | 根据文章类型选最少必要模块
02 | 填写字段 | 按 YAML 规范填写必填项
03 | 渲染预览 | 调用 convert 确认效果
:::

:::timeline[演进路径]
2024 Q1 | 基础渲染 | Markdown 基础渲染上线
2024 Q3 | 主题系统 | 多主题支持发布
2025 Q1 | API 模式 | 开放 API 转换接口
2026 Q2 | 高级排版 | 53 个推荐语法
:::

## 三、证据类（evidence）

:::quote
quote: 好的排版不是装饰，是帮读者省时间。
source: geekjourney
:::

:::callout
排版的本质：让读者用最少精力获取最大价值。
:::

:::definition
{"term":"高级排版模块","def":"一套预定义的结构化内容块，每个模块服务四个阅读目标之一，通过 :::name 语法触发渲染。","termLabel":"术语"}
:::

## 四、品牌类（brand）

:::author-card
name: geekjourney
bio: 专注微信公众号内容工程化
:::

:::subscribe
title: 每周分享内容创作效率工具
subtitle: 覆盖 AI 写作、排版系统和公众号工作流
:::

## 五、转化类（conversion）

:::faq[常见问题]
支持哪些排版模块？ | 53 个推荐语法，另有 3 个 compatibility 模块和 4 项基础增强。
需要懂代码吗？ | 不需要，只需按模块格式填写内容即可。
:::

:::checklist
已安装 md2wechat CLI
已配置 API Key
已确认本地服务运行中
已选择合适的排版模块
:::

:::notice[适用说明]
适合 | 干货长文、教程拆解、白皮书、活动总结 | 适合需要结构感和复用性的内容
前提 | 先把信息分层 | 不要把所有信息都塞进一个模块
不适合 | 特别短的快讯 | 这类内容通常一两个基础模块就够了
:::

:::cta
title: 立即体验高级排版
note: 支持 43 个模块，覆盖六大内容类型
:::

:::summary
eyebrow: 一句话总结
highlight: 先把结构搭稳，再让主题接管气质
body: 同一篇内容切到不同主题时，重点和节奏仍然清楚。
:::

## 六、图像与互动类（evidence / interactive）

:::gallery-grid{columns=3 variant=card}
![编辑器](https://images.unsplash.com/photo-1496171367470-9ed9a91ea931?w=800&h=600&fit=crop) | 编辑状态
![工作台](https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&h=600&fit=crop) | 工作状态
![设计稿](https://images.unsplash.com/photo-1558655146-d09347e92766?w=800&h=600&fit=crop) | 结果状态
:::

:::gallery-story{variant=card}
![第一步](https://images.unsplash.com/photo-1496171367470-9ed9a91ea931?w=800&h=600&fit=crop) | 整理草稿 | 先建立稳定的正文结构。
![第二步](https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800&h=600&fit=crop) | 检查结果 | 再核对手机阅读效果。
:::

:::figure-caption{caption_style=numbered}
![排版工作台](https://images.unsplash.com/photo-1496171367470-9ed9a91ea931?w=800&h=600&fit=crop)
caption: 高级排版渲染工作台
source: 公开示例图片
:::

:::image-phone-shot{columns=2 image_shape=phone}
![首页](https://images.unsplash.com/photo-1496171367470-9ed9a91ea931?w=600&h=1000&fit=crop) | 首页截图
![详情](https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=600&h=1000&fit=crop) | 详情截图
:::

:::svg-reveal{accent=brand}
question: 点击查看高级排版推荐语法数量
answer: 53
:::

:::svg-swipe-gallery
![第一张](https://images.unsplash.com/photo-1496171367470-9ed9a91ea931?w=600&h=1000&fit=crop) | 第一张静态回退图
![第二张](https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=600&h=1000&fit=crop) | 第二张静态回退图
:::

## 七、自由布局类（free-layout）

:::split
## 主判断

模块应当先解决阅读任务。

---

## 落地方式

左侧给结论，右侧给证据或下一步。
:::

:::flow[Agent 发布流程]
草稿输入 → 结构判断 → 模块选择 → 视觉校准 → 发布检查
:::

:::matrix headers=能力,基础版,专业版,企业版
高级模块|有|有|有
主题定制|无|有|有
API 调用|无|有|有
私有部署|无|无|有
:::

:::dialogue-pair left=读者 right=作者
U: 高级模块和普通 Markdown 有什么区别？
E: 高级模块把结构、层级和视觉节奏一起带进公众号正文。

U: 我需要学多少个模块？
E: 先学开场、信息卡、证据、总结和 CTA。
:::
