# FlagForge

Feature Flag 与灰度规则平台。当前为 T0 工程骨架：后端提供 `GET /healthz`，前端显示 FlagForge 标题。后续任务按 `docs/plan/` 完成数据层、规则引擎和业务页面。

## 环境

- Go ≥1.22、Node ≥18、npm、Docker Desktop（含 Compose，Linux 容器模式）。
- Windows 使用 Git Bash；安装工具后重新打开终端以刷新 PATH。
- 端口：后端 8787、前端 5173、PostgreSQL 55432。
- Docker 使用 WSL 2 时，首次启用 Windows 虚拟机平台后可能需要重启电脑。

## 快速开始

在项目根目录的 Git Bash 执行：

```bash
cd backend
go mod download
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
cd ../frontend
npm ci
cd ..
bash scripts/dev.sh
```

安装了 GNU Make 时，最后一步也可运行 `make dev`。脚本会启动 PostgreSQL、等待就绪、执行已有迁移和 seed、后台启动后端，再启动前端。T0 阶段迁移和 seed 尚未交付，dev 会跳过；T1 交付后自动执行。前端访问 [localhost:5173](http://localhost:5173)，健康检查访问 [localhost:8787/healthz](http://localhost:8787/healthz)。停止 dev 使用 Ctrl+C；数据库通过 `bash scripts/down.sh` 停止。

默认 `DATABASE_URL` 为：

```text
postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable
```

通过环境变量覆盖；T0 只读取配置，后续数据/API 任务接入数据库。

## 常用命令

| Make | Git Bash 等价命令 | 作用 |
| --- | --- | --- |
| `make dev` | `bash scripts/dev.sh` | 启动开发环境 |
| `make migrate-up` | `bash scripts/migrate-up.sh` | 应用迁移（T1 提供迁移文件） |
| `make migrate-down` | `bash scripts/migrate-down.sh` | 回退最近一次迁移 |
| `make seed` | `bash scripts/seed.sh` | 演示数据（T1 提供） |
| `make test` | `bash scripts/test.sh` | `go test ./...` |
| `make smoke` | `bash scripts/smoke.sh` | T0 占位，T5 补全冒烟序列 |
| `make down` | `bash scripts/down.sh` | 停止 Compose，不删除数据卷 |

独立启动：`docker compose up -d`、`cd backend && go run ./cmd/server`、`cd frontend && npm run dev`，分别在独立终端执行。前端 `/api` 经 Vite 转发到 `http://localhost:8787`；使用原生 fetch。

中国大陆拉包失败时使用 `GOPROXY=https://goproxy.cn,direct` 和 `npm ci --registry=https://registry.npmmirror.com`，仍失败再使用 `http://127.0.0.1:10808` 代理。

设计与验收要求见 `docs/plan/`，执行记录见 `docs/ai-logs/`。
