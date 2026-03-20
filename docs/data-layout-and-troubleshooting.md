# 数据目录契约与排障

## 1. 标准目录契约

WeLink 消费的标准目录如下：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

说明：

- `contact/contact.db` 与 `message/message_*.db` 为必需
- `sns/sns.db` 为可选，但会参与目录校验与状态展示
- 目录可来自任意容器外工具，WeLink 只负责消费标准结构

## 2. 关键环境变量

```env
WELINK_ANALYSIS_DATA_DIR=/absolute/path/to/standard_dir
WELINK_DATA_DIR=/absolute/path/to/standard_dir
WELINK_SOURCE_DATA_DIR=
WELINK_WORK_DIR=./.tmp/welink-workdir
WELINK_MSG_DIR=/absolute/path/to/msg
```

Docker 首期推荐：

```env
WELINK_MODE=analysis-only
WELINK_INGEST_ENABLED=false
WELINK_DECRYPT_ENABLED=false
WELINK_DECRYPT_AUTO_START=false
WELINK_SYNC_ENABLED=false
```

含义：

- `WELINK_ANALYSIS_DATA_DIR`: 分析读取目录（核心）
- `WELINK_SOURCE_DATA_DIR`: ingest/decrypt 读取目录；Docker 手动同步模式建议留空
- `WELINK_WORK_DIR`: stage 临时目录（可写）
- `WELINK_MSG_DIR`: 媒体目录（可选）

## 3. 常见错误与修复

### `source` 指向了原始 `xwechat_files` 根目录

症状：启动同步/解密时报 `no contact/message databases found under /app/source-data`。

修复：

1. 容器外先整理为标准目录（`contact/message`，可选 `sns`）
2. 将标准目录配置到 `WELINK_ANALYSIS_DATA_DIR`
3. Docker 手动同步模式下保持 `WELINK_SOURCE_DATA_DIR=`（空）

### source 与 analysis 指向同一目录

症状：运行时校验失败、同步链路异常或目录污染。

修复：

- 分离 source / analysis
- 或直接采用 Docker 手动同步模式（source 为空，占位挂载）

### 媒体目录不存在

不会阻塞联系人/关系/关键词分析，只影响图片/视频等媒体回溯。

## 4. 推荐排障顺序

固定顺序：

1. `GET /api/system/config-check`
2. `GET /api/system/runtime`
3. `GET /api/system/logs`
4. 必要时 `POST /api/system/decrypt/start`
5. 必要时 `POST /api/system/reindex`

示例：

```bash
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/system/logs
curl -X POST http://localhost:8080/api/system/reindex
```

重点字段：

- `deployment_target`
- `mode`
- `last_error`
- `last_message_at`
- `last_sns_at`
- `pending_changes`

## 5. 自动刷新与 Docker 的边界

当前 Docker 正式路径是手动同步标准目录，不依赖容器内 watcher 自动刷新。

建议流程：

1. 容器外脚本更新标准目录（`contact/message/sns`）
2. WeLink 执行校验
3. 手动同步/重建索引并观察 runtime

如果你需要完整自动刷新 watcher，优先在本地原生模式使用。
