# 按 AI 工具区分的使用方式

## Codex / Codex Desktop / Codex CLI

最推荐的组合：

1. 自动读取根 `AGENTS.md`
2. 进入 `docs/AI_PROJECT_STARTER/PROJECT_AI_PLAYBOOK.md`
3. 再读 `docs/AI_PROJECT_STARTER/CODEX_RULES.md`
4. 再读 `docs/PROJECT_LOCAL_CONTEXT.md`

如果是新会话，推荐先发：

1. `docs/AI_PROJECT_STARTER/ONE_LINE_PROMPT.txt`
2. `docs/AI_PROJECT_STARTER/TEST_PROMPT.txt`

## Claude Code

推荐方式与 Codex 类似：

1. 从仓库根目录开对话
2. 先让模型读 `AGENTS.md`
3. 若需要强提醒，再手动发 `ONE_LINE_PROMPT.txt`

## 纯聊天型 AI

如果工具不会自动读取 `AGENTS.md`：

1. 先发 `ONE_LINE_PROMPT.txt`
2. 再发 `TEST_PROMPT.txt`
3. 需要时再补 `docs/PROJECT_LOCAL_CONTEXT.md`

## WeLink 适配原则

- AI 治理入口只是一层辅助，不替代根 `README.md` 和 `docs/README.md`
- 项目特有规则统一写在 `docs/PROJECT_LOCAL_CONTEXT.md`
- 当前仓库未发现专门的 doc-sync 脚本，因此不要求 AI 执行额外同步脚本
