# Windows 使用指南

## 1. 先把手机聊天记录迁移到电脑微信

手机微信 -> 我 -> 设置 -> 通用 -> 聊天记录迁移与备份 -> 迁移到电脑。

## 2. 准备解密产物

默认仍建议使用 [ylytdeng/wechat-decrypt](https://github.com/ylytdeng/wechat-decrypt)。

请按该项目的 Windows 说明完成解密，最终准备出：

```text
decrypted/
├── contact/contact.db
└── message/message_*.db
```

## 3. 生成 .env 并校验目录

PowerShell 中执行：

```powershell
cd welink
.\scripts\welink-doctor.ps1 -WriteEnv
```

如需手动指定路径，建议使用正斜杠：

```powershell
.\scripts\welink-doctor.ps1 `
  -DataDir 'C:/Users/you/work/wechat-decrypt/decrypted_with_wal' `
  -MsgDir 'C:/Users/you/Documents/WeChat Files/wxid_xxx/msg' `
  -WriteEnv
```

## 4. 启动

```powershell
docker compose up --build
```

或者直接：

```powershell
.\scripts\start-welink.ps1
```

## 5. 校验

```powershell
curl http://localhost:8080/api/health
curl http://localhost:8080/api/status
```

若你改了 `.env` 里的端口，请把上面的 `8080` 替换成实际 `WELINK_BACKEND_PORT`。

若你后面要接 MCP / AI，先确认 `/api/status` 里的 `is_initialized` 已经变成 `true`；索引未完成前，AI 看到的数据也不完整。

## Windows 注意事项

- `.env` 里的路径建议统一写成正斜杠，例如 `C:/Users/...`。
- 若 Docker Desktop 报挂载失败，先确认对应盘符已授权给 Docker。
- 若媒体目录缺失，可先留空 `WELINK_MSG_DIR`，不影响文本分析。
