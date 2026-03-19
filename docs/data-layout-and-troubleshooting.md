# 数据目录与故障排查

## 目录契约

WeLink 只认一套标准输入：

```text
<DATA_DIR>/
├── contact/contact.db
└── message/message_*.db
```

可选媒体目录：

```text
<MSG_DIR>/
├── image2/
├── video/
└── ...
```

## .env 变量

```env
WELINK_DATA_DIR=/absolute/path/to/decrypted
WELINK_MSG_DIR=/absolute/path/to/msg
WELINK_BACKEND_PORT=8080
WELINK_FRONTEND_PORT=3000
```

`WELINK_DATA_DIR` 必填；`WELINK_MSG_DIR` 可为空。

## 启动前校验

```bash
./scripts/welink-doctor.sh --write-env
```

或 PowerShell：

```powershell
.\scripts\welink-doctor.ps1 -WriteEnv
```

## 常见问题

### `contact/contact.db` 不存在

说明解密产物目录不对，或你传入的是上层目录/错误目录。

### `message/message_*.db` 不存在

说明消息库没有解密成功，或当前目录只有联系人库。

### 前端能打开但消息为空

先看：

```bash
curl http://localhost:8080/api/status
```

若你改了 `.env` 里的 `WELINK_BACKEND_PORT`，把这里的 `8080` 替换成实际端口。

若还没初始化完成，先等待索引结束。

### 电脑微信里看不到完整历史

这是导入问题，不是 WeLink 或解密问题。先在微信客户端确认历史是否已同步完整。

### 媒体目录缺失

不会影响联系人统计、关系分析、关键词搜索；只影响静态媒体回溯。
