# WeLink AI Deploy Entry

如果你要把本仓库直接丢给另一台机器上的 AI，这是唯一推荐入口。

仓库地址：

- `https://github.com/thedawn3/welink`

## 最短提示词

```text
克隆 https://github.com/thedawn3/welink ，按 AI_DEPLOY.md 部署并验收 WeLink。
```

## 稍强一点的提示词

```text
请克隆 https://github.com/thedawn3/welink ，然后严格按 AI_DEPLOY.md 完成 WeLink 端到端部署：先确认电脑微信聊天记录完整，再用 wechat-decrypt 解密出标准目录，然后启动 WeLink，最后完成接口验收并报告结果。
```

## AI 必须完成的事

1. 先确认电脑微信里已经有完整聊天记录
2. 如果没有标准目录，先用 `ylytdeng/wechat-decrypt` 解密
3. 拿到标准目录后，再启动 WeLink
4. 启动后完成接口验收
5. 明确输出成功结果或阻塞原因

硬规则：

- 如果仓库尚未 clone，先 clone
- 如果手里不是标准目录，先解密
- 不要把 `xwechat_files` 根目录直接传给 WeLink
- 如果 `config-check` 仍阻塞，先输出阻塞原因，不要继续猜路径或硬启动

## AI 必须优先阅读

1. `README.md`
2. `docs/ai-end-to-end-deploy-prompt.md`
3. `docs/deploy-docker.md`
4. 当前平台文档：
   - `docs/setup-windows.md`
   - `docs/setup-macos.md`
5. `docs/data-layout-and-troubleshooting.md`

## 关键边界

- `wechat-decrypt` 负责从运行中的微信进程和本地数据库生成标准目录
- WeLink 只负责消费标准目录并分析展示
- 不要把原始 `xwechat_files` 根目录直接传给 WeLink

标准目录必须是：

```text
<STANDARD_DIR>/
├── contact/contact.db
├── message/message_*.db
└── sns/sns.db                 # optional
```

## WeLink 正式模式

- `analysis-only`：只有一个已解密标准目录时使用
- `manual-sync`：同时有 `analysis` 标准目录和 `source` 标准目录时使用

## 固定验收

```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/status
```

如果 `is_initialized=false`，再执行：

```bash
curl -X POST http://localhost:8080/api/system/reindex
```

## 固定排障顺序

1. `/api/system/config-check`
2. `/api/system/runtime`
3. `/api/system/logs`

## 详细版

完整长提示词在：

- `docs/ai-end-to-end-deploy-prompt.md`
