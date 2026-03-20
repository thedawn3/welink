# WeLink 文档总览

这套文档按“先部署成功，再看细节”的顺序组织。

## 推荐阅读顺序

1. [`../README.md`](../README.md)
   - 项目总入口
   - macOS / Windows 一键启动命令
   - AI 最短验收命令
2. [`ai-end-to-end-deploy-prompt.md`](./ai-end-to-end-deploy-prompt.md)
   - 给另一台机器上的 AI 的最短部署提示词
   - 覆盖“电脑微信原始数据 -> wechat-decrypt -> WeLink -> 验收”
3. [`deploy-docker.md`](./deploy-docker.md)
   - Docker 唯一主文档
   - `analysis-only` / `manual-sync` 两种正式模式
   - `.env`、挂载、`config-check -> runtime -> logs` 排障顺序
4. [`setup-macos.md`](./setup-macos.md) 或 [`setup-windows.md`](./setup-windows.md)
   - 只看各平台自己的前置条件、命令差异、常见坑
5. [`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md)
   - 统一的数据目录契约、图片 key、SNS、典型错误解释
6. [`../mcp-server/README.md`](../mcp-server/README.md)
   - 确认 WeLink 已启动并初始化后，再接 MCP

如果是另一台机器上的 AI 来拉仓并部署，先看 `README -> deploy-docker -> 对应平台 setup`，不要一开始分散阅读所有历史文档。

## 主文档分工

| 文档 | 负责什么 | 不负责什么 |
|---|---|---|
| [`../README.md`](../README.md) | 产品入口、两种正式模式、双平台一键命令、验收命令 | 不展开写所有 Docker 细节 |
| [`ai-end-to-end-deploy-prompt.md`](./ai-end-to-end-deploy-prompt.md) | 给 AI 的最短端到端部署提示词 | 不替代平台实操文档 |
| [`deploy-docker.md`](./deploy-docker.md) | Docker 模式、挂载、`.env`、红色阻塞错误、验证顺序 | 不重复讲平台通用背景 |
| [`setup-macos.md`](./setup-macos.md) | macOS 前置条件、命令、路径示例、平台注意事项 | 不重复解释完整 Docker 契约 |
| [`setup-windows.md`](./setup-windows.md) | Windows 前置条件、命令、PowerShell / Python / Compose v2 注意事项 | 不重复解释完整 Docker 契约 |
| [`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md) | 标准目录契约、图片 / SNS / source / analysis 排障 | 不承担平台启动教程 |
| [`../mcp-server/README.md`](../mcp-server/README.md) | MCP 接入顺序、客户端配置、工具说明 | 不承担部署入口 |

## 部署与使用

| 文档 | 说明 |
|---|---|
| [deploy-docker.md](./deploy-docker.md) | Docker 主文档，推荐所有新机器先看 |
| [ai-end-to-end-deploy-prompt.md](./ai-end-to-end-deploy-prompt.md) | 给另一台机器上的 AI 的最短提示词 |
| [setup-macos.md](./setup-macos.md) | macOS 一键部署补充 |
| [setup-windows.md](./setup-windows.md) | Windows 一键部署补充 |
| [data-layout-and-troubleshooting.md](./data-layout-and-troubleshooting.md) | 目录契约、挂载、图片 key、SNS、排障 |
| [../mcp-server/README.md](../mcp-server/README.md) | MCP 接入与验证 |

## 开发与协作

| 文档 | 说明 |
|---|---|
| [developer-workflow.md](./developer-workflow.md) | fork / upstream / 分支与协作流程 |
| [api.md](./api.md) | REST API 细节 |
| [indexing.md](./indexing.md) | 索引流程与数据初始化 |
| [database.md](./database.md) | contact / message / sns 数据结构 |
| [relation-analysis.md](./relation-analysis.md) | 关系分析口径 |
| [sentiment.md](./sentiment.md) | 情感分析规则 |
| [wordcloud.md](./wordcloud.md) | 词云规则 |

## AI 协作与项目规则

| 文档 | 说明 |
|---|---|
| [AI_PROJECT_STARTER/README.md](./AI_PROJECT_STARTER/README.md) | AI 协作入口 |
| [PROJECT_LOCAL_CONTEXT.md](./PROJECT_LOCAL_CONTEXT.md) | WeLink 项目本地规则 |
| [changes/README.md](./changes/README.md) | 变更说明索引 |

当前仓库的部署事实，以 `README.md`、`docs/deploy-docker.md`、`docs/data-layout-and-troubleshooting.md` 为准；如果其他历史文档与这三者冲突，应以后者为准。
