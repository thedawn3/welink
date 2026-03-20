# Windows 使用指南

本页只负责 Windows 平台差异。Docker 模式、`.env`、目录契约、红色阻塞错误，以 [`deploy-docker.md`](./deploy-docker.md) 和 [`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md) 为准。

## 1. Windows 一键部署前提

在 Windows 机器上让 AI 拉仓并启动前，先确认：

- 已安装 Git
- 已安装 Docker Desktop，且 `docker compose version` 可用
- 已安装 Python，且 `py -3` 或 `python` 至少有一个在 `PATH`
- 当前 PowerShell 会话已执行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
```

说明：

- 这条执行策略命令按当前 PowerShell 会话生效；换新终端后可以重新执行一次
- `scripts/welink-doctor.ps1` 会调用 Python 生成 `.env`，所以没有 Python 时脚本不会继续
- Windows 路径统一建议写成正斜杠，例如 `C:/Users/you/...`

## 2. 先判断你属于哪种正式模式

### `analysis-only`

适合你已经有一个可直接分析的标准目录：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

这种模式下：

- 只给 WeLink 一个 `analysis directory`
- 不提供 `source standard directory`
- Docker 不做容器内自动解密 / watcher

### `manual-sync`

适合你除了 analysis 目录，还想额外提供另一个 source 标准目录，供 WeLink 在系统页执行手动同步。

注意：

- `source` 和 `analysis` 都必须是标准目录
- `source` 不能直接指向 `xwechat_files` 根目录
- `source` 与 `analysis` 不能是同一路径

## 3. Windows AI 一键启动命令

### `analysis-only`

```powershell
git clone https://github.com/thedawn3/welink.git
cd welink
Set-ExecutionPolicy -Scope Process Bypass -Force
.\scripts\start-welink.ps1 `
  -Mode analysis-only `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir' `
  -MsgDir 'C:/absolute/path/to/msg' `
  -WechatDecryptDir 'C:/absolute/path/to/wechat-decrypt'
```

### `manual-sync`

```powershell
git clone https://github.com/thedawn3/welink.git
cd welink
Set-ExecutionPolicy -Scope Process Bypass -Force
.\scripts\start-welink.ps1 `
  -Mode manual-sync `
  -DataDir 'C:/absolute/path/to/analysis_standard_dir' `
  -SourceDataDir 'C:/absolute/path/to/source_standard_dir' `
  -MsgDir 'C:/absolute/path/to/msg' `
  -WechatDecryptDir 'C:/absolute/path/to/wechat-decrypt'
```

可选参数说明：

- `-MsgDir`：可省略；省略后不影响文本分析，但聊天图片缩略图 / 点击查看不可用
- `-WechatDecryptDir`：可省略；省略后不影响基础部署，但 V2 图片不会自动读取 `image_keys.json`

## 4. WeLink 需要的目录，不是微信原始目录

最容易误导 AI 的地方只有一个：

- `-SourceDataDir` 或 `-DataDir` 需要的是标准目录根
- 不是 `xwechat_files` 根目录
- 不是 `WeChat Files` 整个用户目录

传错时，典型结果是：

- doctor 直接报缺少 `contact/contact.db` 或 `message/message_*.db`
- 后端 `config-check` 提示 source 不是标准目录
- 系统页拒绝启动同步

如果你只有微信原始目录，先用容器外工具整理成标准目录，再把标准目录传给 WeLink。

## 5. 启动后只看这几条验收命令

```powershell
docker compose ps
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/system/logs
```

判断标准：

- `docker compose ps` 中前后端容器都在运行
- `/api/system/config-check` 没有阻塞项，或明确显示当前是正常的 `analysis-only`
- `/api/system/runtime` 中 `is_initialized=true`
- `/api/system/logs` 没有持续报目录/权限/路径错误

## 6. Windows 平台常见坑

### PowerShell 拒绝执行脚本

先在当前终端执行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass -Force
```

### `docker compose` 不可用

WeLink 脚本要求的是 Docker Compose v2，也就是：

```powershell
docker compose version
```

如果这里只有旧版 `docker-compose`，先在 Docker Desktop 里启用 Compose v2，再继续。

### doctor 报找不到 Python

`scripts/welink-doctor.ps1` 依赖 Python 运行 `scripts/welink_doctor.py`。请先确保以下命令至少一个可用：

```powershell
py -3 --version
python --version
```

### 挂载失败或容器看不到目录

先检查：

- Docker Desktop 是否已授权对应盘符
- 路径是否写成 `C:/...` 这种正斜杠形式
- 目标目录是否真实存在

### source / analysis 写成了同一路径

这会被后端阻止，因为会污染分析目录。请分开两个目录，或退回 `analysis-only`。

## 7. 图片与 SNS

- `ylytdeng/wechat-decrypt` 已验证可产出 `sns/sns.db`
- `WELINK_MSG_DIR` 正确时，聊天流可回溯图片
- `WELINK_WECHAT_DECRYPT_DIR` 正确时，WeLink 会自动尝试读取 `image_keys.json` / `config.json`

如果图片仍只显示 `[图片]`，优先回到 [`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md) 查看图片 key 排障，不要先怀疑部署流程。

## 8. 下一步

Windows 上只要部署验收通过，接下来按这个顺序继续：

1. 打开前端：`http://localhost:3000`
2. 在系统页确认 `config-check` 与 `runtime`
3. 如需 AI 查询，再接 [`../mcp-server/README.md`](../mcp-server/README.md)
