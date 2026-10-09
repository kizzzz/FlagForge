# FlagForge

Feature Flag 与灰度规则平台。已完成 T0、T1、T2、T4a：工程骨架、SQL 迁移与幂等演示数据、纯函数评估引擎，以及 Flag 列表和新建/编辑页面。后端当前提供 `GET /healthz`；前端开发模式默认使用 MSW。业务 API 与详情/控制台页面由后续 T3a/T3b、T4b 接入，计划见 `docs/plan/`。

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

安装了 GNU Make 时，最后一步也可运行 `make dev`。脚本会启动 PostgreSQL、等待就绪、执行迁移和幂等 seed、后台启动后端，再启动前端。前端访问 [localhost:5173](http://localhost:5173)，健康检查访问 [localhost:8787/healthz](http://localhost:8787/healthz)。停止 dev 使用 Ctrl+C；数据库通过 `bash scripts/down.sh` 停止。

开发开关为 `frontend/.env.development` 中的 `VITE_USE_MSW=true`；生产构建默认关闭。Mock 使用内存演示数据，刷新会重置；`billing-portal` 的首次启停故意失败一次，用于验证回滚，重试会成功。关闭 mock 后接真实业务 API 的成功联调属于后续任务。

默认 `DATABASE_URL` 为：

```text
postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable
```

通过环境变量覆盖；迁移、seed 和数据层测试使用此连接。服务端业务 API 的数据库接入属于后续任务。

## 常用命令

| Make | Git Bash 等价命令 | 作用 |
| --- | --- | --- |
| `make dev` | `bash scripts/dev.sh` | 启动开发环境 |
| `make migrate-up` | `bash scripts/migrate-up.sh` | 应用 SQL 迁移 |
| `make migrate-down` | `bash scripts/migrate-down.sh` | 回退最近一次迁移 |
| `make seed` | `bash scripts/seed.sh` | 幂等写入演示数据 |
| `make test` | `bash scripts/test.sh` | `go test ./...` |
| `make smoke` | `bash scripts/smoke.sh` | T0 占位，T5 补全冒烟序列 |
| `make down` | `bash scripts/down.sh` | 停止 Compose，不删除数据卷 |

独立启动：`docker compose up -d`、`cd backend && go run ./cmd/server`、`cd frontend && npm run dev`，分别在独立终端执行。前端 `/api` 经 Vite 转发到 `http://localhost:8787`；使用原生 fetch。

后端普通检查使用 `bash scripts/test.sh`。真实数据库集成测试在应用迁移后执行：

```bash
cd backend
DATABASE_URL='postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable' go test -tags=integration ./... -count=1
go test ./internal/eval/ -cover
```

前端使用 `npm run build`、`npm run lint` 和浏览器验收；按设计不添加前端测试框架。当前波次结果及验证限制见 `docs/ai-logs/wave2-report.md`。

中国大陆拉包失败时使用 `GOPROXY=https://goproxy.cn,direct` 和 `npm ci --registry=https://registry.npmmirror.com`，仍失败再使用 `http://127.0.0.1:10808` 代理。

设计与验收要求见 `docs/plan/`，执行记录见 `docs/ai-logs/`。
