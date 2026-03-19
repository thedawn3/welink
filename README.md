<p align="center">
  <img src="logo.svg" width="80" height="80" alt="WeLink Logo" />
</p>

<h1 align="center">WeLink - 微信聊天记录导入与关系分析平台</h1>

WeLink 把本地微信数据整理成一条完整链路：

- 手机聊天记录迁移到电脑微信
- 解密准备（默认对接 `wechat-decrypt`）
- 数据目录校验与环境变量生成
- 本地分析展示 + MCP 查询

当前正式支持平台：`macOS`、`Windows`。

## 快速入口

- macOS 使用指南：[docs/setup-macos.md](docs/setup-macos.md)
- Windows 使用指南：[docs/setup-windows.md](docs/setup-windows.md)
- 数据目录与故障排查：[docs/data-layout-and-troubleshooting.md](docs/data-layout-and-troubleshooting.md)
- 开发者工作流：[docs/developer-workflow.md](docs/developer-workflow.md)
- 关系分析算法口径：[docs/relation-analysis.md](docs/relation-analysis.md)
- MCP 配置：[mcp-server/README.md](mcp-server/README.md)

## 核心能力

- 联系人、群聊、词云、情感、全局趋势分析
- 统一的关系分析体验：`客观模式 / 争议模式`
- 关系分析语义收敛：
  - `score` = 关系信号强度
  - `confidence` = 当前结论可信度
- 久未联系联系人会全局下调 `confidence`
- 前台联系人分类只保留 `全部联系人 / 普通联系人 / 已删好友`
- 支持 MCP，在 Claude Code 中直接自然语言查询本地微信数据

## 多平台快速开始

### 1. 先把手机聊天记录迁移到电脑

手机微信 -> 我 -> 设置 -> 通用 -> 聊天记录迁移与备份 -> 迁移到电脑。

### 2. 准备解密产物

默认使用 [ylytdeng/wechat-decrypt](https://github.com/ylytdeng/wechat-decrypt)。

WeLink 只负责目录契约、校验、分析与展示，不内嵌第三方解密核心代码。

期望目录：

```text
decrypted/
├── contact/contact.db
└── message/message_*.db
```

### 3. 生成 `.env`

macOS:

```bash
./scripts/welink-doctor.sh --write-env
```

PowerShell:

```powershell
.\scripts\welink-doctor.ps1 -WriteEnv
```

### 4. 启动

```bash
docker compose up --build
```

也可以直接用一键脚本：

- macOS：`./scripts/start-welink.sh`
- PowerShell：`.\scripts\start-welink.ps1`

- 默认前端：`http://localhost:3000`
- 默认后端：`http://localhost:8080`
- 若你在 `.env` 里改了 `WELINK_FRONTEND_PORT` / `WELINK_BACKEND_PORT`，以 `.env` 为准

### 5. 校验

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/status
```

如果你改了 `.env` 中的 `WELINK_BACKEND_PORT`，把这里的 `8080` 换成实际端口即可。

## 环境变量

项目使用 `.env` 驱动路径配置，核心变量如下：

```env
WELINK_DATA_DIR=/absolute/path/to/decrypted
WELINK_MSG_DIR=/absolute/path/to/msg
WELINK_BACKEND_PORT=8080
WELINK_FRONTEND_PORT=3000
```

- `WELINK_DATA_DIR`：必填
- `WELINK_MSG_DIR`：可选；缺失时会使用仓库内空目录占位

## Fork 后开发流程

1. fork 上游 `runzhliu/welink`
2. clone 个人 fork 到正式工作目录
3. 配置 `origin` / `upstream`
4. 使用 `codex/*` 分支
5. 先做 baseline 同步，再做功能增量提交

详见 [docs/developer-workflow.md](docs/developer-workflow.md)。

## MCP

MCP 查询依赖两件事：

- 数据目录已准备完成
- WeLink 后端已完成索引

详见 [mcp-server/README.md](mcp-server/README.md)。

## Demo 模式

```bash
docker compose -f docker-compose.demo.yml up
```

## 数据安全

所有分析默认在本地执行，不上传聊天数据。请仅处理你有权限访问的数据。
