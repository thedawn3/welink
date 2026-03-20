# WeLink 文档总览

## 快速上手

| 文档 | 说明 |
|---|---|
| [setup-macos.md](./setup-macos.md) | macOS 全流程：导入、解密准备、目录校验、启动 |
| [setup-windows.md](./setup-windows.md) | Windows 全流程：导入、解密准备、目录校验、启动 |
| [deploy-docker.md](./deploy-docker.md) | Docker 主文档：手动同步标准目录模式、source/analysis/work/msg 映射、`config-check -> runtime -> logs` 排障顺序 |
| [data-layout-and-troubleshooting.md](./data-layout-and-troubleshooting.md) | 标准目录契约（含可选 `sns/sns.db`）、环境变量语义、典型错误与修复 |
| [../README.md](../README.md) | 统一运行时、系统页、SSE `/api/events`、自动解密/刷新、ChatLab 导出总览（含新 env/config 字段） |
| [../mcp-server/README.md](../mcp-server/README.md) | MCP 接入：让 AI 直接查询本地聊天数据 |

## 开发与协作

| 文档 | 说明 |
|---|---|
| [developer-workflow.md](./developer-workflow.md) | fork/upstream、分支约定、baseline 同步流程 |
| [api.md](./api.md) | 后端 REST API 文档（system/chatlab 新端点以仓库根 README 与 mcp README 为准） |
| [indexing.md](./indexing.md) | 索引与初始化流程 |
| [database.md](./database.md) | contact/message 数据库结构 |

## AI 协作与变更说明

| 文档 | 说明 |
|---|---|
| [AI_PROJECT_STARTER/README.md](./AI_PROJECT_STARTER/README.md) | AI 协作入口、阅读顺序、工具使用方式 |
| [PROJECT_LOCAL_CONTEXT.md](./PROJECT_LOCAL_CONTEXT.md) | WeLink 项目特有规则、代码风格和联动约束 |
| [changes/README.md](./changes/README.md) | 变更说明索引与维护约定 |
| [changes/relationship-analysis-v2-and-mcp-onboarding.md](./changes/relationship-analysis-v2-and-mcp-onboarding.md) | 本轮关系分析 v2 / 聊天时间线 / 统一 runtime+MCP+ChatLab 整合交接 |

建议把 `relationship-analysis-v2-and-mcp-onboarding.md` 与根 `README.md` 搭配阅读：前者负责交接本轮 runtime / ingest / sync / MCP 整合细节，后者负责对外统一入口。

## 分析算法口径

| 文档 | 说明 |
|---|---|
| [relation-analysis.md](./relation-analysis.md) | 关系分析 v2：`score` vs `confidence`、久未联系衰减、榜单策略 |
| [sentiment.md](./sentiment.md) | 情感分析规则 |
| [wordcloud.md](./wordcloud.md) | 词云分词与过滤策略 |
