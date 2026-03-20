# WeLink MCP Server

让 Claude Code 等 AI 客户端把 WeLink 当作 MCP 数据源，直接查询和分析你的本地微信聊天数据。

## 前提条件

在配置 MCP 之前，前置顺序固定为：

1. 先确认聊天记录已经完整导入到电脑微信
2. 再按平台文档完成解密，拿到标准数据库目录（`contact/message`，可选 `sns`）
3. 运行 `welink-doctor` 校验目录并生成 `.env`
4. 启动 WeLink，先检查 `GET /api/system/config-check`，再确认 `GET /api/system/runtime` 中 `is_initialized=true`
5. 本次双平台验收优先走 Docker 的 `analysis-only` / `manual-sync` 两种正式模式
6. 最后再连接 MCP 客户端

兼容模式下也可检查 `GET /api/status`（同样返回统一运行时状态）。

部署文档：

- [../README.md](../README.md)
- [../docs/deploy-docker.md](../docs/deploy-docker.md)
- [../docs/setup-macos.md](../docs/setup-macos.md)
- [../docs/setup-windows.md](../docs/setup-windows.md)

## 构建

```bash
cd mcp-server
go build -o welink-mcp .
```

## 配置 Claude Code

完成前置步骤后，再把 MCP 接到客户端：

编辑 `~/.claude.json`：

```json
{
  "mcpServers": {
    "welink": {
      "command": "/你的路径/welink/mcp-server/welink-mcp",
      "env": {
        "WELINK_URL": "http://localhost:8080"
      }
    }
  }
}
```

也可以直接用命令行：

```bash
claude mcp add welink /你的路径/welink/mcp-server/welink-mcp -e WELINK_URL=http://localhost:8080
```

如果你改了 `.env` 中的 `WELINK_BACKEND_PORT`，把上面的 `8080` 改成实际端口。

## MCP 工具能力（统一版）

现有 MCP 同时提供分析工具和 system 控制工具，并直接复用 WeLink 前端正在使用的 `/api/system/*` + `/api/export/chatlab/*` 契约。

### 分析类

- `get_contact_stats`
- `get_contact_detail`
- `get_contact_wordcloud`
- `get_contact_sentiment`
- `get_contact_messages`
- `get_global_stats`
- `get_groups`
- `get_group_detail`
- `get_stats_by_timerange`

### 运行时与控制类

- `get_runtime_status`
- `start_decrypt`
- `stop_decrypt`
- `rebuild_index`
- `get_recent_changes`
- `export_chatlab`

## 后端接口映射

| MCP 工具 | 后端接口 | 行为 |
|---|---|---|
| `get_runtime_status` | `GET /api/system/runtime` | 缺失即报错 |
| `start_decrypt` | `POST /api/system/decrypt/start` | 缺失即报错 |
| `stop_decrypt` | `POST /api/system/decrypt/stop` | 缺失即报错 |
| `rebuild_index` | `POST /api/system/reindex` | 缺失即报错 |
| `get_recent_changes` | `GET /api/system/changes` | 缺失即报错 |
| `export_chatlab` | `GET /api/export/chatlab/{contact\|group\|search}` | 缺失即报错 |

说明：

- 当前 MCP 直接对齐 WeLink 的统一 `system/export` 契约：运行时状态与控制走 `/api/system/*`，ChatLab 导出走 `/api/export/chatlab/*`。
- 本次双平台部署建议优先使用 Docker 的 `analysis-only` / `manual-sync` 两种正式模式；MCP 也基于这两种模式工作。
- `start_decrypt` 调用前，建议先通过 `GET /api/system/config-check` 确认目录和模式已就绪。
- `get_recent_changes` / `rebuild_index` 建议结合 `data_revision`、`pending_changes`、`last_reindex_at` 一起判断。

补充说明：

- `start_decrypt` 对应 `POST /api/system/decrypt/start`；当目录或模式校验失败时，会返回 `400` 与可执行错误信息。
- `stop_decrypt` 对应 `POST /api/system/decrypt/stop`；若当前没有进行中的解密任务，会返回 `200` no-op，而不是报错。
- `GET /api/system/logs?task_id=...` 返回的是该解密任务的 orchestrator 日志，不混入统一运行时日志。
- `task_id` 日志按产生顺序返回（最早 -> 最晚），`limit` 表示最早的前 N 条，而不是最近 N 条。

## 建议先做的后端探活

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/system/changes
```


## 手动恢复建议

遇到这几类情况时，优先使用 system 工具而不是重启整套服务：

- `decrypt_state=error`：先看 `get_runtime_status` 中的 `last_error`，再调用 `start_decrypt`
- `pending_changes` 长时间不回落：先 `get_recent_changes`，必要时 `rebuild_index`
- 平台解密任务卡住：先 `get_runtime_status` / `GET /api/system/tasks` 确认状态，再调用 `stop_decrypt`；即使当前没有进行中的任务，也会返回安全 no-op

恢复成功后，重点观察：

- `last_decrypt_at`：最近一次成功解密/内置 stage 完成时间
- `last_reindex_at`：最近一次成功重建索引时间
- `data_revision`：数据版本是否继续单调递增

## 接入后的作用

- AI 可以直接查询联系人、消息统计、关系分析和关键词结果
- AI 可以基于 WeLink 已完成的本地索引做总结、筛选和对比
- AI 可以通过 MCP 触发解密任务、重建索引并查看运行时变化
- AI 可以直接导出 ChatLab 标准数据用于后续分析链路
- 如果索引尚未完成，AI 看到的数据也会为空或不完整

## 确认加载

Claude Code 中执行：

```text
/mcp
```

应看到 `welink` 状态为 connected。

## 推荐 Skills 配置

把以下片段加入 `~/.claude/CLAUDE.md`：

```markdown
## WeLink MCP

当用户询问微信聊天数据、社交关系、消息统计、聊天记录时，
主动使用 WeLink MCP 工具（welink）来回答。
```

## 常见问题

**后端无法访问**
- 先确认 `docker compose up --build` 正常运行
- 再确认 `curl http://localhost:8080/api/health` 返回正常
- 若你改了端口，则以 `.env` 中的 `WELINK_BACKEND_PORT` 为准

**返回数据为空**
- 通常是索引尚未完成，优先检查 `/api/system/runtime`
- 也可能是电脑微信本机记录本身不完整

**MCP 已连接但分析结果不全**
- 先回到平台文档，确认电脑微信里确实已有完整聊天记录
- 再重新解密并运行 `welink-doctor`

**start_decrypt 后没有看到数据变化**
- 先回到 `GET /api/system/config-check`，确认当前模式是否真的支持这次操作
- Docker 正式模式推荐 `analysis-only` / `manual-sync`，通常以手动同步与重建索引为准
- 若你走的是本地高级链路，再看 `GET /api/system/logs` 与 `GET /api/system/changes`

**system 工具报接口不存在**
- 说明后端尚未上线 `/api/system/*` 或 `/api/export/chatlab/*` 的统一契约
- 完整能力仍建议升级后端到 `/api/system/*` + `/api/export/chatlab*`，因为 MCP 不再保留旧 fallback 实现
