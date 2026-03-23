# 真实烟雾测试记录

本文档记录最近一次带真实配置、真实外部服务的最小闭环验证结果。

最近更新：

- 日期：2026-03-19
- 环境：本地配置文件 `~/.config/md2wechat/config.yaml`
- 目标：验证安装、配置、排版、图片、微信上传、草稿创建是否真实可用

> 说明：本文档只记录验证结论和关键观察，不记录敏感凭证、草稿 ID、素材 ID。

---

## 测试前提

已具备：

- 可读取的 `~/.config/md2wechat/config.yaml`
- 有效的 `WECHAT_APPID` / `WECHAT_SECRET`
- 可用的图片服务配置
- 微信接口白名单已放通当前执行机 IP

执行过的基础自检：

```bash
md2wechat config validate --json
md2wechat config show --format json
```

---

## 通过的真实链路

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
5. `md2wechat convert --mode api --upload --draft --output ...`
6. 最后再测 `md2wechat convert --mode ai --json`

---

## 相关文档

- [配置指南](CONFIG.md)
- [安装指南](INSTALL.md)
- [OpenClaw 指南](OPENCLAW.md)
