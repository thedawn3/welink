# 变更说明总览

这个目录用于沉淀 WeLink 的内部交接型变更说明。

用途：

- 让后续开发者或 AI 不用只靠 `git diff` 理解最近一轮改动
- 记录用户可见变化、接口变化、文档联动和验证结果
- 沉淀“这轮为什么这样改、后续维护时要看哪里”

## 编写约定

- 面向内部交接，不写成宣传型 release note
- 每次较大改动单独建一篇
- 优先记录：
  - 分支 / 提交信息
  - 用户可见变化
  - 接口与类型变化
  - 文档同步变化
  - 验证结果
  - 当前已知状态
  - 后续维护提醒

## 命名规范

- 文件名统一使用小写英文加连字符：`feature-a-and-b.md`
- 名称优先反映“本轮最重要的用户可见变化 + 关键技术主题”
- 不用日期当前缀，避免同一轮多次修订时不断改名
- 不用 `misc`、`update`、`final` 这类信息量过低的名字

推荐模式：

- `major-feature-and-key-integration.md`
- `area-refactor-and-api-update.md`
- `platform-setup-and-mcp-onboarding.md`

结合 WeLink 当前场景，推荐类似：

- `relationship-analysis-v2-and-mcp-onboarding.md`
- `timeline-and-global-search-improvements.md`
- `windows-setup-and-data-doctoring.md`

## 建议模板

新建变更说明时，优先直接复制：

- [CHANGE_TEMPLATE.md](./CHANGE_TEMPLATE.md)

模板正文如下：

```md
# 标题

## 变更摘要

- 分支：
- 关键提交：
- 目标：

## 用户可见变化

- 无

## 接口与类型变化

- 无

## 文档同步变化

- 无

## 验证结果

- 无

## 当前已知状态

- 无

## 后续维护提醒

- 无
```

如果本轮没有接口变化或没有用户可见变化，可以保留标题但明确写“无”。

## 当前条目

| 文档 | 说明 |
|---|---|
| [relationship-analysis-v2-and-mcp-onboarding.md](./relationship-analysis-v2-and-mcp-onboarding.md) | 关系分析 v2、联系人聊天时间线、统一 runtime/MCP/ChatLab 整合 |
