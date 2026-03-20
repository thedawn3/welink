# 数据目录契约与排障

本页负责三件事：

1. 说明 WeLink 认可的标准目录结构
2. 说明 `msg` / `wechat-decrypt` / 图片 key 的作用
3. 给出统一排障顺序：`config-check -> runtime -> logs`

## 1. 标准目录契约

WeLink 只消费这种标准结构：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

规则：

- `contact/contact.db`：必需
- `message/message_*.db`：必需
- `sns/sns.db`：可选，但会进入状态展示与朋友圈能力检测
- `xwechat_files` 根目录不是标准目录

已在当前仓库链路实测：`ylytdeng/wechat-decrypt` 可产出 `sns/sns.db`。

## 2. 目录角色解释

| 目录 | 作用 | 是否必需 |
|---|---|---|
| `analysis directory` | WeLink 当前分析使用的稳定目录 | 是 |
| `source standard directory` | 仅 `manual-sync` 使用；供你后续同步 | `manual-sync` 必需 |
| `work directory` | 内置 stage 的临时可写目录 | 建议保留 |
| `msg` 目录 | 媒体索引与聊天图片显示 | 否 |
| `wechat-decrypt` 目录 | 自动读取图片 key / 配置 | 否 |

## 3. 关键环境变量

```env
WELINK_MODE=analysis-only
WELINK_ANALYSIS_DATA_DIR=/absolute/path/to/analysis_standard_dir
WELINK_DATA_DIR=/absolute/path/to/analysis_standard_dir
WELINK_SOURCE_DATA_DIR=
WELINK_WORK_DIR=./.tmp/welink-workdir
WELINK_MSG_DIR=
WELINK_WECHAT_DECRYPT_DIR=
WELINK_IMAGE_AES_KEY=
WELINK_IMAGE_AES_KEY_FILE=
WELINK_WECHAT_DECRYPT_CONFIG=
```

说明：

- `WELINK_ANALYSIS_DATA_DIR`：核心目录
- `WELINK_DATA_DIR`：兼容旧逻辑，建议与 `WELINK_ANALYSIS_DATA_DIR` 相同
- `WELINK_SOURCE_DATA_DIR`：`analysis-only` 留空；`manual-sync` 指向标准目录
- `WELINK_MSG_DIR`：不影响文本分析，只影响媒体体验
- `WELINK_WECHAT_DECRYPT_DIR`：推荐配置，便于自动读取 `image_keys.json` / `config.json`

## 4. 图片与 `wechat-decrypt`

如果你希望聊天记录里的图片可预览、可点击查看：

1. 配置 `WELINK_MSG_DIR`
2. 推荐再配置 `WELINK_WECHAT_DECRYPT_DIR`

WeLink 读取图片密钥的优先级：

1. `image_keys.json`
2. `config.json.image_aes_key`
3. `WELINK_IMAGE_AES_KEY`

推荐做法：

```env
WELINK_MSG_DIR=/absolute/path/to/msg
WELINK_WECHAT_DECRYPT_DIR=/absolute/path/to/wechat-decrypt
```

如果工具目录还没有 key，可在宿主机执行：

```bash
./scripts/extract-image-key.sh --restart
```

### 没有图片 key 会怎样

- 文本分析继续可用
- 旧图片可能仍能显示
- 一部分 2025-08+ V2 图片会退化成 `[图片]`

## 5. 常见错误与修复

### `analysis-only` + `source` 留空

这不是错误，是正常状态。

### `source` 指向原始 `xwechat_files` 根目录

症状：

- `缺少 contact/contact.db`
- `缺少 message/message_*.db`
- `no contact/message databases found under /app/source-data`

修复：

- 先在容器外整理成标准目录
- 再作为 `analysis` 或 `source` 传给 WeLink

### `source` 与 `analysis` 同目录

症状：

- `config-check` 返回阻塞错误
- 后端拒绝启动同步

修复：

- `manual-sync` 必须拆成两个目录
- 如果没有独立 source，就用 `analysis-only`

### 没有 `msg` 目录

不会影响联系人、关系分析、搜索、文本时间线，只影响媒体展示。

### 有 `msg` 目录但图片打不开

按这个顺序查：

1. `msg` 路径是否正确
2. 工具目录是否正确
3. 工具目录里是否已经生成 `image_keys.json`
4. 系统页“图片预览 / 密钥诊断”是否显示 ready

## 6. 推荐排障顺序

无论 macOS 还是 Windows，都按这 4 步查：

```bash
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/system/logs
curl -X POST http://localhost:8080/api/system/reindex
```

重点看：

- `mode`
- `primary_issue`
- `blocking_reasons`
- `last_error`
- `last_message_at`
- `last_sns_at`
- `is_initialized`

## 7. 平台路径示例

### macOS

```text
/Users/you/work/wechat-decrypt/decrypted_with_wal
/Users/you/Library/Containers/com.tencent.xinWeChat/Data/Documents/xwechat_files/<wxid>/msg
```

### Windows

```text
C:/Users/you/work/wechat-decrypt/decrypted_with_wal
C:/Users/you/Documents/WeChat Files/<wxid>/msg
```

Windows 建议统一写成 `C:/...`。
