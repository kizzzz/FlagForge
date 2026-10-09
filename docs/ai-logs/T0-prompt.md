<!-- 提示词归档日期：2026-10-09（Asia/Shanghai）；来源：docs/plan/prompts/T0-scaffold.md -->

# 任务 T0：脚手架与环境探测

## 你的角色

你是 FlagForge 项目的工程脚手架工程师。本任务是整个项目的起点：搭出可运行的空骨架，
让后续任务（数据库 / 引擎 / 前后端）能并行开工。**你不实现任何业务逻辑。**

## 必读（动手前读完）

1. `docs/plan/01-design.md` §2 技术栈、§3 仓库结构
2. `docs/plan/02-edge-cases.md` 组 10（复现与工程）
3. `docs/plan/00-README.md`（了解全局，1 分钟）

## 环境背景

- 工作机：Windows 11 + Git Bash（路径用正斜杠；shell 命令按 Unix 语法写）
- 项目根：`C:\Users\77230\FlagForge`（本文件所在目录的上两级）
- 网络在中国大陆：Go module / npm 拉包失败时优先换国内源
  （`GOPROXY=https://goproxy.cn,direct`；npm 用 `--registry=https://registry.npmmirror.com`），
  仍失败再尝试代理 `http://127.0.0.1:10808`

## 步骤

### 1. 环境探测（结果写入执行报告）

```bash
docker --version && docker compose version
go version          # 需 ≥1.22
node --version      # 需 ≥18
npm --version
```

- 任何缺失：**停止后续步骤，在报告中写明缺什么 + 建议安装命令，等待人工确认**。
  不擅自安装 Docker / Go / Node 等大件。
- 检查端口占用：`netstat -ano | grep -E '8787|5173|55432'`，被占用则报告。

### 2. 后端骨架

- `backend/go.mod`（module 名 `flagforge`，Go 1.22）
- 依赖：gin、gorm、gorm postgres driver、golang-migrate（CLI 使用方式写入 Makefile 注释）、
  gorm.io/datatypes、testify
- `cmd/server/main.go`：读 `DATABASE_URL` 环境变量（默认
  `postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable`），
  Gin 起在 `:8787`，仅注册 `GET /healthz` → `{"status":"ok"}`
- `internal/` 下建空目录占位：config/model/repo/service/eval/api/dto（各放一个 doc.go 或空文件）
- 首次 `go build ./...` 通过

### 3. 前端骨架

- Vite + React 18 + TypeScript 模板生成到 `frontend/`
- 依赖：antd@5、react-router-dom@6、dayjs、msw、axios（或用 fetch，二选一，写进报告）
- dev server 起在 5173；`vite.config.ts` 配 proxy：`/api` → `http://localhost:8787`
- 首页临时放一个 "FlagForge" 标题即可（后续任务替换）

### 4. 基建文件

- `docker-compose.yml`：

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: flagforge
      POSTGRES_PASSWORD: flagforge
      POSTGRES_DB: flagforge
    ports: ["55432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
volumes:
  pgdata:
```

- `Makefile` 目标：`dev`（起 compose + 后台起后端 + 前端 dev）、`migrate-up`、`migrate-down`、
  `seed`、`test`（go test ./...）、`smoke`（占位 echo）、`down`（compose down）
  Windows Git Bash 下 Make 可用性不保证：同时提供 `scripts/*.sh` 等价脚本，README 提及两种方式
- `.gitignore`：node_modules、dist、.env、*.log、backend 的构建产物

### 5. Git

- `git init` + 首次 commit（`chore: scaffold project skeleton`）

## 验收标准（逐条自验后在报告打勾）

- [ ] `docker compose up -d` 后 `pg_isready` 或后端连接成功
- [ ] `cd backend && go build ./...` 零报错
- [ ] `cd backend && go run ./cmd/server` 后 `curl localhost:8787/healthz` 返回 ok
- [ ] `cd frontend && npm run dev` 起在 5173，浏览器见标题
- [ ] `make` 或 `scripts/*.sh` 目标齐全可执行
- [ ] git log 有首次 commit

## 硬约束

- 不实现任何业务表、业务路由、业务页面
- 不安装 Docker/Go/Node 等大件（探测缺失即上报）
- 目录/端口/命名严格按 01-design §3（后续任务按路径找文件）

## 完成后必做

写 `docs/ai-logs/T0-report.md`，模板：

```markdown
# T0 执行报告
## 环境探测结果
（docker/go/node 版本，端口占用情况）
## 做了什么
（文件清单 + 关键选择，例如 fetch vs axios）
## 遇到的问题与解决
## 未尽事项 / 需要人工决策
## 验收自验清单
（逐条打勾 + 命令输出摘要）
```

然后 `git add -A && git commit -m "chore: T0 scaffold done"`。
