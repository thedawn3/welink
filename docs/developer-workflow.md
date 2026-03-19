# 开发者工作流

## 仓库基线

- 上游：`runzhliu/welink`
- 个人 fork：`<your-account>/welink`
- 本地正式工作目录：建议 `projects/welink`
- 临时验证目录：仅用于实验，不作为长期基线

## 推荐流程

```bash
gh repo fork runzhliu/welink --clone=false
git clone https://github.com/<you>/welink.git
cd welink
git remote add upstream https://github.com/runzhliu/welink.git
git checkout -b codex/<topic>
```

## 分支与提交约定

- 分支前缀：`codex/`
- 先做 `baseline sync` 提交，再做功能增量提交
- 不提交本机绝对路径、解密数据、媒体文件、`.env`

## 本地配置

- 把 `.env.example` 复制成 `.env`
- 路径统一放进 `.env`
- `docker-compose.yml` 不允许再写死个人机器路径

## 提交前检查

```bash
cd backend && go test ./...
cd ../frontend && npm ci && npm run build
```

## 文档要求

只要改了以下任一能力，就同步文档：

- 启动方式 / 环境变量 / 目录契约
- 关系分析口径
- MCP 前置依赖
- 跨平台脚本与排障流程
