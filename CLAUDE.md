# Claude 工作指南

> 本文件为 AI 助手（Claude）在本项目中的工作规范和流程指南。

---

## 项目概述

**md2wechat** 是一个 Markdown 转微信公众号格式的 CLI 工具，同时提供 Claude Code 和 OpenClaw 的 Skill 支持。

**核心原则：用户阅读体验和使用体验第一**

## Agent 发现优先

当任务依赖以下资源时，不要先猜：

- 图片 provider
- 可用主题
- Prompt 模板
- 当前实例支持的 CLI 能力

先执行：

```bash
md2wechat capabilities --json
md2wechat providers list --json
md2wechat themes list --json
md2wechat prompts list --json
md2wechat layout list --json
```

需要具体资源时，再执行：

```bash
md2wechat providers show <name> --json
md2wechat themes show <name> --json
md2wechat prompts show <name> --kind <kind> --json
md2wechat prompts render <name> --kind <kind> --var KEY=VALUE --json
md2wechat layout show <name> --json
md2wechat layout render <name> --var KEY=VALUE
md2wechat layout validate --file <file.md> --json
```

当前 prompt catalog 主要承载：

- `humanizer`（强度：`gentle` / `medium` / `aggressive` / `authentic`）
- `refine`
- `image`

扩展封面图、信息图、润色策略时，优先新增 YAML prompt 资产，不要直接把大段提示词继续写进 Go 代码。

---

## 版本发布流程

发布新版本时，**必须**按以下顺序执行检查：

### 0. 真实环境 E2E 渲染验证（必须第一个执行）

**在任何文档检查、版本号更新、git 操作之前**，先用真实 API 验证高级排版语法渲染正常。

```bash
# 确认本地 API 服务已启动（http://localhost:3000）
curl -s http://localhost:3000/ > /dev/null || echo "⚠️ 本地服务未启动，请先启动再发布"

# 构建最新 CLI
make build

# 运行高级排版渲染验证
./md2wechat convert /tmp/layout-e2e-test.md --mode api --output /tmp/layout-smoke.html

# 检查所有核心模块是否正常渲染（无 ::: 原始语法残留）
python3 -c "
modules = ['hero','toc','verdict','audience-fit','myth-fact','metrics','compare','steps',
           'timeline','quote','callout','definition','author-card','subscribe',
           'faq','checklist','cta','notice','summary']
html = open('/tmp/layout-smoke.html').read()
failed = [m for m in modules if ':::' + m in html]
ok = [m for m in modules if ':::' + m not in html]
print(f'✅ 渲染成功: {len(ok)}/{len(modules)}')
if failed: print(f'❌ 未渲染模块: {failed}')
else: print('全部通过，可以继续发布流程')
"

# 用 layout validate 检查测试文件语法正确性
./md2wechat layout validate --file /tmp/layout-e2e-test.md --json
```

**E2E 测试文件**：保存在 `examples/layout-e2e-test.md`，覆盖 6 大类模块的正确语法示例。每次发布前用此文件验证。

**通过标准**（两项都必须满足才能继续）：
- `convert` 输出 HTML 中，所有已知模块无 `:::name` 原始语法残留
- `layout validate` 返回 `LAYOUT_VALIDATED`（errors: 0，warnings: 0）

**如果 E2E 验证失败**：
- 检查是否是测试文件的语法问题（字段名、格式）
- 检查是否是 API 服务版本不兼容
- 检查是否是 YAML 模块定义的 `example` 示例有误
- **不允许跳过这一步继续发布**

### 1. 代码与文档一致性检查

```bash
# 检查所有文档中的命令示例是否与实际 CLI 一致
grep -r "md2wechat" docs/ README.md skills/md2wechat/SKILL.md

# 检查目录结构描述是否与实际一致
tree -L 3 --dirsfirst

# 检查配置示例是否与代码中的默认值一致
grep -r "config" internal/config/
```

**检查项：**
- [ ] CLI 命令示例能正常执行
- [ ] 配置文件示例格式正确
- [ ] 环境变量名称与代码一致
- [ ] 项目结构图与实际目录一致
- [ ] 安装路径描述正确

### 2. 版本号一致性检查

**所有需要更新版本号的位置：**

| 文件 | 位置 | 格式 |
|------|------|------|
| `VERSION` | 文件内容 | `x.y.z` |
| `package.json` | `"version"` 字段 | `x.y.z` |
| `.claude-plugin/marketplace.json` | plugin version / owner / author | `x.y.z` / 当前维护者身份 |
| `platforms/openclaw/md2wechat/SKILL.md` | `metadata.openclaw.install` | 唯一安装项必须是 `@geekjourneyx/md2wechat` 的 Node/npm 安装资源 |
| `CHANGELOG.md` | 新版本章节标题 | `## [x.y.z] - YYYY-MM-DD` |
| `CHANGELOG.md` | 版本历史表格 | 新增一行 |

```bash
# 快速检查版本号一致性（等价于 quality-gates step 0）
echo "=== 版本号检查 ==="
echo "VERSION:      $(cat VERSION)"
echo "package.json: $(node -e "process.stdout.write(require('./package.json').version)")"
echo "marketplace:  $(sed -n 's/.*"version": "\([0-9][^"]*\)".*/\1/p' .claude-plugin/marketplace.json | head -1)"
echo "CHANGELOG:    $(grep -m1 '^## \[' CHANGELOG.md | sed 's/^## \[\([^]]*\)\].*/\1/')"
```

> **说明**：OpenClaw SKILL.md 使用官方 `kind: "node"` 安装资源，由 npm 包自身版本与 release 资产保持一致；metadata 不再携带 Homebrew 或 Go 安装项。

**强制规则：**
- 发版前先运行 `bash scripts/quality-gates.sh`（step 0 会在 < 1s 内检查所有版本字段，发现漂移立即 fail）
- 发版前必须显式审校 `package.json`、`.claude-plugin/marketplace.json`
- 发版前必须显式审校 `skills/md2wechat/SKILL.md` 与 `platforms/openclaw/md2wechat/SKILL.md`
- 发版前必须显式审校 `scripts/install.sh`、`scripts/install-openclaw.ps1`、`platforms/openclaw/md2wechat/SKILL.md`
- 如果这些文件中的版本、下载 URL、命令示例、维护者信息没有同步，本次发布不能算完成

### 3. 文档规范检查

**Markdown 规范：**
- [ ] 标题层级正确（# → ## → ### 递进）
- [ ] 代码块指定语言（```bash, ```json, ```yaml）
- [ ] 表格对齐，表头完整
- [ ] 链接可访问，相对路径正确
- [ ] 无孤立的 HTML 标签

**内容规范：**
- [ ] 中英文之间有空格
- [ ] 专有名词大小写正确（GitHub, Claude Code, OpenClaw）
- [ ] 命令示例可直接复制执行
- [ ] 错误提示有对应的解决方案

**用户体验：**
- [ ] 新功能有使用示例
- [ ] 复杂操作有分步说明
- [ ] FAQ 覆盖常见问题
- [ ] 故障排查指南完整

### 4. CHANGELOG.md 更新

**必须包含：**
- 版本号和日期
- Added（新增功能）
- Changed（变更）
- Fixed（修复）
- Removed（移除）
- Technical Details（技术细节）
- Migration Guide（迁移指南，如有破坏性变更）

**模板：**
```markdown
## [x.y.z] - YYYY-MM-DD

### Added
- **功能名称**: 功能描述
  - 子功能点 1
  - 子功能点 2

### Changed
- **模块名称**: 变更描述

### Fixed
- 修复了 xxx 问题

### Removed
- 移除了 xxx

### Technical Details
- **New Files**: 新增文件列表
- **Modified Files**: 修改文件列表

### Migration Guide
迁移说明（如无破坏性变更则写 "No migration required"）
```

### 5. 等待用户确认

**在执行以下操作前，必须等待用户明确确认：**

- `git add` - 暂存更改
- `git commit` - 提交更改
- `git tag` - 创建标签
- `git push` - 推送到远程
- `gh release create` - 创建 GitHub Release
- `clawhub publish` - 发布到 ClawHub

**确认提示模板：**
```
准备发布 v{VERSION}

变更摘要：
- 新增: {N} 个功能
- 修改: {N} 个文件
- 删除: {N} 个文件

待执行操作：
1. git add -A
2. git commit -m "feat: ..."
3. git tag v{VERSION}
4. git push origin main --tags
5. gh release create v{VERSION}
6. clawhub publish (发布到 ClawHub)

是否继续？请确认。
```

### 6. 发布到 ClawHub（可选）

GitHub Release 创建完成后，尝试发布到 ClawHub 技能市场：

```bash
# 检查登录状态
clawhub whoami

# 如未登录，尝试登录
clawhub login

# 发布技能
clawhub publish ./platforms/openclaw/md2wechat \
  --slug md2wechat \
  --name "md2wechat" \
  --version {VERSION} \
  --changelog "版本更新说明" \
  --tags latest
```

**如果 ClawHub 发布失败（登录失败、网络问题等），可跳过此步骤。**

提示用户：
```
ClawHub 发布跳过。如需手动发布，请稍后执行：
  clawhub login
  clawhub publish ./platforms/openclaw/md2wechat --slug md2wechat --version {VERSION} --tags latest
```

**ClawHub 发布注意事项：**
- GitHub 账号需注册满 1 周才能发布
- `--slug` 是技能的唯一标识，首次发布后不可更改
- 发布后可在 https://clawhub.ai/ 查看

---

## 日常开发规范

### 代码修改后

1. 更新相关文档
2. 检查 SKILL.md 中的命令示例
3. 运行功能测试
4. 如果修改涉及 CLI command / subcommand / flag / JSON 输出 / provider / theme / prompt：
   - 必须同步审校 `README.md`
   - 必须同步审校 `docs/DISCOVERY.md`
   - 必须同步审校 `docs/FAQ.md`
   - 必须同步审校两套 `SKILL.md`
     - `skills/md2wechat/SKILL.md`
     - `platforms/openclaw/md2wechat/SKILL.md`
5. 这类任务不以“代码改完”为完成标准，必须完成高价值入口文档校准，防止代码和文档漂移
6. 这类任务如果会触发 CI 或 release，不要只跑 `go test`。必须先跑和云端一致的本地 gate：`make quality-gates`

### 新增命令 / 新增功能后的测试闭环

新增命令和新增功能，不能只要求“补测试”，而要按第一性原理补**有效测试**。

先问 3 个问题：

1. 这个功能最可能在哪种失败方式下破坏用户信任？
2. 这个功能最可能在哪种失败方式下误导 Agent 的下一步动作？
3. 这个功能和现有 `inspect` / `preview` / `convert` / `upload` / `draft` 哪些边界必须保持一致？

只有回答清楚这 3 个问题后，测试才算设计完成。

**强制规则：**

1. 不要为了 coverage 数字强行加测试。
2. 测试必须覆盖核心功能、关键契约、阻断条件或真实用户路径。
3. 新增命令至少要覆盖：
   - 合法输入
   - 非法输入
   - 机器输出契约（如果支持 `--json`）
   - 与相邻命令/流程的一致性边界
4. 如果新增的是确认层能力，必须补“确认层和执行层一致性测试”。
5. 如果新增的是多条件行为，优先写表驱动矩阵测试，不要只写一个 happy path。
6. 如果新增能力依赖外部系统，默认回归测试应尽量本地可重复；真实 smoke 只保留少量最小闭环。

**测试优先级顺序：**

1. CLI 契约测试
   - 参数校验
   - exit code
   - JSON envelope
   - stdout/stderr 边界
2. 确认层与执行层一致性测试
   - `inspect` / `preview` 不能比真实执行层更宽松
3. 阻断性 readiness / validation 矩阵
   - 标题/作者/摘要限制
   - 缺图
   - 缺封面
   - 缺凭证
4. 发布链路核心测试
   - 资产处理
   - draft 映射
   - 微信错误翻译
5. 最小真实 smoke
   - 至少保一条真实外部闭环

**完成标准：**

新增功能不以“代码能跑”为完成，也不以“coverage 提高了”为完成。
必须同时满足：

- 关键契约有测试保护
- 关键失败路径有测试保护
- 文档已同步
- 如有 release 或 CI gate，已通过和云端一致的本地 gate：`make quality-gates`

### 新增图片 Prompt 后

新增 `internal/assets/builtin/prompts/image/*.yaml` 时，必须把它视为完整产品变更，而不是单纯加一个模板。

**最小字段要求：**
- `name`
- `kind: image`
- `description`
- `version`
- `archetype`
- `primary_use_case`
- `recommended_aspect_ratios`
- `default_aspect_ratio`
- `metadata.author`
- `metadata.provenance`
- `template`

**按需补充：**
- `compatible_use_cases`
- `tags`
- `examples`
- `metadata.inspired_by`

**结构约束：**
- `default_aspect_ratio` 必须包含在 `recommended_aspect_ratios` 中
- 如果 prompt 可兼作封面/信息图，必须显式写 `compatible_use_cases`
- 不要把长 prompt 直接写进 Go 代码，优先落到 YAML 资产

**新增图片 prompt 后必须执行：**
1. `gofmt -l .`
2. `GOCACHE=/tmp/md2wechat-go-build go test ./internal/promptcatalog ./cmd/md2wechat`
3. `GOCACHE=/tmp/md2wechat-go-build go test ./...`
4. 必须校准高信号入口：
   - `README.md`
   - `docs/DISCOVERY.md`
   - `docs/FAQ.md`
   - `skills/md2wechat/SKILL.md`
   - `platforms/openclaw/md2wechat/SKILL.md`

**防漂移原则：**
- 如果漏了主用途、默认比例、来源字段，测试应直接拦住
- 如果新增了高频 preset，但两套 skill 没同步，这次任务不能算完成
- 用户、Agent、CLI 三个层面的说法必须一致

### 目录变更后

1. 更新 README.md 项目结构图
2. 更新安装脚本中的路径
3. 检查 .gitignore

### 新增功能后

1. 添加 CLI help 文档
2. 更新 SKILL.md 使用说明
3. 添加 FAQ 条目（如适用）
4. 更新 references/ 参考文档
5. 如果新增 provider/theme/prompt/discovery 命令，同步更新 `docs/DISCOVERY.md`
6. 如果影响配置、安装、默认行为或平台路径，同步审校：
   - `docs/CONFIG.md`
   - `docs/QUICKSTART.md`
   - `docs/USAGE.md`
   - `docs/OPENCLAW.md`（如影响 OpenClaw）
7. 按上面的测试闭环规则补足高价值测试；如果没有真实需要保护的契约，就不要为了形式感硬加测试

---

## 文件索引

### 核心配置
- `skills/md2wechat/SKILL.md` - 技能定义文件
- `platforms/openclaw/md2wechat/SKILL.md` - OpenClaw 专用 skill
- `docs/DISCOVERY.md` - 发现命令与 Prompt Catalog 说明

### 文档
- `README.md` - 项目主文档
- `docs/QUICKSTART.md` - 快速入门
- `docs/HUMANIZE.md` - 人性化润色保姆级教程（4 种强度 + authentic 模式）
- `CHANGELOG.md` - 版本变更记录
- `docs/` - 详细文档目录

### 安装脚本
- `scripts/install.sh` - CLI 全局安装
- `scripts/install-openclaw.ps1` - Windows OpenClaw 安装

---

## 常用命令

```bash
# 构建
make build

# 测试
GOCACHE=/tmp/md2wechat-go-build go test ./...

# 检查代码风格
go vet ./...
gofmt -l .

# 查看当前版本
cat VERSION

# 创建 GitHub Release（推 tag 后 GitHub Actions 自动创建，无需 gh CLI）
git tag v{VERSION}
git push origin main --tags
# → GitHub Actions release.yml 自动触发：构建 → smoke 测试 → 创建 Release

# ClawHub 发布
clawhub login                           # 首次使用需登录
clawhub whoami                          # 检查登录状态
clawhub publish ./platforms/openclaw/md2wechat \    # 发布 OpenClaw 技能
  --slug md2wechat \
  --name "md2wechat" \
  --version {VERSION} \
  --tags latest
clawhub search "md2wechat"              # 验证发布
```

---

## 注意事项

1. **永远不要**在未经用户确认的情况下执行 git push
2. **永远不要**修改历史提交（rebase, amend 等）除非用户明确要求
3. **始终**保持文档与代码同步
4. **始终**检查版本号一致性
5. **优先**考虑用户阅读和使用体验
6. **禁止**在 git commit 中包含 `Co-Authored-By: Claude` 签名 - 提交信息必须简洁，只写实质性内容
7. `docs/superpowers/` 目录存本地、不提交远程（已在 `.gitignore` 中排除）
8. Brand Profile 文件：`~/.config/md2wechat/brand.md`（Markdown 格式，非 YAML），由 `brand init` 初始化，Agent 读取但 CLI 不解析
