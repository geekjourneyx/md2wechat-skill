# 真实烟雾测试记录

本文档记录最近一次带真实配置、真实外部服务的最小闭环验证结果。

最近更新：

- 日期：2026-03-26
- 环境：本地配置文件 `~/.config/md2wechat/config.yaml`
- 目标：验证安装、配置、确认层、排版、图片、微信上传、草稿创建是否真实可用

> 说明：本文档只记录验证结论和关键观察，不记录敏感凭证、草稿 ID、素材 ID。

---

## 测试前提

已具备：

- 可读取的 `~/.config/md2wechat/config.yaml`
- 有效的 `WECHAT_APPID` / `WECHAT_SECRET`
- 有效的 `MD2WECHAT_API_KEY`
- 可用的图片服务配置
- 微信接口白名单已放通当前执行机 IP

### 高级排版 API 一致性

高级排版 conformance 默认离线跳过，不属于 `go test ./...` 的外部网络前提。显式执行：

```bash
make e2e-layout
```

该目标与 API 模式 `convert` 共用同一端点解析：默认完整 URL 是 `https://www.md2wechat.cn/api/convert`，并对 `MD2WECHAT_BASE_URL` / 配置文件值统一补全 `/api/convert`。本地或 staging 验证可显式设置 `MD2WECHAT_BASE_URL` 覆盖目标。凭证从 `~/.config/md2wechat/config.yaml` 的 `api.md2wechat_key` 读取，`MD2WECHAT_API_KEY` 可覆盖配置文件。API key 只进入 `X-API-Key` 请求头，不会出现在命令行、JSONL 报告或测试日志中。

报告默认写入 `/tmp/md2wechat-layout-conformance.jsonl`，可通过 `LAYOUT_CONFORMANCE_OUTPUT` 改路径。脚本为 Go 测试设置六分钟总时限；请求串行执行，且仅网络/5xx 短暂失败会最多重试两次。它会运行 84 个 witness conformance，以及一个覆盖 `default`、`apple`、`cyber`、`bytedance`、`sports`、`chinese` 的紧凑边界/组合探针。84 个 witness 由 56 个 canonical、25 个结构不同的 non-default branch 和 3 个 compatibility witness 组成；任一 module marker、稳定正文、精确 variant 分支属性、语义 DOM 约束或原始 fence 不一致都会失败。

可选设置 `MD2WECHAT_API_BUILD_ID` 锁定预期部署版本；设置后每个响应都必须携带 recognized build identity header 且精确匹配。未设置时仍要求本次 84-witness run 内所有响应 identity 一致；若均无 header，只记录 target 与 UTC 观测时间，并明确标记为非 commit 证据。失败类别区分 authentication、API drift 与 network failure。

发布时保留 JSONL 报告和测试日志作为目标 API 证据；不要把本地 `layout validate` 或常青文档中的历史运行结果当作远端部署证明。

执行过的基础自检：

```bash
md2wechat config validate --json
md2wechat config show --format json
md2wechat skills read md2wechat --json
```

---

## 通过的真实链路

### 0. 确认层：inspect / preview

命令形态：

```bash
md2wechat inspect article.md --json
md2wechat preview article.md --json
```

结果：

- `inspect` 能返回最终 title / author / digest 来源、`data.readiness`、`data.checks`；`data.readiness.targets/blockers` 能表达目标是否 blocked 以及对应原因
- 真实验证到 `TITLE_BODY_MISMATCH`、`DIGEST_METADATA_ONLY`、`IMAGE_REPLACEMENT_REQUIRES_UPLOAD_OR_DRAFT`
- `preview` 在转换成功时写入字节一致的 converter HTML；AI handoff、API 失败和空结果在本次调用中均不新建或覆盖 HTML，既有显式输出不得冒充本次结果
- `convert` 只有在显式请求 `--upload` / `--draft` 时进入远程副作用，并在任何远程调用前拒绝已知无效的 draft intent、cover 或本地资产

结论：

- confirm-first 路径真实可用
- 确认层没有用 dashboard、错误页或 Markdown fallback 伪造最终视觉结果

### 1. 图片生成

命令：

```bash
md2wechat generate_image --json "minimal smoke test banner..."
```

结果：

- 外部图片服务返回生成结果
- 生成图下载成功
- 微信素材上传成功

结论：

- `generate_image` 真实闭环通过

### 2. 上传本地图片

命令：

```bash
md2wechat upload_image --json /tmp/md2wechat-real-smoke.png
```

结果：

- 微信素材上传成功

结论：

- `upload_image` 真实闭环通过

### 3. 创建图片帖子

命令：

```bash
md2wechat create_image_post --json \
  -t "Smoke Test" \
  --images /tmp/md2wechat-real-smoke.png
```

结果：

- 图片上传成功
- 微信图片消息 / newspic 草稿创建成功

结论：

- `create_image_post` 真实闭环通过

### 4. 从 Markdown 创建图片帖子

验证了两种输入：

- 本地图片
- 远程图片 URL

结果：

- 两种输入都能进入统一资产链
- 远程图片会先下载，再上传微信，再创建图片帖子

结论：

- `create_image_post --from-markdown` 本地图 / 远程图均通过

### 5. 创建普通草稿

命令：

```bash
md2wechat test-draft --json draft.html cover.png
```

结果：

- 封面上传成功
- 微信普通草稿创建成功

结论：

- `test-draft` 真实闭环通过

### 5.1 草稿错误码提示

命令形态：

```bash
md2wechat create_draft oversize-digest.json --json
```

结果：

- 真实触发微信 `errcode=45004`
- 返回 `description size out of limit`
- CLI 额外补充字段级 hint，明确先检查 `--digest` / frontmatter `digest` / `summary` / `description`

结论：

- `45004` 不再被误导成“先缩正文”
- draft 错误提示已能指导用户和 Agent 走正确排障路径

### 6. API 模式转换并创建草稿

命令形态：

```bash
md2wechat convert \
  --json \
  --mode api \
  --upload \
  --draft \
  --cover cover.png \
  --output output.html \
  article.md
```

结果：

- `md2wechat.cn` API 排版成功
- 文内本地图片上传成功并回填
- 封面上传成功
- 草稿创建成功
- HTML 文件成功写出

结论：

- `convert --mode api --upload --draft --output` 真实闭环通过

### 7. AI 模式动作输出

命令形态：

```bash
md2wechat convert \
  --json \
  --mode ai \
  --theme autumn-warm \
  --output result.html \
  article.md
```

结果：

- 返回 `status=action_required`
- 返回 `code=CONVERT_AI_REQUEST_READY`
- 实际写出的是 `result.prompt.txt`
- 不再把 prompt 错写成 `.html`

结论：

- AI 模式当前是“生成 AI request / prompt”的半闭环
- 输出契约已经修正，不会伪造 HTML 成功

---

## 测试中发现的重要现象

### 1. 微信对白名单敏感

如果当前执行机 IP 不在微信公众号接口白名单中，上传图片和建草稿都会失败。  
这不是代码问题，是微信接口前置条件。

### 2. 极小测试图可能被微信拒绝

1x1 PNG 测试图曾被微信返回：

```text
unsupported file type
```

换成正常尺寸 PNG 后上传成功。  
结论是：真实 smoke 不要使用异常小图作为上传样本。

### 2.1 `--json` 契约已收口

当前 `inspect --json`、`preview --json` 等命令已验证：

- stdout 只输出 JSON
- JSON 是单行紧凑对象并以最终换行结束；人工检查时用 `jq` 格式化
- 结构化输出不再混入配置 banner
- `inspect --json` 的 Agent 决策字段位于 `data.readiness.targets/blockers`
- `skills read md2wechat --json` 能读取当前二进制内置 SOP，避免 smoke Agent 依赖旧 README 或旧外部 skill

这让 Agent / 脚本可以直接解析 stdout，而不用先清洗杂音。

### 3. AI 模式不是“自动完成 HTML 排版”

当前 CLI 下的 AI 模式语义是：

- 准备 prompt / request
- 交给外部 AI 继续完成

所以它应被视为 `action_required`，不是 `completed`。

---

## 当前建议的真实验证顺序

如果你要在新环境复现这组 smoke，按这个顺序最稳：

1. `md2wechat config validate --json`
2. `md2wechat upload_image --json <normal-image.png>`
3. `md2wechat create_image_post --json ...`
4. `md2wechat test-draft --json ...`
5. `md2wechat convert --mode api --upload --draft --cover ... --output ...`
6. 最后再测 `md2wechat convert --mode ai --json`

---

## 相关文档

- [配置指南](CONFIG.md)
- [安装指南](INSTALL.md)
- [OpenClaw 指南](OPENCLAW.md)
