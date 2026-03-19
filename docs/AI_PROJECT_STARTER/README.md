# WeLink AI 协作入口

这组文档是给 Codex / Claude Code / 其它代理式 AI 工具使用的精简治理入口。

目标不是再造一套并行文档，而是让 AI 在新对话里更快进入 WeLink 的正确工作方式：

- 先探索，再修改
- 先做远端一致性检查
- 优先复用现有文档和实现，不平行造新体系
- 改动行为、接口、路径或文档入口时，顺手补齐联动资产

## 阅读顺序

1. `PROJECT_AI_PLAYBOOK.md`
2. `CODEX_RULES.md`
3. `../PROJECT_LOCAL_CONTEXT.md`

## 文件分工

- `PROJECT_AI_PLAYBOOK.md`：通用 AI 协作原则
- `CODEX_RULES.md`：Codex 的精简命中层
- `USAGE_BY_AI_TOOL.md`：不同 AI 工具的推荐使用方式
- `ONE_LINE_PROMPT.txt`：新会话第一句提示词
- `TEST_PROMPT.txt`：校验当前会话是否已经命中规则

## WeLink 特别说明

- 本仓库已经有稳定的主入口：根 `README.md`
- 本仓库已经有稳定的文档索引：`docs/README.md`
- WeLink 项目特有规则、代码风格、联动约束，不写在 starter 文档里，统一写在 `docs/PROJECT_LOCAL_CONTEXT.md`
- 当前仓库未发现独立 doc-sync 脚本，因此相关步骤默认跳过
