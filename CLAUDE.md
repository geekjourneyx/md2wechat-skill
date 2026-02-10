# Claude 工作指南

> 本文件为 AI 助手（Claude）在本项目中的工作规范和流程指南。

---

## 项目概述

**md2wechat** 是一个 Markdown 转微信公众号格式的 CLI 工具，同时提供 Claude Code 和 OpenClaw 的 Skill 支持。

**核心原则：用户阅读体验和使用体验第一**

---

## 版本发布流程

发布新版本时，**必须**按以下顺序执行检查：

### 1. 代码与文档一致性检查

```bash
# 检查所有文档中的命令示例是否与实际 CLI 一致
grep -r "md2wechat" docs/ README.md QUICKSTART.md skills/md2wechat/SKILL.md

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
| `skills/md2wechat/scripts/run.sh` | 第 13 行 | `VERSION="x.y.z"` |
| `.claude-plugin/marketplace.json` | `plugins[0].version` | `"x.y.z"` |
| `CHANGELOG.md` | 新版本章节标题 | `## [x.y.z] - YYYY-MM-DD` |
| `CHANGELOG.md` | 版本历史表格 | 新增一行 |

```bash
# 快速检查版本号一致性
echo "=== 版本号检查 ==="
echo "run.sh: $(grep 'VERSION=' skills/md2wechat/scripts/run.sh | head -1)"
echo "marketplace.json: $(grep '"version"' .claude-plugin/marketplace.json)"
echo "CHANGELOG.md: $(grep '## \[' CHANGELOG.md | head -1)"
```

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
clawhub publish ./skills/md2wechat \
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
  clawhub publish ./skills/md2wechat --slug md2wechat --version {VERSION} --tags latest
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

### 目录变更后

1. 更新 README.md 项目结构图
2. 更新安装脚本中的路径
3. 检查 .gitignore

### 新增功能后

1. 添加 CLI help 文档
2. 更新 SKILL.md 使用说明
3. 添加 FAQ 条目（如适用）
4. 更新 references/ 参考文档

---

## 文件索引

### 核心配置
- `.claude-plugin/marketplace.json` - Claude Code Plugin 配置
- `skills/md2wechat/SKILL.md` - 技能定义文件
- `skills/md2wechat/scripts/run.sh` - 二进制下载器

### 文档
- `README.md` - 项目主文档
- `QUICKSTART.md` - 快速入门
- `CHANGELOG.md` - 版本变更记录
- `docs/` - 详细文档目录

### 安装脚本
- `scripts/install.sh` - CLI 全局安装
- `scripts/install-openclaw.sh` - OpenClaw 安装

---

## 常用命令

```bash
# 构建
make build

# 测试
go test ./...

# 检查代码风格
go vet ./...
gofmt -l .

# 查看当前版本
grep 'VERSION=' skills/md2wechat/scripts/run.sh

# 创建 GitHub Release
git tag v1.x.x
git push origin main --tags
gh release create v1.x.x --generate-notes

# ClawHub 发布
clawhub login                           # 首次使用需登录
clawhub whoami                          # 检查登录状态
clawhub publish ./skills/md2wechat \    # 发布技能
  --slug md2wechat \
  --name "md2wechat" \
  --version 1.x.x \
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
