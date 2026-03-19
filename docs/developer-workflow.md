# 开发者工作流

## 仓库基线

- 上游：`runzhliu/welink`
- 个人 fork：`<your-account>/welink`
- 本地正式工作目录：建议 `projects/welink`
- 临时验证目录：仅用于实验，不作为长期基线

## 推荐流程

```bash
gh repo fork runzhliu/welink --clone=false
git clone https://github.com/<you>/welink.git
cd welink
git remote add upstream https://github.com/runzhliu/welink.git
git checkout -b codex/<topic>
```

## 分支与提交约定

- 分支前缀：`codex/`
- 先做 `baseline sync` 提交，再做功能增量提交
- 不提交本机绝对路径、解密数据、媒体文件、`.env`

## 本地配置

- 把 `.env.example` 复制成 `.env`
- 路径统一放进 `.env`
- `docker-compose.yml` 不允许再写死个人机器路径

## 提交前检查

```bash
cd backend && go test ./...
cd ../frontend && npm ci && npm run build
```

## 文档要求

只要改了以下任一能力，就同步文档：

- 启动方式 / 环境变量 / 目录契约
- 关系分析口径
- MCP 前置依赖
- 跨平台脚本与排障流程

## AI 接手入口

WeLink 的 AI 协作入口固定分三层：

- `AGENTS.md`：仓库级自动入口
- `docs/AI_PROJECT_STARTER/`：通用 AI 协作规则与使用方式
- `docs/PROJECT_LOCAL_CONTEXT.md`：WeLink 项目特有事实、代码风格和联动约束

职责分层：

- starter 文档只放跨项目可复用的协作原则
- 项目特有规则统一放到 `docs/PROJECT_LOCAL_CONTEXT.md`
- 产品和运行入口仍然以根 `README.md` 与 `docs/README.md` 为准

## 哪类变更必须同步变更说明

以下变化默认需要同步 `docs/changes/`：

- 新增或调整用户可见入口
- 新增后端 API、前端关键类型字段
- 改动 MCP 接入前置顺序
- 改动启动方式、路径契约、索引流程
- 改动关系分析算法口径或用户可见文案语义
