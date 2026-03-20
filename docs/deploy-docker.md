# Docker 部署与一键启动

本页是 `thedawn3/welink` 的 Docker 主文档。

## 1. 先记结论

当前 Docker 只保留两种正式模式：

- `analysis-only`：只消费 `analysis directory`
- `manual-sync`：消费 `analysis directory + source standard directory`

共同规则：

- 容器外工具负责准备标准目录
- WeLink 容器内只做校验、手动同步、索引、分析与展示
- Docker 首期不把容器内 watcher / auto refresh 作为正式主路径
- `msg` 与 `wechat-decrypt` 是可选增强项，不是启动硬前置

## 2. 启动前提

在任一平台执行脚本前，请先确认：

- Git 已安装
- Docker Desktop 已安装，且 `docker compose version` 可用
- Python 3 已安装
- 已准备标准目录：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

目录语义：

- `analysis directory`：WeLink 当前分析所使用的稳定目录
- `source standard directory`：仅 `manual-sync` 需要；供你后续手动同步
- `work directory`：内置 stage 临时工作目录，可写
- `media directory`：微信 `msg` 目录，可选
- `wechat-decrypt directory`：工具目录，可选

## 3. 推荐：直接用启动脚本

### macOS / Linux

`manual-sync`：

```bash
./scripts/start-welink.sh \
  --mode manual-sync \
  --data-dir /absolute/path/to/analysis_standard_dir \
  --source-data-dir /absolute/path/to/source_standard_dir
```

`analysis-only`：

```bash
./scripts/start-welink.sh \
  --mode analysis-only \
  --data-dir /absolute/path/to/analysis_standard_dir
```

### Windows PowerShell

先执行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
```

然后：

```powershell
.\scripts\start-welink.ps1 `
  -Mode manual-sync `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir' `
  -SourceDataDir 'C:/absolute/path/to/source_standard_dir'
```

或：

```powershell
.\scripts\start-welink.ps1 `
  -Mode analysis-only `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir'
```

### 可选增强参数

如有需要，可在两端都追加：

- `msg` 目录
- `wechat-decrypt` 目录

示例：

```bash
--msg-dir /absolute/path/to/msg --wechat-decrypt-dir /absolute/path/to/wechat-decrypt
```

```powershell
-MsgDir 'C:/absolute/path/to/msg' -WechatDecryptDir 'C:/absolute/path/to/wechat-decrypt'
```

## 4. 两种正式模式怎么选

| 模式 | 何时使用 | 必填参数 | 典型用途 |
|---|---|---|---|
| `analysis-only` | 你已经有一个可分析的标准目录 | `data-dir` | 只浏览、搜索、分析现有数据 |
| `manual-sync` | 你既有当前分析目录，又有待同步的 source 标准目录 | `data-dir` + `source-data-dir` | 容器外更新 source，WeLink 内手动同步 |

明确禁止：

- 直接把 `xwechat_files` 根目录当 `source-data-dir`
- `manual-sync` 下把 `source` 和 `analysis` 指到同一路径

## 5. `.env` 模板

最推荐直接让脚本生成 `.env`。如果你必须手写，请参考仓库根的 [`.env.example`](../.env.example)。

最小 `analysis-only`：

```env
WELINK_MODE=analysis-only
WELINK_ANALYSIS_DATA_DIR=/absolute/path/to/analysis_standard_dir
WELINK_DATA_DIR=/absolute/path/to/analysis_standard_dir
WELINK_SOURCE_DATA_DIR=
WELINK_WORK_DIR=./.tmp/welink-workdir
WELINK_MSG_DIR=
WELINK_WECHAT_DECRYPT_DIR=
WELINK_INGEST_ENABLED=false
WELINK_DECRYPT_ENABLED=false
WELINK_DECRYPT_AUTO_START=false
WELINK_SYNC_ENABLED=false
```

切到 `manual-sync` 时，最少改：

```env
WELINK_MODE=manual-sync
WELINK_SOURCE_DATA_DIR=/absolute/path/to/source_standard_dir
```

## 6. AI 验收检查顺序

启动完成后，固定按这个顺序检查：

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/system/logs
```

建议理解为：

1. `health`：服务是否活着
2. `config-check`：目录和模式是否正确
3. `runtime`：索引是否就绪、最近错误是什么
4. `logs`：如果失败，具体失败点在哪里

## 7. 什么情况下会出现红色阻塞错误

| 场景 | 是否阻塞 | 说明 |
|---|---|---|
| `analysis-only` + `source` 留空 + `analysis` 就绪 | 否 | 正常可用，不是错误 |
| `manual-sync` + `source` 不是标准目录 | 是 | 必须包含 `contact/contact.db` 和 `message/message_*.db` |
| `manual-sync` + `source/analysis` 同目录 | 是 | 会污染分析目录 |
| `work_dir` 不可写 | 是 | 内置 stage 无法运行 |
| `msg` 缺失 | 否 | 只影响媒体体验，不影响文本分析 |
| `wechat-decrypt` 缺失 | 否 | 只影响 V2 图片自动取 key |

## 8. 最常见的 5 个坑

### 1) 把 `xwechat_files` 根目录传给 `source`

这不是标准目录，会报：

- `缺少 contact/contact.db`
- `缺少 message/message_*.db`
- `source_data_dir 不是标准目录`

### 2) Windows 没先执行 `Set-ExecutionPolicy`

PowerShell 会直接阻止脚本运行。

### 3) Windows 没装 Python 3

`welink-doctor.ps1` 依赖 `py -3` 或 `python`。

### 4) `source` / `analysis` 用了同一个目录

后端会阻止这种配置。

### 5) 图片一直显示 `[图片]`

优先检查：

- `msg` 目录是否正确
- `wechat-decrypt` 工具目录是否正确
- 工具目录里是否已经生成 `image_keys.json`

## 9. Docker 与图片 / `sns.db`

当前仓库链路已验证：

- `ylytdeng/wechat-decrypt` 可产出 `sns/sns.db`
- WeLink 可自动读取 `image_keys.json`
- 若图片只有缩略图 key，聊天记录也会优先回退展示缩略图，而不是完全失效

如果你只关心文本分析，可以完全不配置 `msg` / `wechat-decrypt`。

## 10. 平台差异文档

- macOS：[`setup-macos.md`](./setup-macos.md)
- Windows：[`setup-windows.md`](./setup-windows.md)
- 数据目录与排障：[`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md)
