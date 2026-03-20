# Docker 部署与手动同步（主文档）

本页是 Docker 场景的唯一主入口，适用于 `thedawn3/welink`。

当前 Docker 只保留两种正式模式：

- `analysis-only`：只消费 analysis directory，不提供 source standard directory
- `manual-sync`：提供 source standard directory，系统页执行“校验并同步标准目录”

共同原则：

- 容器外工具负责准备标准目录（聊天/朋友圈）
- WeLink 容器内只做校验、手动同步、索引与分析
- 默认不启用容器内 watcher，不自动启动解密
- `ylytdeng/wechat-decrypt` 已在本机链路实测可产出 `sns/sns.db`

## 1. 目录契约与挂载模型

### 标准目录契约（source / analysis 都遵循）

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # 可选
```

### Docker 挂载语义

- `WELINK_SOURCE_DATA_DIR` -> `/app/source-data`（source standard directory，只读）
- `WELINK_ANALYSIS_DATA_DIR` -> `/app/analysis-data`（analysis directory，可写）
- `WELINK_WORK_DIR` -> `/app/workdir`（work directory，可写）
- `WELINK_MSG_DIR` -> `/app/msg`（media directory，只读，可空）

默认行为：

- `WELINK_SOURCE_DATA_DIR` 为空时，compose 自动挂载 `./.tmp/welink-empty-source`
- 不会再默认回退到 `./decrypted`，避免 source / analysis 混用

## 2. canonical 模式 A：analysis-only（默认）

### macOS / Linux

```bash
./scripts/welink-doctor.sh \
  --mode analysis-only \
  --data-dir /absolute/path/to/standard_dir \
  --msg-dir /absolute/path/to/msg \
  --write-env

docker compose up -d --build
```

### Windows PowerShell

```powershell
.\scripts\welink-doctor.ps1 `
  -Mode analysis-only `
  -DataDir 'C:/absolute/path/to/standard_dir' `
  -MsgDir 'C:/absolute/path/to/msg' `
  -WriteEnv

docker compose up -d --build
```

doctor 在 `analysis-only` 下会生成安全默认：

- `WELINK_INGEST_ENABLED=false`
- `WELINK_DECRYPT_ENABLED=false`
- `WELINK_DECRYPT_AUTO_START=false`
- `WELINK_SYNC_ENABLED=false`
- `WELINK_SOURCE_DATA_DIR=`（留空，走空占位挂载）

适用场景：

- 你已经有可分析的标准目录，只需要浏览 / 搜索 / 关系分析
- 你不希望在容器内保留任何 source 同步动作

## 3. canonical 模式 B：manual-sync

### macOS / Linux

```bash
./scripts/welink-doctor.sh \
  --mode manual-sync \
  --data-dir /absolute/path/to/analysis_standard_dir \
  --source-data-dir /absolute/path/to/source_standard_dir \
  --msg-dir /absolute/path/to/msg \
  --write-env

docker compose up -d --build
```

### Windows PowerShell

```powershell
.\scripts\welink-doctor.ps1 `
  -Mode manual-sync `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir' `
  -SourceDataDir 'C:/absolute/path/to/source_standard_dir' `
  -MsgDir 'C:/absolute/path/to/msg' `
  -WriteEnv

docker compose up -d --build
```

适用场景：

- 容器外工具会持续更新 source standard directory
- 你希望保留“先校验，再手动同步/重建”的收敛路径

## 4. `.env` 模板

```env
WELINK_BACKEND_PORT=8080
WELINK_FRONTEND_PORT=3000
WELINK_GIN_MODE=release

# analysis-only
WELINK_ANALYSIS_DATA_DIR=/absolute/path/to/standard_dir
WELINK_DATA_DIR=/absolute/path/to/standard_dir
WELINK_SOURCE_DATA_DIR=
WELINK_WORK_DIR=./.tmp/welink-workdir
WELINK_MSG_DIR=/absolute/path/to/msg
WELINK_MODE=analysis-only
WELINK_INGEST_ENABLED=false
WELINK_DECRYPT_ENABLED=false
WELINK_DECRYPT_AUTO_START=false
WELINK_SYNC_ENABLED=false

# manual-sync（切换时至少改这两项）
# WELINK_MODE=manual-sync
# WELINK_SOURCE_DATA_DIR=/absolute/path/to/source_standard_dir
```

## 5. 配置/状态检查顺序

启动后建议固定按这个顺序排障：

1. `GET /api/system/config-check`（目录与配置是否可操作）
2. `GET /api/system/runtime`（当前状态、最近消息时间、SNS 时间、错误）
3. `GET /api/system/logs`（任务日志与详细报错）
4. 必要时 `POST /api/system/decrypt/start`（先过 config-check）
5. 如需强制重建：`POST /api/system/reindex`

示例：

```bash
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/system/logs
```

## 6. 红色阻塞错误什么时候会出现

| 场景 | 是否红色阻塞 | 说明 |
|---|---|---|
| `analysis-only` + `source` 为空 + `analysis` 就绪 | 否 | 正常可用状态，不是错误 |
| `manual-sync` + `source` 不是标准目录 | 是 | 必须包含 `contact/contact.db` 与 `message/message_*.db` |
| `manual-sync` + `source/analysis` 同目录 | 是 | 会污染分析目录，后端会阻止启动 |
| `manual-sync` + `work_dir` 不可写 | 是 | 内置 stage 无法运行 |

## 7. Docker 场景常见错误

### 报错：`no contact/message databases found under /app/source-data`

原因通常是：

- 你把 `xwechat_files` 根目录挂成了 source
- 该目录不符合标准契约（缺少 `contact/message`）

处理：

- 容器外先整理成标准目录，再挂载到 `WELINK_ANALYSIS_DATA_DIR`
- Docker 手动同步模式下建议保持 `WELINK_SOURCE_DATA_DIR` 为空

### source / analysis 指向同一路径

这会导致同步与分析互相污染，后端会阻止启动并返回可执行错误说明。请分离目录或保持 source 为空（手动同步模式）。

### 端口冲突

```bash
lsof -nP -iTCP:8080 -sTCP:LISTEN
lsof -nP -iTCP:3000 -sTCP:LISTEN
```

然后修改 `.env`：

```env
WELINK_BACKEND_PORT=18080
WELINK_FRONTEND_PORT=13000
```

### Docker Desktop 挂载失败

- macOS: 确认目录已在 Docker Desktop 文件共享列表中
- Windows: 确认盘符授权、路径使用正斜杠（`C:/...`）

## 8. 容器外工具边界（重要）

Docker 模式下，WeLink 不负责在容器内抓取微信原始目录或执行平台特定抓取逻辑。

建议流程：

1. 容器外工具生成标准目录（`contact/message`，可选 `sns`）
2. Docker 启动 WeLink
3. 在系统页执行“校验并同步标准目录”或手动重建索引

## 9. 与其他文档的关系

- macOS 平台流程：[setup-macos.md](./setup-macos.md)
- Windows 平台流程：[setup-windows.md](./setup-windows.md)
- 目录契约与排障详情：[data-layout-and-troubleshooting.md](./data-layout-and-troubleshooting.md)
- API 文档：[api.md](./api.md)
