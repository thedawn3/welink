# 关系分析 v2 + 聊天时间线 + MCP 首页说明

## 变更摘要

- 分支：`codex/relationship-analysis-v2`
- 关键提交：`e50c4c0`
- 目标：把关系分析 v2、联系人聊天时间线、首页 MCP/AI 接入说明收敛成同一轮可维护改动

## 用户可见变化

- 首页新增 `AI / MCP 接入` 卡片
  - 明确顺序：先导入、再解密、再索引、最后连接 MCP
  - 明确未完成索引前，AI 查询结果会不完整
- 联系人详情新增 `聊天记录` tab
  - 按日期分组
  - 支持双方消息
  - 支持继续加载更早消息
  - 支持按某天定位
  - 热力图点击会联动到时间线
- 关系分析继续收敛
  - 首页统一 `客观模式 / 争议模式`
  - 详情显示 `confidence`、`stale_hint`、`confidence_reason`
  - 联系人前台分类只保留 `全部 / 普通 / 已删好友`

## 接口与类型变化

- 新增接口：`GET /api/contacts/messages/history`
  - 参数：`username`、`before`、`limit`
  - 用途：联系人详情时间线分页
- `ChatMessage` 新增 `timestamp`
- 前端 relation / controversy 相关类型补充：
  - `confidence`
  - `stale_hint`
  - `confidence_reason`

## 文档同步变化

- `README.md`：补 MCP 入口与 AI 协作导航
- `docs/README.md`：补 MCP、AI 协作、变更说明入口
- `mcp-server/README.md`：统一 MCP 前置顺序
- `docs/setup-macos.md`、`docs/setup-windows.md`、`docs/data-layout-and-troubleshooting.md`
  - 明确索引未完成前，AI / MCP 结果也会不完整

## 验证结果

- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `docker compose up -d --build`
- `GET /api/health`
- `GET /api/status`
- `GET /api/contacts/messages/history`

## 当前已知状态

- 本地服务已经按新版本重启
- 若当前实例 `is_initialized=false`，说明仍需重新建索引
- 在未完成索引前，前端分析结果与 MCP 查询结果都可能为空或不完整

## 后续维护提醒

- 以后只要改首页 MCP 说明、路径契约、索引流程或关系分析口径，都应回看本文件与 `docs/PROJECT_LOCAL_CONTEXT.md`
- 如果再新增联系人详情能力，优先继续并入 `ContactModal.tsx`，不要再开平行详情入口


## 统一 runtime / ingest / sync / ChatLab 整合

### 新增整合点

- 吸收 `chatlog_alpha` 的核心链路到 WeLink 内部：自动刷新、WAL 监听、revision 事件、ChatLab 导出、统一 MCP 控制
- 后端新增 `ingest / sync / runtime / export` 模块边界，不再让解密、刷新、索引状态分散在多套入口
- 前端新增“系统与同步”页，统一承载运行时状态、任务、日志、手动控制与 ChatLab 导出

### 自动刷新链路

- `sync` 监听 `message_*.db` 以及可选的 `-wal/-shm` 文件
- 连续文件事件经过 debounce 后合并成一个 revision
- revision 触发分析层自动重建，成功后 `data_revision` 单调递增
- 前端通过 `/api/events` SSE 感知变化，断开时回退轮询 `/api/system/runtime`
- 系统页现在会显式展示 SSE/轮询退化状态、最近事件时间、最近刷新时间与 watcher/WAL 状态

### 系统页与导出补强

- “系统与同步”页补了平台/目录/命令级解密启动参数，可直接控制 `auto_refresh` 与 `wal_enabled`
- 运行时日志支持按 source / level / 关键字筛选，并展示结构化 `fields`
- ChatLab 导出支持：
  - 联系人 `limit`
  - 群聊 `date`
  - 搜索 `include_mine` + `limit`
- ChatLab 响应新增 `summary`，便于前端直接提示导出规模（消息数、成员数、会话名）

### MCP 收口

- 保留单一 WeLink MCP，不再维护第二套 chatlog 独立 MCP
- MCP 新增 `get_runtime_status`、`start_decrypt`、`stop_decrypt`、`rebuild_index`、`get_recent_changes`、`export_chatlab`
- MCP 与前端共享同一组 `/api/system/*` 与 `/api/export/chatlab/*` 契约

### 手动语义变化

- `POST /api/init`：保留为“手动强制重建”兼容入口
- `POST /api/system/reindex`：作为统一 runtime 语义下的重建入口
- `POST /api/system/decrypt/start|stop`：作为平台解密/内置 stage 的显式控制入口

### 本轮补充验证

- `cd backend && go test ./...`
- `cd frontend && npm run build`
- `cd frontend && npm test`
- `cd mcp-server && go test ./...`

### 维护提醒补充

- 以后只要改 revision 语义、SSE 事件类型、ChatLab 导出结构或 MCP system 工具，都应回看本文件
- 如果前端刷新策略有变化，优先保持“局部刷新、不重置当前上下文”的体验约束
