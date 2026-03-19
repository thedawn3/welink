# Codex Starter Rules

这份文件是给 Codex 用的精简规则层。

默认要求：

- 先阅读 `docs/AI_PROJECT_STARTER/PROJECT_AI_PLAYBOOK.md`
- 再阅读 `docs/PROJECT_LOCAL_CONTEXT.md`
- 新对话先做非侵入式探索，再决定是否修改
- 修改 Git 仓库前，先运行 `$git-remote-sync-guard` 或等价的远端一致性检查
- 如果仓库是 `behind`、`diverged`、`no-upstream`、`no-remote` 或远端抓取失败，先告知用户，不开始编辑
- 不要自动 `pull`、`rebase`、`merge`、`push`
- 不要默认新建平行文档、平行规则、平行详情入口
- 文档更新优先并入现有章节，而不是不断补平行小节

## Codex 在 WeLink 的推荐输出节奏

1. 当前状态
2. 现有资产盘点
3. 兼容性审计
4. 下一步最小行动

## WeLink 额外提醒

- 新增后端接口时，默认检查路由、Swagger、`frontend/src/services/api.ts`、`frontend/src/types/index.ts`
- 新增联系人详情能力时，默认挂到 `ContactModal.tsx`
- 改 MCP 说明时，默认同步首页、README、`docs/README.md`、`mcp-server/README.md` 与平台 setup 文档
