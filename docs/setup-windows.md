# Windows 使用指南

## 1. 先把手机聊天记录迁移到电脑微信

手机微信 -> 我 -> 设置 -> 通用 -> 聊天记录迁移与备份 -> 迁移到电脑。

## 2. 准备解密产物

默认仍建议使用 [ylytdeng/wechat-decrypt](https://github.com/ylytdeng/wechat-decrypt)。

请按该项目的 Windows 说明完成解密，最终准备出：

```text
decrypted/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

### Docker 模式补充（重要）

Docker 下建议使用手动同步标准目录模式，容器内不负责抓取原始微信目录。

- 容器外工具先准备标准目录（`contact/message`，可选 `sns`）
- 再按 [deploy-docker.md](./deploy-docker.md) 的映射方式启动 WeLink
- 启动后优先检查 `/api/system/config-check` 和 `/api/system/runtime`

## 3. 生成 .env 并校验目录

PowerShell 中执行：

```powershell
cd welink
.\scripts\welink-doctor.ps1 -WriteEnv
```

如需手动指定路径，建议使用正斜杠：

```powershell
.\scripts\welink-doctor.ps1 `
  -DataDir 'C:/Users/you/work/wechat-decrypt/decrypted_with_wal' `
  -MsgDir 'C:/Users/you/Documents/WeChat Files/wxid_xxx/msg' `
  -WriteEnv
```

### 可选：启用 WeLink 自动解密 / 自动刷新链路

Windows 是这一阶段自动刷新吸收最新消息的重点场景。若你希望 WeLink 启动后自动监听增量并触发重建，可在 `.env` 中确认这些值：

```env
WELINK_INGEST_ENABLED=true
WELINK_SOURCE_DATA_DIR=C:/Users/you/work/wechat-decrypt/decrypted_with_wal
WELINK_ANALYSIS_DATA_DIR=C:/Users/you/work/welink-analysis
WELINK_DECRYPT_ENABLED=true
WELINK_DECRYPT_AUTO_START=true
WELINK_SYNC_ENABLED=true
WELINK_SYNC_WATCH_WAL=true
```

说明：

- `WELINK_SOURCE_DATA_DIR` 可以指向带 `message_*.db-wal` 的目录
- `WELINK_ANALYSIS_DATA_DIR` 建议单独准备，避免分析过程与源目录互相干扰
- 若你配置了 `WELINK_WINDOWS_DECRYPT_COMMAND`，运行时会优先使用该命令；否则 provider 为 `builtin` 时会走内置 stage

## 4. 启动

```powershell
docker compose up --build
```

或者直接：

```powershell
.\scripts\start-welink.ps1
```

## 5. 校验

```powershell
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/status
```

若你改了 `.env` 里的端口，请把上面的 `8080` 替换成实际 `WELINK_BACKEND_PORT`。

优先观察 `/api/system/runtime`：

- `is_initialized=true` 表示当前索引已可供前端 / MCP 使用
- `decrypt_state` 可判断当前是否正在解密、已就绪或失败
- `data_revision` 单调递增，代表最新一次数据刷新版本

`/api/status` 仍保留为兼容接口；若你后面要接 MCP / AI，仍建议以 `/api/system/runtime` 为准。索引未完成前，AI 看到的数据也不完整。

如需观察实时刷新，可额外执行：

```powershell
curl -N http://localhost:8080/api/events
```

## 可选：启用自动解密 + 自动刷新链路

要让 Windows 一键启动解密命令并把 `analysis` 目录维持最新消息，可以向 `.env` 添加：

```env
WELINK_INGEST_ENABLED=true
WELINK_SOURCE_DATA_DIR=C:/Users/you/wechat/source
WELINK_ANALYSIS_DATA_DIR=C:/Users/you/wechat/analysis
WELINK_SYNC_ENABLED=true
WELINK_SYNC_WATCH_WAL=true
WELINK_DECRYPT_ENABLED=true
WELINK_DECRYPT_AUTO_START=true
WELINK_WINDOWS_DECRYPT_COMMAND="python decrypt.py --input ${source_data_dir} --output ${analysis_data_dir}"
```

该配置会先从源目录拉取 `.db`/`.db-wal`，`sync` 将它们 debounce 成 `revision` 并触发 `analysis` 重建；`runtime` 会实时更新 `decrypt_state` 与 `data_revision`。

如果你用的是 `auto_refresh=true` 但同步监听未配置或启动失败，解密任务仍可能成功启动；这时要优先看 `GET /api/system/logs` 里是否出现 `warn/sync`，而不是只看 `last_error`。

确认管道：

- `/api/system/runtime` 中 `decrypt_state` 走 `running`→`ready`
- `/api/system/changes` 中 `pending_changes` 回零，`data_revision` 递增
- `curl -N http://localhost:8080/api/events` 会看到 `runtime.revision.detected`/`runtime.decrypt.finished`

## Windows 注意事项

- `.env` 里的路径建议统一写成正斜杠，例如 `C:/Users/...`。
- 若 Docker Desktop 报挂载失败，先确认对应盘符已授权给 Docker。
- 若媒体目录缺失，可先留空 `WELINK_MSG_DIR`，不影响文本分析。
