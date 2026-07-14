# 架构说明

md2wechat-skill 的核心目标不是“把 Markdown 变好看”，而是把文章稳定转换成 **可发布、可验证、可自动化消费** 的微信公众号内容。

## 当前主线

当前代码按这条主线组织：

`cmd -> inspect + converter/publish orchestrators -> asset pipeline -> draft/wechat adapters`

这条主线的含义是：

- `cmd/md2wechat` 只负责参数解析、命令入口、输出 envelope 和错误出口。
- `internal/inspect` 负责把 metadata 来源、readiness 和 checks 收口成一份可解释真相。
- `cmd/md2wechat/preview.go` 复用 inspect 诊断并调用 converter；只有成功结果会被原样写入 HTML 文件。
- `internal/publish` 负责应用层编排，承接文章转换、图片处理、草稿保存和图片帖子创建。
- `internal/publish/AssetPipeline` 负责解析后的资产上传、生成、下载和 HTML 回填。
- `internal/draft` 和 `internal/wechat` 负责平台适配，不再承担命令级业务编排。

## 平台适配层

仓库把 skill 封装分成两条路径，避免把不同平台的安装模型混在一起：

- `skills/md2wechat/`：面向 Claude Code / Codex / OpenCode 的 coding-agent skill。
- `platforms/openclaw/md2wechat/`：面向 OpenClaw / ClawHub 的专用 skill 包。

两条路径共享同一个 CLI 内核，但平台适配不同：

- coding-agent skill 可以保留面向终端工作流的自然语言说明。
- OpenClaw skill 需要结构化声明依赖、安装和凭证要求。
- OpenClaw 安装主线是 skill 壳与已安装 CLI 配合使用；不应在 skill 包内保留运行时下载或 wrapper 自举逻辑。

## 模块职责

### `cmd/md2wechat`

- Cobra 命令定义
- 参数校验
- 统一 JSON envelope
- CLI 错误码与退出路径

### `internal/publish`

- `Service`: 图文发布主编排
- `ImagePostService`: 图片帖子主编排
- `AssetPipeline`: 发布期图片处理与回填
- `model.go`: canonical article / asset / artifact model

### `internal/inspect`

- resolved metadata 计算
- metadata source 标注
- readiness 计算（旧布尔字段 + targets/blockers 结构化投影）
- publish risk / semantic checks

### `internal/converter`

- Markdown 转 HTML
- frontmatter / metadata 提取
- Markdown 图片引用解析
- AI prompt 构建与 API/AI 转换模式

### `internal/image`

- 图片生成
- 图片压缩
- 上传/下载能力的运行时注入

### `internal/draft`

- 标准图文草稿 adapter
- 图片帖子 `newspic` adapter
- 摘要生成与微信草稿结构映射

### `internal/wechat`

- 微信 SDK 封装
- 素材上传与重试
- 新建草稿 / newspic draft
- 远程下载边界与 SSRF 防护

## 两条发布流

### 确认层

在真正执行发布前，当前架构多了一条独立确认层：

1. `inspect` 读取 Markdown 与配置
2. 解析 metadata、正文结构、资产和上下文
3. 输出 resolved metadata、source、readiness（含 targets/blockers）、checks
4. `preview` 复用 inspect 结果并调用 converter；成功时原样写入 HTML，AI handoff、失败或空结果在本次调用中不新建或覆盖文件，既有显式输出仍可能作为陈旧内容保留

这里最重要的约束是：

- `inspect` 是真相源
- `preview` 不是第二套规则引擎
- inspect 诊断属于 JSON 响应，不得包进 preview HTML
- 任何 metadata/source/checks 的业务判断都不应该只存在于 `preview`

### `convert`

1. 读取 Markdown
2. 提取 metadata，并应用 `--title` / `--author` / `--digest` 覆盖
3. 解析图片引用
4. 执行 API 或 AI 转换
5. 必要时通过 `AssetPipeline` 上传并回填 HTML
6. 可选保存 draft JSON
7. 可选上传封面并创建微信草稿

metadata 解析顺序：

- 标题：命令行 `--title` -> frontmatter `title` -> 正文首个 Markdown 标题 -> `未命名文章`
- 作者：命令行 `--author` -> frontmatter `author`
- 摘要：命令行 `--digest` -> frontmatter `digest` / `summary` / `description`

### `create_image_post`

1. 读取标题、内容和图片输入
2. 必要时从 Markdown 提取本地图片
3. 通过 `ImagePostService` 统一归一化为 `AssetRef`
4. 通过 `AssetPipeline` 上传图片
5. 通过 `draft` / `wechat` adapter 创建 `newspic` 草稿

## 稳定契约

当前工程里有 5 个关键契约：

1. `internal/publish/model.go`
   - 统一 article / asset / artifact model
2. `internal/action/action.go`
   - 统一 `completed / action_required / failed`
3. CLI JSON envelope
   - 统一 `success / code / message / schema_version / status / retryable / data / error`
4. `internal/inspect`
   - 统一 resolved metadata / source / readiness（含 targets/blockers 投影）/ checks
5. `AssetPipeline`
   - 统一图片上传、生成、下载、回填

## 当前设计原则

- 命令层不做业务编排
- `inspect` 是确认层真相源，`preview` 只做视图层
- 平台适配层不做内容预处理
- 图片链只保留一套发布期实现
- 机器输出优先稳定，再考虑人类可读文案
- 文档只描述已经落地、已验证的路径

## 下一阶段建议

当前本地架构已经基本收口。下一阶段优先级不再是继续拆内部模块，而是：

1. 做真实微信/API 凭证的 smoke/staging 验证
2. 继续收口外部系统契约，而不是继续刷局部覆盖率
3. 只有在出现第二个发布后端时，才考虑把 `AssetPipeline` 再抽成独立 domain
