# macOS 使用指南

## 1. 先把手机聊天记录迁移到电脑微信

手机微信 -> 我 -> 设置 -> 通用 -> 聊天记录迁移与备份 -> 迁移到电脑。

这一步决定了本机数据库是否完整。若本机记录不全，WeLink 的分析也只会基于不完整数据。

## 2. 准备解密产物

默认方案使用 [ylytdeng/wechat-decrypt](https://github.com/ylytdeng/wechat-decrypt)。

```bash
git clone https://github.com/ylytdeng/wechat-decrypt
cd wechat-decrypt
sudo python3 main.py
```

期望产物目录：

```text
decrypted/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

### Docker 模式补充（重要）

Docker 下建议使用手动同步标准目录模式，容器内不负责抓取原始 `xwechat_files` 或执行平台抓取脚本。

- 容器外工具先准备标准目录（`contact/message`，可选 `sns`）
- 再按 [deploy-docker.md](./deploy-docker.md) 的映射方式启动 WeLink
- 启动后先看 `/api/system/config-check` 再执行同步/重建

## 3. 生成 .env 并校验目录

```bash
cd welink
./scripts/welink-doctor.sh --write-env
```

若自动发现失败，可显式指定：

```bash
./scripts/welink-doctor.sh \
  --data-dir /absolute/path/to/decrypted_with_wal \
  --msg-dir /Users/you/Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/wxid_xxx/msg \
  --write-env
```

### 可选：启用 WeLink 自动解密 / 自动刷新链路

如果你不想只走“先手工解密，再让 WeLink 读取结果”的旧路径，可以直接让 WeLink 在启动后自动编排解密与刷新：

```env
WELINK_INGEST_ENABLED=true
WELINK_SOURCE_DATA_DIR=/absolute/path/to/source_or_decrypted_with_wal
WELINK_ANALYSIS_DATA_DIR=/absolute/path/to/analysis
WELINK_DECRYPT_ENABLED=true
WELINK_DECRYPT_AUTO_START=true
WELINK_SYNC_ENABLED=true
WELINK_SYNC_WATCH_WAL=true
```

说明：

- `WELINK_SOURCE_DATA_DIR`：给解密器 / 内置 stage 读取的源目录
- `WELINK_ANALYSIS_DATA_DIR`：给 WeLink 分析服务使用的稳定目录
- `WELINK_SYNC_WATCH_WAL=true`：允许吸收最新 `-wal/-shm` 增量
- 如果你没有配置 `WELINK_MAC_DECRYPT_COMMAND`，且 `WELINK_DECRYPT_PROVIDER=builtin`，WeLink 会退回内置 stage 模式

## 4. 启动

```bash
docker compose up --build
```

或者直接：

```bash
./scripts/start-welink.sh
```

- 默认前端：`http://localhost:3000`
- 默认后端：`http://localhost:8080`
- 若你改了 `.env` 中的端口，以 `.env` 为准

## 5. 校验

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/status
```

优先观察 `/api/system/runtime`：

- `is_initialized=true` 表示当前索引已可供前端 / MCP 使用
- `decrypt_state` 可判断当前是否正在解密、已就绪或失败
- `data_revision` 单调递增，代表最新一次数据刷新版本

`/api/status` 仍保留为兼容接口；若你后面要接 MCP / AI，仍建议以 `/api/system/runtime` 为准。索引未完成前，AI 看到的数据也不完整。

如需观察实时刷新，可额外执行：

```bash
curl -N http://localhost:8080/api/events
```

## 可选：启用自动解密 + 自动刷新链路

当你希望 WeLink 启动时自动拉起 macOS 解密命令并在 `analysis` 目录保持最新数据，可以设置额外的 `.env` 变量，例如：

```env
WELINK_INGEST_ENABLED=true
WELINK_SOURCE_DATA_DIR=/Users/you/wechat/source
WELINK_ANALYSIS_DATA_DIR=/Users/you/wechat/analysis
WELINK_SYNC_ENABLED=true
WELINK_SYNC_WATCH_WAL=true
WELINK_DECRYPT_ENABLED=true
WELINK_DECRYPT_AUTO_START=true
WELINK_MAC_DECRYPT_COMMAND="/usr/bin/python3 decrypt.py --input ${source_data_dir} --output ${analysis_data_dir}"
```

这个配置会让 `ingest` 先把源目录同步到分析目录，`sync` 监听 `*.db` 与 `-wal/-shm` 并将变化合并成 revision，最终触发 `analysis` 的重建流程。

如果你启用了 `auto_refresh=true`，但同步监听未配置或启动失败，解密任务仍可能成功启动；这时应优先检查 `GET /api/system/logs` 中是否有 `warn/sync`，而不是只盯着 `last_error`。

确认是否生效：

- `/api/system/runtime` 中 `decrypt_state` 应先变为 `running`，最终停在 `ready`
- `/api/system/changes` 里 `pending_changes` 归零，`data_revision` 单调递增
- `/api/events` 显示 `runtime.revision.detected`、`runtime.decrypt.finished` 等事件

## 常见情况

- doctor 找不到 `msg` 目录：不影响核心聊天分析，只影响媒体回溯。
- 数据库体积远小于手机导出体积：手机导出包含媒体文件，数据库只包含结构化消息与引用。
- 已迁移但消息仍不全：先确认电脑微信里能看到完整历史，再重新解密。
