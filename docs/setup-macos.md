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
└── message/message_*.db
```

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
curl http://localhost:8080/api/status
```

若你后面要接 MCP / AI，先确认 `/api/status` 里的 `is_initialized` 已经变成 `true`；索引未完成前，AI 看到的数据也不完整。

## 常见情况

- doctor 找不到 `msg` 目录：不影响核心聊天分析，只影响媒体回溯。
- 数据库体积远小于手机导出体积：手机导出包含媒体文件，数据库只包含结构化消息与引用。
- 已迁移但消息仍不全：先确认电脑微信里能看到完整历史，再重新解密。
