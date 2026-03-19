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
