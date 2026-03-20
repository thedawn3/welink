# macOS 使用指南

本页只负责 macOS 平台差异。Docker 模式、`.env`、目录契约、红色阻塞错误，以 [`deploy-docker.md`](./deploy-docker.md) 和 [`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md) 为准。

## 1. macOS 一键部署前提

在 macOS 机器上让 AI 拉仓并启动前，先确认：

- 已安装 Git
- 已安装 Docker Desktop，且 `docker compose version` 可用
- 已安装 Python 3，且 `python3 --version` 可用
- 已准备至少一个标准目录：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

可选目录：

- `msg`：只影响聊天图片缩略图 / 点击查看
- `wechat-decrypt`：只影响自动读取 `image_keys.json` / `config.json`

## 2. 先判断你属于哪种正式模式

### `analysis-only`

- 你已经有一个可直接分析的标准目录
- 只给 WeLink 一个 `analysis directory`
- Docker 不做容器内 watcher 或自动解密

### `manual-sync`

- 你除了 analysis 目录，还要额外提供一个 source 标准目录
- `source` 和 `analysis` 都必须是标准目录
- `source` 不能直接指向 `xwechat_files` 根目录
- `source` 与 `analysis` 不能是同一路径

## 3. macOS AI 一键启动命令

### `analysis-only`

```bash
git clone https://github.com/thedawn3/welink.git
cd welink
./scripts/start-welink.sh \
  --mode analysis-only \
  --data-dir /absolute/path/to/analysis_standard_dir \
  --msg-dir /absolute/path/to/msg \
  --wechat-decrypt-dir /absolute/path/to/wechat-decrypt
```

### `manual-sync`

```bash
git clone https://github.com/thedawn3/welink.git
cd welink
./scripts/start-welink.sh \
  --mode manual-sync \
  --data-dir /absolute/path/to/analysis_standard_dir \
  --source-data-dir /absolute/path/to/source_standard_dir \
  --msg-dir /absolute/path/to/msg \
  --wechat-decrypt-dir /absolute/path/to/wechat-decrypt
```

可选参数说明：

- `--msg-dir`：可省略；省略后不影响文本分析，但聊天图片缩略图 / 点击查看不可用
- `--wechat-decrypt-dir`：可省略；省略后不影响基础部署，但 V2 图片不会自动读取 `image_keys.json`

## 4. 如果你还没有标准目录

默认建议用 [`ylytdeng/wechat-decrypt`](https://github.com/ylytdeng/wechat-decrypt) 在容器外先准备标准目录。

典型流程：

```bash
git clone https://github.com/ylytdeng/wechat-decrypt
cd wechat-decrypt
sudo python3 main.py
```

当前仓库已实际验证：

- 该工具可产出 `sns/sns.db`
- 该工具目录可包含 `image_keys.json` / `config.json`
- WeLink 配置 `WELINK_WECHAT_DECRYPT_DIR` 后可自动读取这些结果

## 5. 启动后只看这几条验收命令

```bash
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

## 6. macOS 平台常见坑

### `python3` 不可用

WeLink 的 doctor 依赖 Python 3：

```bash
python3 --version
```

如果没有 Python 3，先装好再运行 `start-welink.sh`。

### 把 `xwechat_files` 根目录当成标准目录

这会导致：

- doctor 提示缺少 `contact/contact.db` 或 `message/message_*.db`
- `config-check` 提示 source 不是标准目录
- 系统页拒绝同步

正确做法是先用容器外工具整理出标准目录，再把标准目录传给 WeLink。

### Docker 看不到目录

先检查：

- Docker Desktop 是否允许访问对应目录
- 路径是否真实存在
- `source` 与 `analysis` 是否错误地写成同一路径

### 图片仍只显示 `[图片]`

优先检查：

- `msg` 目录是否正确
- `WELINK_WECHAT_DECRYPT_DIR` 是否指向真正的 `wechat-decrypt` 工具目录
- 该目录下是否已经生成 `image_keys.json`

必要时可在仓库根目录执行：

```bash
./scripts/extract-image-key.sh --restart
```

## 7. 图片与 SNS

- `ylytdeng/wechat-decrypt` 已验证可产出 `sns/sns.db`
- `WELINK_MSG_DIR` 正确时，聊天流可回溯图片
- `WELINK_WECHAT_DECRYPT_DIR` 正确时，WeLink 会自动尝试读取 `image_keys.json` / `config.json`

更完整的目录与图片排障，统一看 [`data-layout-and-troubleshooting.md`](./data-layout-and-troubleshooting.md)。

## 8. 下一步

macOS 上只要部署验收通过，接下来按这个顺序继续：

1. 打开前端：`http://localhost:3000`
2. 在系统页确认 `config-check` 与 `runtime`
3. 如需 AI 查询，再接 [`../mcp-server/README.md`](../mcp-server/README.md)
