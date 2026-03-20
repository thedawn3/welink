# API 接口文档

> 后端基于 Go + Gin，默认监听 `:8080`，所有接口前缀 `/api`。
> 在线文档：访问 `/swagger/` 可查看 Swagger UI。


## 目录

- [初始化与状态](#初始化与状态)
- [统一运行时与实时事件](#统一运行时与实时事件)
- [ChatLab 导出](#chatlab-导出)
- [联系人分析](#联系人分析)
- [关系分析](#关系分析)
- [群聊分析](#群聊分析)
- [数据库浏览器](#数据库浏览器)
- [其他](#其他)


## 初始化与状态

### `POST /api/init`

触发后端重新建立索引，必须在使用其他分析接口前调用。索引在后台异步进行。当前语义更接近“手动强制重建”，自动刷新链路检测到 revision 后也会复用同一套分析重建流程。

**请求体**

```json
{
  "from": 1672531200,
  "to":   1704067200
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `from` | int64 | 开始时间（Unix 秒），`0` 表示不限 |
| `to`   | int64 | 结束时间（Unix 秒），`0` 表示不限 |

**响应**

```json
{ "status": "indexing" }
```


### `GET /api/status`

查询兼容态索引进度，旧前端/脚本可继续轮询判断是否可以开始使用；新前端与 MCP 更推荐读取 `/api/system/runtime`。

**响应**

```json
{
  "is_indexing":    false,
  "is_initialized": true,
  "total_cached":   312
}
```

| 字段 | 说明 |
|------|------|
| `is_indexing` | 是否正在索引 |
| `is_initialized` | 是否已完成初始化（可正常使用） |
| `total_cached` | 当前缓存的联系人数量 |


### `GET /api/health`

健康检查，返回数据库连接数。

**响应**

```json
{ "status": "ok", "db_connected": 5 }
```

## 统一运行时与实时事件

### `GET /api/system/config-check`

统一配置/目录校验接口。前端系统页与 Docker 排障应先看这个接口，再决定是否可启动同步/解密。

**典型响应**

```json
{
  "deployment_target": "docker",
  "mode": "analysis-only",
  "analysis_dir": {
    "path": "/app/analysis-data",
    "exists": true,
    "has_contact_db": true,
    "has_message_db": true
  },
  "source_dir": {
    "path": "/app/source-data",
    "exists": false,
    "is_standard_layout": false
  },
  "sns": {
    "detected": true,
    "ready": true,
    "db_path": "/app/analysis-data/sns/sns.db"
  },
  "suggested_actions": [
    "prepare_standard_directory",
    "rebuild_index"
  ]
}
```

关键字段：

- `deployment_target`：`docker` 或 `host`
- `mode`：当前运行模式（例如 `analysis-only`、`manual-stage`、`external-command`）
- `analysis_dir/source_dir/work_dir`：目录可用性与契约命中情况
- `sns`：是否检测到 `sns.db`
- `suggested_actions`：UI/MCP 可执行建议

### `GET /api/system/runtime`

统一运行时状态接口，前端系统页与 MCP 都优先读取这里。

**典型响应**

```json
{
  "engine_type": "windows",
  "deployment_target": "docker",
  "decrypt_state": "ready",
  "is_indexing": false,
  "is_initialized": true,
  "total_cached": 312,
  "data_revision": 5,
  "pending_changes": 0,
  "last_decrypt_at": "2026-03-20T03:20:00Z",
  "last_reindex_at": "2026-03-20T03:20:04Z",
  "last_message_at": "2026-03-20T03:19:58Z",
  "last_sns_at": "2026-03-19T21:10:00Z",
  "last_error": ""
}
```

关键字段：

- `engine_type`：当前运行引擎 / 平台
- `decrypt_state`：`idle/running/stopping/ready/error`
- `data_revision`：每次成功吸收新数据并完成重建后递增
- `pending_changes`：已检测到但尚未完全消化的变更数
- `last_message_at`：最近一条消息时间（若可检测）
- `last_sns_at`：最近一条朋友圈时间（检测到 `sns.db` 时）
- `last_error`：最近一次运行时错误

### `GET /api/system/tasks`

返回最近的运行时任务队列，例如解密、内置 stage、reindex。

**Query 参数**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `limit` | `100` | 返回最近任务数量 |

**响应**

```json
{
  "items": [
    {
      "id": "task-1",
      "type": "decrypt",
      "status": "running",
      "message": "builtin stage started",
      "started_at": "2026-03-20T03:20:00Z"
    }
  ]
}
```

### `GET /api/system/logs`

返回运行时日志；可结合任务状态排查平台命令失败、目录错误、自动刷新卡住等问题。

**Query 参数**

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `limit` | `200` | 返回日志数量 |
| `task_id` | 空 | 传入时只返回该任务日志 |

**行为说明**

- 不传 `task_id` 时，返回统一运行时日志（`decrypt` / `sync` / `analysis` 等）的最近记录。
- 传入 `task_id` 时，只返回该解密任务的 orchestrator 日志，不混入统一运行时日志。
- `task_id` 日志按产生顺序返回（最早 -> 最晚），不是倒序。
- `task_id` 日志上的 `limit` 表示“返回最早的前 N 条匹配日志”，不是“最近 N 条”。
- 若 `task_id` 不存在，返回 `404` 与 `{ "error": "task not found" }`。

**响应**

```json
{
  "items": [
    {
      "id": 12,
      "time": "2026-03-20T03:20:01Z",
      "level": "info",
      "source": "sync",
      "message": "detected change revision rev-1",
      "fields": {
        "changed_databases": "message_0.db,message_0.db-wal"
      }
    }
  ]
}
```

补充说明：

- 统一运行时日志会携带结构化 `fields`，前端可直接做字段筛选或展示。
- `task_id` 模式下的 orchestrator 日志还会带上 `task_id`、`stream` 等字段。

### `GET /api/system/changes`

返回最近 revision 与同步层摘要，用于判断自动刷新是否已检测到数据库变化。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `since_revision` | 否 | 仅当当前 `data_revision` 大于该值时返回最近日志；同时返回 `has_newer_revision` |

**响应**

```json
{
  "data_revision": 5,
  "pending_changes": 0,
  "last_reindex_at": "2026-03-20T03:20:04Z",
  "last_change_reason": "message_0.db,message_0.db-wal",
  "last_error": "",
  "items": [
    {
      "time": "2026-03-20T03:20:01Z",
      "level": "info",
      "source": "sync",
      "message": "detected change revision rev-1"
    }
  ],
  "sync": {
    "running": true,
    "watch_wal": true,
    "last_revision_seq": 12
  }
}
```

其中：

- `items` 是最近的运行时日志摘要
- `sync` 是同步层状态快照，便于判断自动刷新是否还在推进
- `last_revision_seq` 更适合排查 watcher 是否已持续感知到新变化

### `POST /api/system/decrypt/start`

手动启动当前平台解密任务。可带平台、目录、命令覆盖参数。

**请求体（可选字段）**

```json
{
  "platform": "windows",
  "source_data_dir": "C:/wechat/source",
  "analysis_data_dir": "C:/wechat/analysis",
  "work_dir": "C:/wechat/workdir",
  "command": "python decrypt.py --input ${source_data_dir} --output ${analysis_data_dir}",
  "auto_refresh": true,
  "wal_enabled": true
}
```

**字段补充说明**

- 启动前会复用 `config-check` 校验。
- 若 `source` 不存在、不是标准目录、`source/analysis` 同目录、`work_dir` 不可写等，返回 `400` + 可执行错误描述。
- Docker 推荐模式是手动同步标准目录；是否允许启动由当前模式与校验结果共同决定。

**响应**

```json
{
  "task_id": "task-1",
  "status": "started"
}
```

### `POST /api/system/decrypt/stop`

手动停止当前运行中的解密任务。

**请求体（可选）**

```json
{
  "task_id": "task-1"
}
```

若不传 `task_id`，后端会尝试停止当前正在运行的解密任务。

仅 `running` / `stopping` 状态的任务可被停止；若任务已完成或根本不存在，会返回 `400`。

**响应**

```json
{
  "task_id": "task-1",
  "status": "stopping"
}
```

### `POST /api/system/reindex`

强制重建索引；当自动刷新已吸收新数据但分析结果未更新时可手动调用。

**请求体**

```json
{
  "from": 0,
  "to": 0
}
```

`from` / `to` 继续沿用旧初始化接口的 Unix 秒过滤语义；传 `0` 表示不限。

**响应**

```json
{ "status": "indexing" }
```

### `GET /api/events`

SSE 事件流。前端会优先订阅它，在断开时回退轮询。常见事件包括：

- `runtime.revision.detected`
- `runtime.reindex.started`
- `runtime.reindex.finished`
- `runtime.decrypt.started`
- `runtime.decrypt.finished`
- `runtime.decrypt.failed`

**SSE 数据格式**

```json
{
  "id": 42,
  "type": "runtime.revision.detected",
  "at": "2026-03-20T03:20:01Z",
  "revision": "rev-1",
  "message": "reindex started",
  "payload": {
    "revision": "rev-1"
  }
}
```


## 联系人分析

### `GET /api/contacts/stats`

获取所有联系人的统计信息（从内存缓存返回，极速）。

> 在 `/api/init` 完成前返回空数组。

**响应** `[]ContactStatsExtended`

```json
[
  {
    "username":           "wxid_abc123",
    "nickname":           "张三",
    "remark":             "张三（同事）",
    "alias":              "",
    "flag":               3,
    "description":        "",
    "big_head_url":       "https://...",
    "small_head_url":     "https://...",
    "total_messages":     1234,
    "their_messages":     700,
    "my_messages":        534,
    "first_message_time": "2020-03-01",
    "last_message_time":  "2024-11-15",
    "first_msg":          "你好呀",
    "emoji_count":        42,
    "type_pct": {
      "文本": 80.5,
      "图片": 12.3,
      "语音": 5.0,
      "表情": 1.5,
      "视频": 0.5,
      "其他": 0.2
    },
    "type_cnt": {
      "文本": 993,
      "图片": 152
    }
  }
]
```


### `GET /api/global`

获取全局聚合统计（所有联系人汇总）。

**响应** `GlobalStats`

```json
{
  "total_friends":      312,
  "zero_msg_friends":   28,
  "total_messages":     186432,
  "busiest_day":        "2023-02-05",
  "busiest_day_count":  412,
  "emoji_king":         "张三（同事）",
  "monthly_trend": {
    "2023-01": 1234,
    "2023-02": 2100
  },
  "hourly_heatmap":     [12, 8, 3, 1, 0, 2, 15, 45, ...],
  "type_mix": {
    "文本": 150000,
    "图片": 20000
  },
  "late_night_ranking": [
    {
      "name":             "李四",
      "late_night_count": 342,
      "total_messages":   2100,
      "ratio":            16.3
    }
  ]
}
```


### `GET /api/contacts/detail`

获取单个联系人的深度分析（按需计算，非缓存）。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `username` | 是 | 联系人 wxid |

**响应** `ContactDetail`

```json
{
  "hourly_dist":     [5, 2, 1, 0, 0, 0, 8, 30, 55, ...],
  "weekly_dist":     [10, 85, 92, 78, 88, 60, 20],
  "daily_heatmap":   { "2023-01-15": 32, "2023-01-16": 18 },
  "late_night_count": 342,
  "money_count":      5,
  "initiation_count": 180,
  "total_sessions":   200
}
```

| 字段 | 说明 |
|------|------|
| `hourly_dist` | 长度 24，按小时统计消息数（索引 = 小时） |
| `weekly_dist` | 长度 7，`[0]` = 周日，`[1]` = 周一 … `[6]` = 周六 |
| `daily_heatmap` | `"YYYY-MM-DD"` → 当日消息数 |
| `late_night_count` | 0~5 点消息数 |
| `money_count` | 红包/转账次数（双方合计） |
| `initiation_count` | 你主动发起的对话段数 |
| `total_sessions` | 总对话段数（间隔 > 6h 视为新段） |


### `GET /api/contacts/wordcloud`

获取联系人的词云数据（中文分词 + 停用词过滤，返回 top 120）。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `username`    | 是 | 联系人 wxid |
| `include_mine` | 否 | 值为 `"true"` 时包含双方消息，默认仅对方 |

**响应** `[]WordCount`

```json
[
  { "word": "哈哈", "count": 312 },
  { "word": "知道", "count": 280 }
]
```


### `GET /api/contacts/sentiment`

按月份进行情感分析（基于关键词词典评分）。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `username`    | 是 | 联系人 wxid |
| `include_mine` | 否 | 值为 `"true"` 时包含双方消息，默认仅对方 |

**响应** `SentimentResult`

```json
{
  "monthly": [
    { "month": "2023-01", "score": 0.72, "count": 134 },
    { "month": "2023-02", "score": 0.58, "count": 98 }
  ],
  "overall":  0.68,
  "positive": 420,
  "negative": 85,
  "neutral":  210
}
```

| 字段 | 说明 |
|------|------|
| `score` | 0.0~1.0，0.5 为中性基线 |
| `overall` | 全部参与评分消息的平均分 |
| `positive` | score ≥ 0.6 的消息数 |
| `negative` | score ≤ 0.4 的消息数 |
| `neutral`  | 0.4 < score < 0.6 的消息数 |


### `GET /api/contacts/common-groups`

获取当前用户与指定联系人的共同群聊列表（在群消息表中查找对方是否有发言记录）。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `username` | 是 | 联系人 wxid |

**响应** `[]GroupInfo`（结构同 `/api/groups`）

```json
[
  {
    "username":           "12345678@chatroom",
    "name":               "工作群",
    "small_head_url":     "https://...",
    "total_messages":     8423,
    "first_message_time": "2021-06-01",
    "last_message_time":  "2024-11-15"
  }
]
```

### `GET /api/search/messages`

全量关键词搜索，返回跨联系人/群聊的命中消息。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `q` | 是 | 搜索关键词 |
| `include_mine` | 否 | 值为 `"true"` 时包含我发送的消息 |
| `limit` | 否 | 最大返回条数，默认 200 |

## 关系分析

### `GET /api/relations/overview`

首页客观模式榜单，包含：

- `warming`
- `cooling`
- `initiative`
- `fast_reply`

每个 item 都包含以下核心字段：

| 字段 | 说明 |
|------|------|
| `score` | 关系信号强度 |
| `confidence` | 当前结论可信度 |
| `stale_hint` | 久未联系时的降置信提示 |
| `confidence_reason` | 低样本 / 久未联系等解释文案 |

### `GET /api/relations/detail?username=...`

联系人关系档案详情，返回：

- `objective_summary`
- `playful_summary`
- `metrics`
- `stage_timeline`
- `evidence_groups`
- `confidence`
- `stale_hint`
- `confidence_reason`

### `GET /api/controversy/overview`

首页争议模式榜单，包含：

- `simp`
- `ambiguity`
- `faded`
- `tool_person`
- `cold_violence`

每个 item 同样返回 `score`、`confidence`、`stale_hint`、`confidence_reason`，用于前端区分“当前高置信判断”和“历史回看”。

### `GET /api/controversy/detail?username=...`

联系人争议模式详情，返回争议标签列表。每个 label 除 `score` / `confidence` 外，还会返回：

- `why`
- `metrics`
- `evidence_groups`
- `stale_hint`
- `confidence_reason`


### `GET /api/contacts/messages`

获取某一天与指定联系人的完整聊天记录（用于日历点击查看）。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `username` | 是 | 联系人 wxid |
| `date`     | 是 | 日期，格式 `"YYYY-MM-DD"`（北京时间） |

**响应** `[]ChatMessage`

```json
[
  { "time": "09:12", "content": "早上好", "is_mine": false, "type": 1 },
  { "time": "09:14", "content": "早！",   "is_mine": true,  "type": 1 },
  { "time": "10:30", "content": "[图片]", "is_mine": false, "type": 3 }
]
```

| 字段 | 说明 |
|------|------|
| `time` | HH:MM，北京时间 |
| `content` | 文本内容，非文本类型为描述字符串（见下表） |
| `is_mine` | `true` = 你发的 |
| `type` | 消息类型（见消息类型说明） |

**非文本消息内容映射**

| type | content 值 |
|------|-----------|
| 3  | `[图片]` |
| 34 | `[语音]` |
| 43 | `[视频]` |
| 47 | `[表情]` |
| 49（含 wcpay/redenvelope） | `[红包/转账]` |
| 49（其他） | `[链接/文件]` |
| 其他 | `[消息类型 N]` |


## 群聊分析

### `GET /api/groups`

获取所有有消息记录的群聊列表。

**响应** `[]GroupInfo`

```json
[
  {
    "username":           "12345678@chatroom",
    "name":               "工作群",
    "small_head_url":     "https://...",
    "total_messages":     8423,
    "first_message_time": "2021-06-01",
    "last_message_time":  "2024-11-15"
  }
]
```


### `GET /api/groups/detail`

获取群聊深度分析（懒加载，首次计算后缓存内存）。

**Query 参数**

| 参数 | 必填 | 说明 |
|------|------|------|
| `username` | 是 | 群 @chatroom wxid |

**响应** `GroupDetail`

```json
{
  "hourly_dist":  [3, 1, 0, 0, 0, 1, 8, 45, ...],
  "weekly_dist":  [5, 120, 135, 110, 125, 80, 25],
  "daily_heatmap": { "2023-01-15": 48 },
  "member_rank": [
    { "speaker": "张三", "count": 1240 },
    { "speaker": "李四", "count": 980 }
  ],
  "top_words": [
    { "word": "明天", "count": 145 },
    { "word": "会议", "count": 120 }
  ]
}
```


## 数据库浏览器

### `GET /api/databases`

列出所有已加载的数据库文件。

**响应** `[]DBInfo`

```json
[
  { "name": "contact.db",   "path": "/data/contact/contact.db",   "size": 2097152, "type": "contact" },
  { "name": "message_1.db", "path": "/data/message/message_1.db", "size": 52428800, "type": "message" }
]
```


### `GET /api/databases/:dbName/tables`

列出指定数据库的所有表及行数。

**响应** `[]TableInfo`

```json
[
  { "name": "contact", "row_count": 512 },
  { "name": "Msg_96e07f9a...", "row_count": 3420 }
]
```


### `GET /api/databases/:dbName/tables/:tableName/schema`

查看表结构。

**响应** `[]ColumnInfo`

```json
[
  { "cid": 0, "name": "local_id", "type": "INTEGER", "not_null": false, "default_value": "", "primary_key": true },
  { "cid": 5, "name": "create_time", "type": "INTEGER", "not_null": false, "default_value": "", "primary_key": false }
]
```


### `GET /api/databases/:dbName/tables/:tableName/data`

分页获取表数据。

**Query 参数**

| 参数 | 默认值 | 最大值 | 说明 |
|------|--------|--------|------|
| `offset` | 0 | - | 起始行偏移 |
| `limit`  | 50 | 200 | 每页行数 |

**响应** `TableData`

```json
{
  "columns": ["local_id", "create_time", "message_content"],
  "rows": [[1, 1627920012, "你好"], [2, 1627920100, null]],
  "total": 3420
}
```


## 其他

### `GET /api/stats/filter`

用自定义时间范围计算统计（不更新缓存，即时返回）。

**Query 参数**

| 参数 | 说明 |
|------|------|
| `from` | 开始时间（Unix 秒），可省略 |
| `to`   | 结束时间（Unix 秒），可省略 |

**响应** `FilteredStats`

```json
{
  "contacts":    [ ...同 /api/contacts/stats... ],
  "global_stats": { ...同 /api/global... }
}
```


## 错误响应格式

```json
{ "error": "username required" }
```

HTTP 状态码：`400` Bad Request / `500` Internal Server Error。


## ChatLab 导出

### `GET /api/export/chatlab/contact`

参数：`username`、`limit`、`download`

### `GET /api/export/chatlab/group`

参数：`username`、`date`、`download`

### `GET /api/export/chatlab/search`

参数：`q`、`include_mine`、`limit`、`download`

### `POST /api/export/chatlab`

统一导出入口，`scope` 支持：`contact`、`group`、`search`。

**请求体**

```json
{
  "scope": "search",
  "query": "晚安",
  "include_mine": "true",
  "limit": 200
}
```

所有导出接口都返回：

```json
{
  "file_name": "xxx.chatlab.json",
  "mime_type": "application/json",
  "data": { "chatlab": { "version": "0.0.1" } },
  "summary": {
    "scope": "search",
    "query": "晚安",
    "include_mine": true,
    "limit": 200,
    "conversation_name": "搜索结果",
    "message_count": 42,
    "member_count": 8
  }
}
```

`summary` 用于前端/MCP 快速确认导出上下文与规模，常见字段包括：

- `scope`
- `username` / `query`
- `date`
- `include_mine`
- `limit`
- `conversation_name`
- `message_count`
- `member_count`
