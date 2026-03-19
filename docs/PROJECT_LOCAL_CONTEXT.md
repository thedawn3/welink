# WeLink 项目本地上下文

这份文档只记录 WeLink 项目特有的稳定事实。

## 1. 项目定位

- 项目名称：WeLink
- 一句话目标：把本地微信聊天数据串成“导入/解密准备/索引/分析展示/MCP 查询”的完整链路
- 当前阶段：`mixed`
- 主要交付对象：本地使用者、后续维护开发者、AI 协作代理

## 2. 关键路径

- 主入口文档：`README.md`
- 文档索引：`docs/README.md`
- 后端目录：`backend/`
- 前端目录：`frontend/`
- MCP 目录：`mcp-server/`
- 脚本目录：`scripts/`

## 3. 本项目硬约束

- 修改前先做远端一致性检查
- 不提交本机绝对路径、解密数据、媒体文件、`.env`
- 路径只允许走 `.env` / compose 环境变量，不允许写死个人机器路径
- 面向用户的前端文案默认中文
- 本地服务状态若 `is_initialized=false`，则说明当前实例仍需重新建索引

## 4. 最低验证命令

```bash
cd backend && go test ./...
cd frontend && npm run build
```

如果改动影响容器、启动或入口说明，再补：

```bash
docker compose up -d --build
curl http://localhost:8080/api/health
curl http://localhost:8080/api/status
```

## 5. 项目级代码风格与一致性规则

- 新增后端接口时，必须同步：
  - 路由
  - Swagger
  - `frontend/src/services/api.ts`
  - `frontend/src/types/index.ts`
- 联系人详情能力统一挂在 `frontend/src/components/contact/ContactModal.tsx`，不要再造平行详情入口
- 首页说明类改动必须同步：
  - `frontend/src/components/common/WelcomePage.tsx`
  - `README.md`
  - `docs/README.md`
  - `mcp-server/README.md`
  - 平台 setup / troubleshooting 文档
- 关系分析语义保持：
  - `score = 强度`
  - `confidence = 可信度`
- 联系人对外分类只保留：
  - `全部联系人`
  - `普通联系人`
  - `已删好友`

## 6. 高风险联动资产

- 关系分析：
  - 后端实现
  - 前端 overview/detail
  - `docs/relation-analysis.md`
- MCP：
  - 首页接入说明
  - `mcp-server/README.md`
  - 平台 setup 文档
- 启动与路径：
  - `.env.example`
  - `docker-compose.yml`
  - `docs/data-layout-and-troubleshooting.md`

## 7. 变更说明触发规则

以下变化应同步维护变更说明：

- 新增或调整用户可见功能入口
- 新增 HTTP API 或类型字段
- 改动 MCP 接入前置顺序
- 改动启动方式、路径契约、索引流程
- 改动关系分析算法口径或文案语义

变更说明建议落到 `docs/changes/`。

## 8. 新对话建议

- 默认先探索：`README.md`、`docs/README.md`、`docs/developer-workflow.md`
- 与 AI 协作相关的任务，再补读：`docs/AI_PROJECT_STARTER/`、`docs/PROJECT_LOCAL_CONTEXT.md`
- 如果信息不足，优先从仓库结构、现有文档和接口定义补事实，不先问用户
