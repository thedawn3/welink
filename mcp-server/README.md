# WeLink MCP Server

让 Claude Code 等 AI 客户端把 WeLink 当作 MCP 数据源，直接查询和分析你的本地微信聊天数据。

## 前提条件

在配置 MCP 之前，前置顺序固定为：

1. 先确认聊天记录已经完整导入到电脑微信
2. 再按平台文档完成解密，拿到标准数据库目录
3. 运行 `welink-doctor` 校验目录并生成 `.env`
4. 启动 WeLink，并等待索引完成：`GET /api/status` 中 `is_initialized=true`
5. 最后再连接 MCP 客户端

平台文档：

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

## 接入后的作用

- AI 可以直接查询联系人、消息统计、关系分析和关键词结果
- AI 可以基于 WeLink 已完成的本地索引做总结、筛选和对比
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
- 通常是索引尚未完成，先检查 `/api/status`
- 也可能是电脑微信本机记录本身不完整

**MCP 已连接但分析结果不全**
- 先回到平台文档，确认电脑微信里确实已有完整聊天记录
- 再重新解密并运行 `welink-doctor`
