# AI 端到端部署提示词

这份文件的用途只有一个：给另一台机器上的 AI 一段尽可能短的指令，让它自己完成 `电脑微信原始数据 -> wechat-decrypt 解密 -> WeLink 启动 -> 验收` 的全流程。

## 最短可用提示词

如果目标 AI 可以读取仓库文件，直接给它这句就够：

```text
请克隆 https://github.com/thedawn3/welink ，然后严格按仓库里的 docs/ai-end-to-end-deploy-prompt.md 完成 WeLink 端到端部署：先确认电脑微信聊天记录完整，再用 wechat-decrypt 解密出标准目录，然后启动 WeLink，最后完成接口验收并报告结果。
```

如果你想再短一点，可用：

```text
克隆 https://github.com/thedawn3/welink ，按 docs/ai-end-to-end-deploy-prompt.md 部署并验收 WeLink。
```

## 给 AI 的完整执行说明

```text
你现在在一台新机器上，请严格按仓库文档完成 WeLink 的端到端部署与验收。

目标：
1. 先确认电脑微信里已有完整聊天记录
2. 如果手里还没有标准目录，先用 ylytdeng/wechat-decrypt 在容器外解密
3. 拿到标准目录后，再启动 WeLink
4. 启动后完成接口验收
5. 输出实际执行结果、阻塞原因和下一步建议

仓库地址：
- WeLink: https://github.com/thedawn3/welink
- wechat-decrypt: https://github.com/ylytdeng/wechat-decrypt

如果当前机器还没有 WeLink 仓库，先执行：
git clone https://github.com/thedawn3/welink.git
cd welink

仓库内必须优先阅读：
1. README.md
2. AI_DEPLOY.md
3. docs/deploy-docker.md
4. 当前平台对应的 setup 文档：
   - Windows: docs/setup-windows.md
   - macOS: docs/setup-macos.md
5. docs/data-layout-and-troubleshooting.md

执行边界：
- wechat-decrypt 负责从运行中的微信进程和本地数据库生成标准目录
- WeLink 只负责消费标准目录并分析展示
- 不要把原始 xwechat_files 根目录直接传给 WeLink 的 DataDir 或 SourceDataDir
- 如果没找到标准目录，必须停在“缺少标准目录”，不要擅自拿原始目录顶上

标准目录定义：
<STANDARD_DIR>/
  contact/contact.db
  message/message_*.db
  sns/sns.db                 # optional

正式模式只允许两种：
- analysis-only：只有一个已解密标准目录时使用
- manual-sync：同时有 analysis 标准目录和 source 标准目录时使用

固定要求：
- 先确认电脑微信里真的有聊天记录；如果电脑微信本身不完整，后续结果也不会完整
- 如果还没有标准目录，先运行 ylytdeng/wechat-decrypt
- 在运行 wechat-decrypt 前，必须确认微信桌面端正在运行
- Windows 必须先执行 Set-ExecutionPolicy -Scope Process Bypass -Force
- Python 3 必须可用
- docker compose version 必须可用
- source 和 analysis 不能是同一路径

启动 WeLink 前必须先输出：
1. 仓库实际路径
2. 当前平台
3. 标准目录路径
4. 目录完整性检查结果：
   - contact/contact.db 是否存在
   - message/message_*.db 是否存在
   - sns/sns.db 是否存在

Windows 平台额外必须先打印：
- docker compose version
- py -3 --version 或 python --version

如果 config-check 仍阻塞：
- 禁止继续尝试 start_decrypt 或猜测目录
- 必须先输出阻塞原因和修复建议

启动 WeLink 后固定验收：
curl http://localhost:8080/api/health
curl http://localhost:8080/api/system/config-check
curl http://localhost:8080/api/system/runtime
curl http://localhost:8080/api/status

如 is_initialized=false，再执行：
curl -X POST http://localhost:8080/api/system/reindex

排障顺序固定：
1. /api/system/config-check
2. /api/system/runtime
3. /api/system/logs

输出要求：
1. 仓库路径
2. 平台
3. 电脑微信聊天记录是否已确认完整
4. wechat-decrypt 是否执行成功
5. 最终使用的标准目录路径
6. 目录完整性检查结果：
   - contact/contact.db
   - message/message_*.db
   - sns/sns.db
7. 最终使用的是 analysis-only 还是 manual-sync
8. 4 条验收接口结果
9. 阻塞项 / 下一步修复建议
```

## 平台提醒

- Windows：完整步骤看 `docs/setup-windows.md`
- macOS：完整步骤看 `docs/setup-macos.md`
- Docker 模式与 `.env` 看 `docs/deploy-docker.md`
- 图片 key / `sns.db` / 目录排障看 `docs/data-layout-and-troubleshooting.md`
- 图片 key 失败不阻塞基础文本部署；先完成基础验收，再回头补图片能力
