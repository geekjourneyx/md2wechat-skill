# 文档索引

`docs/` 保存仓库的详细文档。根目录只保留核心入口文档：

- `README.md`
- `CHANGELOG.md`
- `CLAUDE.md`

## 平台适配层

仓库里有两条 skill 路径，文档分别按平台维护：

- `skills/md2wechat/`：面向 Claude Code / Codex / OpenCode 的 coding-agent skill
- [Obsidian / Claudian 指南](OBSIDIAN.md)：在 Obsidian 的 Claudian 插件里使用 `md2wechat`
- `platforms/openclaw/md2wechat/`：面向 OpenClaw / ClawHub 的专用 skill 包
- [OpenClaw 指南](OPENCLAW.md)：OpenClaw 安装、验证与配置说明

## 入门

- [安装指南](INSTALL.md)
- [新手快速开始](QUICKSTART.md)
- [配置保姆级指南](CONFIG-WALKTHROUGH.md)
- [配置指南](CONFIG.md)
- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)：包含高级版 API 固定出口配置
- [能力发现与 Prompt Catalog](DISCOVERY.md)
- [公众号标题建议](TITLE_SUGGEST.md)
- [Agent 图片计划模式](AGENT_IMAGE_GEN.md)：当前 Agent 运行时暴露 Image Gen 时的 `--plan --json` 工作流
- [示例配置](examples/config.yaml.example)
- [完整使用说明](USAGE.md)
- [真实烟雾测试记录](SMOKE.md)
- [常见问题](FAQ.md)
- [故障排查](TROUBLESHOOTING.md)

## 架构与规范

- [架构说明](ARCHITECTURE.md)
- [设计原则](DESIGN.md)
- [Agent 协作规范](../AGENTS.md)
- [Claude Skill 规则](SKILL-RULE.md)

## 能力专题

- [**高级排版模块教程（保姆级）**](LAYOUT.md) ← API 模式专属，53 个推荐语法名（68 个场景条目）
- [配置保姆级指南](CONFIG-WALKTHROUGH.md)
- [配置说明](CONFIG.md)
- [微信凭证与 IP 白名单指南](WECHAT-CREDENTIALS.md)：包含高级版 API 固定出口配置
- [能力发现与 Prompt Catalog](DISCOVERY.md)
- [公众号标题建议](TITLE_SUGGEST.md)
- [内置资产](CONFIG.md#内置资产)
- [真实烟雾测试记录](SMOKE.md)
- [图片服务配置](IMAGE_PROVISIONERS.md)
- [Agent 图片计划模式](AGENT_IMAGE_GEN.md)
- [OpenClaw 指南](OPENCLAW.md)
- [Obsidian / Claudian 指南](OBSIDIAN.md)
- [写作功能问答](WRITING_FAQ.md)
