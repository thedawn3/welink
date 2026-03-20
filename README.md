<p align="center">
  <img src="logo.svg" width="80" height="80" alt="WeLink Logo" />
</p>

<h1 align="center">WeLink - 微信聊天记录导入、分析与 MCP 查询平台</h1>

WeLink 把本地微信数据库整理成一条稳定链路：

- 校验标准目录：`contact/contact.db` + `message/message_*.db`，可选 `sns/sns.db`
- 启动统一前后端：联系人、群聊、关系分析、聊天时间线、系统状态
- 支持聊天图片索引：聊天记录里可显示缩略图并点击查看
- 支持 MCP：让 Claude Code 等 AI 直接查询本地微信数据

当前对外推荐的正式部署路径是：`macOS / Windows + Docker`。

## 先看哪份文档

- 一键部署入口：`README.md`（本页）
- Docker 主文档：[`docs/deploy-docker.md`](docs/deploy-docker.md)
- macOS 平台说明：[`docs/setup-macos.md`](docs/setup-macos.md)
- Windows 平台说明：[`docs/setup-windows.md`](docs/setup-windows.md)
- 数据目录与图片排障：[`docs/data-layout-and-troubleshooting.md`](docs/data-layout-and-troubleshooting.md)
- MCP 接入：[`mcp-server/README.md`](mcp-server/README.md)
- 文档索引：[`docs/README.md`](docs/README.md)

## AI 验收最短路径

如果你要让另一台 `macOS` 或 `Windows` 电脑上的 AI 直接拉仓并一键部署，先只记住这几件事：

1. Docker 只有两种正式模式：`analysis-only` 和 `manual-sync`
2. `manual-sync` 的 `source` 必须是标准目录，不能直接指向原始 `xwechat_files` 根目录
3. `msg` 和 `wechat-decrypt` 都是可选增强项，不是启动必需项
4. Windows 额外需要：`PowerShell 脚本放行 + Python 3 在 PATH 中`

### 验收前提

- 已安装 Git
- 已安装 Docker Desktop，且 `docker compose version` 可用
- 已安装 Python 3
- 已准备好一个标准目录：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

可选增强目录：

- `msg`：聊天图片缩略图 / 点击查看依赖它
- `wechat-decrypt`：用于自动读取 `image_keys.json` / `config.json`，改善 2025-08+ V2 图片预览

### 最短命令：manual-sync

适合“容器外工具会继续更新 source 标准目录，我需要在 WeLink 里手动同步”。

macOS / Linux：

```bash
git clone https://github.com/thedawn3/welink.git
cd welink
./scripts/start-welink.sh \
  --mode manual-sync \
  --data-dir /absolute/path/to/analysis_standard_dir \
  --source-data-dir /absolute/path/to/source_standard_dir
```

Windows PowerShell：

```powershell
git clone https://github.com/thedawn3/welink.git
cd welink
Set-ExecutionPolicy -Scope Process Bypass -Force
.\scripts\start-welink.ps1 `
  -Mode manual-sync `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir' `
  -SourceDataDir 'C:/absolute/path/to/source_standard_dir'
```

如果你同时有媒体目录和 `wechat-decrypt` 工具目录，可继续追加：

- macOS：`--msg-dir /absolute/path/to/msg --wechat-decrypt-dir /absolute/path/to/wechat-decrypt`
- Windows：`-MsgDir 'C:/absolute/path/to/msg' -WechatDecryptDir 'C:/absolute/path/to/wechat-decrypt'`

### 最短命令：analysis-only

适合“我已经有一个可分析的标准目录，只想直接启动浏览/分析”。

macOS / Linux：

```bash
./scripts/start-welink.sh \
  --mode analysis-only \
  --data-dir /absolute/path/to/analysis_standard_dir
```

Windows PowerShell：

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
.\scripts\start-welink.ps1 `
  -Mode analysis-only `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir'
```

### 启动后只做这 4 条验收

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/status
```

判定标准：

- `/api/health` 返回正常
- `/api/system/config-check` 没有顶层阻塞错误
- `/api/system/runtime` 中 `is_initialized=true`
- 前端可打开 `http://localhost:3000`

## Docker 正式模式

### 模式 A：`analysis-only`

- 只消费 `analysis directory`
- 不需要 `source standard directory`
- Docker 推荐默认模式
- 适合“已经有现成标准目录，只做分析”

### 模式 B：`manual-sync`

- 需要独立的 `analysis directory` 与 `source standard directory`
- source 必须也是标准目录根
- 容器内不跑 watcher，不自动抓微信原始目录
- 适合“容器外工具持续更新 source，WeLink 负责校验 / 手动同步 / 重建索引”

Docker 主文档见：[`docs/deploy-docker.md`](docs/deploy-docker.md)

## 图片能力说明

如果你希望聊天记录中直接看到图片缩略图并可点击查看：

- 配置 `WELINK_MSG_DIR` 指向微信 `msg` 目录
- 推荐同时配置 `WELINK_WECHAT_DECRYPT_DIR`
- WeLink 会按以下优先级自动读取 V2 图片密钥：
  1. `image_keys.json`
  2. `config.json.image_aes_key`
  3. `WELINK_IMAGE_AES_KEY`

当前仓库已实测：`ylytdeng/wechat-decrypt` 能产出 `sns/sns.db`，也能配合 `find_image_key` 产出 `image_keys.json`。

如需补齐图片 key，可在宿主机执行：

```bash
./scripts/extract-image-key.sh --restart
```

更详细说明见：[`docs/data-layout-and-troubleshooting.md`](docs/data-layout-and-troubleshooting.md)

## MCP 能力

WeLink 只保留一套 MCP：

- 查询联系人、群聊、聊天记录、时间范围统计
- 查看运行时状态、最近变更、索引状态
- 触发重建索引、导出 ChatLab

接入方法见：[`mcp-server/README.md`](mcp-server/README.md)

## 常见误区

- `source` 不能直接填原始 `xwechat_files` 根目录
- `manual-sync` 下不要把 `source` 和 `analysis` 指向同一路径
- Windows 下 `Set-ExecutionPolicy -Scope Process Bypass -Force` 只对当前 PowerShell 会话生效；新开窗口要重新执行
- Windows 下 `scripts/welink-doctor.ps1` 依赖 `py -3` 或 `python`
- `msg` 与 `wechat-decrypt` 缺失不会阻止文本分析，只会影响图片体验

## 项目能力概览

- 联系人、群聊、关系分析、搜索、聊天时间线
- 朋友圈数据目录识别与状态展示（`sns.db`）
- ChatLab 导出
- 统一运行时状态面：`config-check / runtime / logs / tasks`
- MCP 与前端共享同一套后端契约

## 开发与协作入口

- 文档索引：[`docs/README.md`](docs/README.md)
- 开发工作流：[`docs/developer-workflow.md`](docs/developer-workflow.md)
- 关系分析算法口径：[`docs/relation-analysis.md`](docs/relation-analysis.md)
- AI 协作入口：[`docs/AI_PROJECT_STARTER/README.md`](docs/AI_PROJECT_STARTER/README.md)
