<p align="center">
  <img src="logo.svg" width="80" height="80" alt="WeLink Logo" />
</p>

<h1 align="center">WeLink - 微信聊天记录导入与关系分析平台</h1>

WeLink 把本地微信数据整理成一条完整链路：

- 手机聊天记录迁移到电脑微信
- 解密准备（默认对接 `wechat-decrypt`）
- 数据目录校验与环境变量生成
- 本地索引、分析展示 + MCP 查询

当前正式支持平台：`macOS`、`Windows`。

## 快速入口

- macOS 使用指南：[docs/setup-macos.md](docs/setup-macos.md)
- Windows 使用指南：[docs/setup-windows.md](docs/setup-windows.md)
- Docker 部署指南：[docs/deploy-docker.md](docs/deploy-docker.md)
- 数据目录与故障排查：[docs/data-layout-and-troubleshooting.md](docs/data-layout-and-troubleshooting.md)
- 开发者工作流：[docs/developer-workflow.md](docs/developer-workflow.md)
- 关系分析算法口径：[docs/relation-analysis.md](docs/relation-analysis.md)
- MCP 配置：[mcp-server/README.md](mcp-server/README.md)
- AI 协作入口：[docs/AI_PROJECT_STARTER/README.md](docs/AI_PROJECT_STARTER/README.md)

## 核心能力

- 联系人、群聊、词云、情感、全局趋势分析
- 统一的关系分析体验：`客观模式 / 争议模式`
- 关系分析语义收敛：
  - `score` = 关系信号强度
  - `confidence` = 当前结论可信度
- 久未联系联系人会全局下调 `confidence`
- 前台联系人分类只保留 `全部联系人 / 普通联系人 / 已删好友`
- 支持 MCP，在 Claude Code 中直接自然语言查询本地微信数据

## 统一运行时（Docker 手动同步 + 本地自动链路）

当前 WeLink 内置统一运行时，支持两条正式 Docker 模式 + 一条本地高级模式：

- Docker 推荐：`analysis-only`（安全默认，只分析）
- Docker 推荐：`manual-sync`（校验并同步标准目录）
- 本地原生高级模式：`decrypt-first` 自动解密 + 自动刷新

统一链路核心流程：

1. 先启动运行时状态机（任务、日志、revision、事件总线）
2. 读取配置校验（目录、模式、平台能力）
3. 按模式执行手动同步或自动解密链路
4. revision 变化触发重建分析缓存，前端与 MCP 通过统一状态读取最新进度

第一阶段已覆盖能力：

- Windows 自动解密 + 自动刷新最新消息
- macOS 外部解密器统一编排接入
- `/api/events` SSE 实时事件
- ChatLab 标准导出（联系人 / 群聊 / 搜索）

后端内部职责已收口为一条统一链路：

- `ingest`：解密任务编排、平台命令调用、内置 stage 回落
- `sync`：监听 `*.db / *.db-wal / *.db-shm`、debounce 合并 revision、输出最近变更
- `analysis`：沿用现有联系人 / 群聊 / 关系分析服务，在 revision 后自动重建缓存
- `runtime`：统一维护状态、任务、日志、`data_revision`、`pending_changes`
- `mcp` / 前端：共同消费 `/api/system/*` 与 `/api/events`，不再维护第二套状态面

这意味着现在只有一个 WeLink 前端、一个 WeLink MCP：

- 前端在“系统与同步”页查看运行时、手动启动/停止解密、强制重建索引、直接导出 ChatLab
- MCP 通过同一组 system/export 接口执行 `get_runtime_status`、`start_decrypt`、`rebuild_index`、`export_chatlab`

## 多平台快速开始

### 1. 先把手机聊天记录迁移到电脑

手机微信 -> 我 -> 设置 -> 通用 -> 聊天记录迁移与备份 -> 迁移到电脑。

### 2. 准备解密产物

默认使用 [ylytdeng/wechat-decrypt](https://github.com/ylytdeng/wechat-decrypt)。

WeLink 只负责目录契约、校验、分析与展示，不内嵌第三方解密核心代码。
Docker 场景下，容器内不负责抓取微信原始目录，需由容器外工具先生成标准目录。
当前仓库已实际验证 `ylytdeng/wechat-decrypt` 可产出 `sns/sns.db`，因此 Docker 文档与状态面按“已验证支持 SNS”处理。

期望目录：

```text
decrypted/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db            # optional
```

### 3. 生成 `.env`

macOS:

```bash
./scripts/welink-doctor.sh --write-env
```

PowerShell:

```powershell
.\scripts\welink-doctor.ps1 -WriteEnv
```

### 4. 启动

如果你希望看完整的 Docker / Compose 启动说明、`.env` 生成、脚本方式和端口冲突处理，优先看 [docs/deploy-docker.md](docs/deploy-docker.md)。

```bash
docker compose up --build
```

也可以直接用一键脚本：

- macOS：`./scripts/start-welink.sh`
- PowerShell：`.\scripts\start-welink.ps1`

- 默认前端：`http://localhost:3000`
- 默认后端：`http://localhost:8080`
- 若你在 `.env` 里改了 `WELINK_FRONTEND_PORT` / `WELINK_BACKEND_PORT`，以 `.env` 为准

### 5. 校验

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/status
```

如果你改了 `.env` 中的 `WELINK_BACKEND_PORT`，把这里的 `8080` 换成实际端口即可。

可选检查（SSE）：

```bash
curl -N http://localhost:8080/api/events
```

## 环境变量

项目使用 `.env` 驱动配置，优先级为：默认值 < `config.yaml` < 环境变量。

### 基础变量

```env
WELINK_DATA_DIR=/absolute/path/to/decrypted
WELINK_MSG_DIR=/absolute/path/to/msg
WELINK_BACKEND_PORT=8080
WELINK_FRONTEND_PORT=3000
```

- `WELINK_DATA_DIR`：必填
- `WELINK_MSG_DIR`：可选；缺失时会使用仓库内空目录占位

### Docker 推荐模板（手动同步）

```env
WELINK_MODE=analysis-only
WELINK_ANALYSIS_DATA_DIR=/absolute/path/to/standard_dir
WELINK_DATA_DIR=/absolute/path/to/standard_dir
WELINK_SOURCE_DATA_DIR=
WELINK_WORK_DIR=./.tmp/welink-workdir
WELINK_INGEST_ENABLED=false
WELINK_DECRYPT_ENABLED=false
WELINK_DECRYPT_AUTO_START=false
WELINK_SYNC_ENABLED=false
```

切换到 `manual-sync` 时，至少改为：

```env
WELINK_MODE=manual-sync
WELINK_SOURCE_DATA_DIR=/absolute/path/to/source_standard_dir
```

### 运行时 / ingest / sync / decrypt 变量

```env
# ingest
WELINK_INGEST_ENABLED=false
WELINK_SOURCE_DATA_DIR=
WELINK_WORK_DIR=./workdir
WELINK_ANALYSIS_DATA_DIR=
WELINK_PLATFORM=auto

# sync
WELINK_SYNC_ENABLED=false
WELINK_SYNC_WATCH_WAL=true
WELINK_SYNC_DEBOUNCE_MS=1000
WELINK_SYNC_MAX_WAIT_MS=10000
WELINK_SYNC_EVENT_BUFFER=128

# decrypt
WELINK_DECRYPT_ENABLED=false
WELINK_DECRYPT_AUTO_START=false
WELINK_DECRYPT_PROVIDER=builtin
WELINK_WINDOWS_DECRYPT_COMMAND=
WELINK_MAC_DECRYPT_COMMAND=
WELINK_LINUX_DECRYPT_COMMAND=
WELINK_DECRYPT_PRESERVE_WAL=false
WELINK_DECRYPT_TIMEOUT_SECONDS=120

# runtime
WELINK_RUNTIME_ENGINE_TYPE=welink
WELINK_RUNTIME_MAX_TASK_RECORDS=200
WELINK_RUNTIME_MAX_LOG_RECORDS=1000
```

解密命令模板支持占位符：

- `${platform}`
- `${source_data_dir}`
- `${analysis_data_dir}`
- `${work_dir}`
- `${auto_refresh}`
- `${wal_enabled}`

### 本地原生高级链路（可选）

```env
WELINK_INGEST_ENABLED=true
WELINK_SYNC_ENABLED=true
WELINK_DECRYPT_ENABLED=true
WELINK_DECRYPT_AUTO_START=true
```

Windows/macOS 可分别配置平台命令：

- `WELINK_WINDOWS_DECRYPT_COMMAND`
- `WELINK_MAC_DECRYPT_COMMAND`

未配置对应平台命令时：

- 若 `WELINK_DECRYPT_PROVIDER=builtin`，运行时会改走内置 stage 模式，把 `WELINK_SOURCE_DATA_DIR` 同步到 `WELINK_ANALYSIS_DATA_DIR`
- 若 provider 不是 `builtin`，运行时会报告 `no decrypt command configured`

## 统一 System API（运行时 / 校验 / 控制）

| 接口 | 说明 |
|---|---|
| `GET /api/system/config-check` | 统一配置/目录校验（部署目标、模式、source/analysis/work/sns 就绪度、建议动作） |
| `GET /api/system/runtime` | 统一运行时状态（`engine_type`、`decrypt_state`、`data_revision`、`last_error` 等） |
| `GET /api/system/tasks` | 运行时任务列表（解密、reindex 等） |
| `GET /api/system/logs` | 运行时日志；支持按任务过滤 |
| `GET /api/system/changes` | 最近 revision 与变更摘要（含同步状态） |
| `POST /api/system/decrypt/start` | 启动解密任务（支持覆盖 command/platform/path） |
| `POST /api/system/decrypt/stop` | 停止当前解密任务 |
| `POST /api/system/reindex` | 触发重建索引 |
| `GET /api/events` | SSE 实时事件流（状态变更、解密状态、reindex 事件） |

兼容接口仍保留：

- `GET /api/status`（返回统一运行时状态）
- `POST /api/init`（手动强制重建）

上述接口与 `/api/events` SSE 由前端与 MCP 共享，确保解密/自动刷新与 revision 变化对所有客户端一致。

## ChatLab 导出

WeLink 现已支持 ChatLab 标准格式导出：

- `GET /api/export/chatlab/contact?username=...&limit=200`
- `GET /api/export/chatlab/group?username=...&date=YYYY-MM-DD`
- `GET /api/export/chatlab/search?q=关键词&include_mine=true&limit=200`
- `POST /api/export/chatlab`（统一入口，`scope=contact/group/search`）

当 query 参数 `download=true` 时，后端会返回带文件名的下载响应头。

前端“系统与同步”页与 MCP `export_chatlab` 工具都直接走这组导出接口，不再维护平行导出链路。

## Fork 后开发流程

1. fork 上游 `runzhliu/welink`
2. clone 个人 fork 到正式工作目录
3. 配置 `origin` / `upstream`
4. 使用 `codex/*` 分支
5. 先做 baseline 同步，再做功能增量提交

详见 [docs/developer-workflow.md](docs/developer-workflow.md)。

## AI 协作入口

如果你让 Codex、Claude Code 等 AI 工具直接维护 WeLink，建议从这些入口开始：

- 仓库级入口：`AGENTS.md`
- AI 协作说明：`docs/AI_PROJECT_STARTER/README.md`
- 项目本地上下文：`docs/PROJECT_LOCAL_CONTEXT.md`
- 本轮改动交接：`docs/changes/relationship-analysis-v2-and-mcp-onboarding.md`

## MCP

如果你要把 WeLink 接给 AI，当成 MCP 数据源，前置顺序固定为：

1. 先确认聊天记录已经完整导入到电脑微信
2. 再完成解密，拿到标准数据库目录
3. 启动 WeLink 并等待索引完成
4. 最后再连接 MCP 客户端

接入后，Claude Code 等 AI 客户端可以直接查询和分析本地聊天数据，并通过 [mcp-server/README.md](mcp-server/README.md) 提供的统一工具调度解密/重建流程。

注意：索引没完成前，MCP 查询结果也会不完整。

详见 [mcp-server/README.md](mcp-server/README.md)。

## Demo 模式

```bash
docker compose -f docker-compose.demo.yml up
```

## 数据安全

所有分析默认在本地执行，不上传聊天数据。请仅处理你有权限访问的数据。
