# T0 执行报告

执行日期：2026-10-09（Asia/Shanghai）
项目根：`C:/Users/77230/FlagForge`
状态：**T0 已完成，六项验收全部通过，可进入波2。**

## 环境探测结果

| 检查 | 最终结果 |
| --- | --- |
| Windows | Windows 11 家庭版中文版，build 26200；本次重启时间 2026-10-09 22:59:32 |
| WSL | 2.7.13.0，内核 6.18.33.2-2；`wsl --status` 正常，默认版本 2 |
| 虚拟化 | `HypervisorPresent: true`；待重启标记已清除 |
| Docker Desktop | 4.94.0；Linux Engine 29.8.2，API 1.56 |
| Docker CLI / Compose | Docker 29.8.2 / Compose v5.5.1 |
| Go | `go1.27.0 windows/amd64`，满足 ≥1.22；项目 go.mod 为 `go 1.22` |
| Node / npm | v24.13.0 / 11.6.2，满足 Node ≥18 |
| golang-migrate | 已安装 CLI，固定源码版本 v4.17.1；源码构建 `-version` 显示 dev |
| PostgreSQL | postgres:16，运行版本 16.15 (Debian 16.15-1.pgdg13+2)；pg_isready 接受连接 |
| 端口 | 初次探测 8787/5173/55432 无原有占用；当前数据库使用 55432；临时验收服务已关闭并释放 8787/5173 |

## 做了什么

- 必读文档：00-README、01-design §2/§3、02-edge-cases 组10；原始提示词归档至 `docs/ai-logs/T0-prompt.md`。
- `backend/go.mod` / `go.sum`：module flagforge，Go 1.22，固定 Gin/GORM/PostgreSQL driver/golang-migrate/datatypes/testify 依赖版本。
- `backend/cmd/server/main.go`：读取 DATABASE_URL，空时使用指定默认 URL；Gin 监听 :8787，仅注册 GET /healthz。
- `backend/internal/{config,model,repo,service,eval,api,dto}/doc.go`：空包占位，未实现业务逻辑。
- `frontend/`：create-vite@5.5.2 的 React TypeScript 模板，React / React DOM 18.3.1；AntD 5.29.3、react-router-dom 6.30.6、dayjs 1.11.23、MSW 2.15.0，Vite 5.4.21。
- **选择原生 fetch**，遵循设计 §3；API client 留给 T4a。首页仅显示 FlagForge；Vite 5173 / strictPort / `/api` → http://localhost:8787。
- `docker-compose.yml`：postgres:16，指定账号和数据库，55432:5432，命名卷 pgdata。
- `Makefile` 与 `scripts/{dev,migrate-up,migrate-down,seed,test,smoke,down}.sh`，共享 common.sh。dev 启动 Compose、等待数据库、执行已有迁移/seed、后台起后端、起前端；Ctrl+C 清理服务。
- migrate 脚本支持 PATH/GOPATH 下的 CLI，以及后续 `cmd/migrate/main.go` 库 API 备选；README 明确 Make 和 Git Bash 两种方式。T1 尚未提供迁移/seed 时，dev 跳过这两步；独立调用相应脚本会给出明确提示。
- `.gitignore` 覆盖 node_modules、dist、.env、日志、后端构建产物和并行任务工作目录 .worktrees/；Bash 文件通过 .gitattributes 固定 LF，Git mode 100755。
- Git 初始化于项目根，分支 codex/scaffold；首次提交 `b4d0310 chore: scaffold project skeleton`。最终按要求提交 `chore: T0 scaffold done`。

## 遇到的问题与解决

1. 初次缺少 Docker / Go，按 T0 提示词停止报告。用户授权「由你安装所有缺失项」后安装；WSL 首次 UAC 取消后，经用户「重试并继续」授权重试成功。用户完成系统重启后，虚拟化和 Docker 引擎正常。
2. WinGet Delivery Optimization 下载停滞，使用官方地址直接下载并校验 SHA-256 / 数字签名；WSL 慢速下载按提示词用 http://127.0.0.1:10808 续传。未修改全局代理或 WinGet 配置。
3. Docker 安装包 SHA-256 `a9814e31049d66156477a86614e83365669677733014ec72f74229623ff3890a`；Go MSI `728e9318c061b95082bcf167bab8b6c6e0d91a045f89f403b3285fa39f0c455b`；WSL MSI `a3505a50f4cc585551d11d9de824ba4375448d7a68f2e71d3fb315fa986fc754`，均与官方清单 / release API 一致。
4. Windows 功能辅助脚本首次误读取 Enable-WindowsOptionalFeature 返回值的 State；修正为重新查询后运行成功。Go 直接 MSI 重试曾遇 1618，确认原始安装已成功后未重复重装。原始安装经过保留在此前 Git 报告版本和本地 *.log 中。
5. Docker Desktop 首次启动需初始化 WSL 数据盘，postgres:16 首次拉取较慢；等待下载及解压完成后 Compose 启动成功，pg_isready 和 SQL 版本查询通过。

安装来源：[Docker 官方说明](https://docs.docker.com/desktop/setup/install/windows-install/)、[Microsoft WSL 说明](https://learn.microsoft.com/en-us/windows/wsl/install)、[WSL 2.7.13 官方发布](https://github.com/microsoft/WSL/releases/tag/2.7.13)。

## 未尽事项 / 需要人工决策

- **T0 无未完成项，无待人工决策。**
- 迁移、seed、业务 API 和业务页面分别属于后续任务，本次未提前实现。
- PostgreSQL 保持运行，供 T1 使用；Compose project 名为 flagforge，容器为 flagforge-postgres-1。
- 按用户要求，T0 完成后同时启动 T1 ∥ T2 ∥ T4a；使用独立 Git worktree，完成后审查和合并。

## 验收自验清单

- [x] `docker compose up -d` 后 PG 可连——容器 flagforge-postgres-1 正常运行；`docker compose exec -T postgres pg_isready -U flagforge -d flagforge` 输出 accepting connections；psql SHOW server_version 返回 16.15。
- [x] `cd backend && go build ./...` 零报错——返回 0；go vet ./...、go test ./... 同样返回 0（T0 空包暂无测试）。
- [x] `cd backend && go run ./cmd/server` 后 curl healthz 返回 ok——HTTP 200，body `{"status":"ok"}`。
- [x] `cd frontend && npm run dev` 起在 5173，浏览器见标题——Vite 正常启动；Codex 浏览器实际 heading FlagForge；npm run build 和 npm run lint 返回 0。
- [x] Make 或 scripts 目标齐全可执行——所有 Bash 文件语法通过；test/smoke 实跑；`bash scripts/dev.sh` 成功启动 Compose、后端、前端，healthz 和首页均 HTTP 200，Ctrl+C 后 8787/5173 无监听。迁移/seed 的交付边界明确。
- [x] git log 有首次 commit——`b4d0310 chore: scaffold project skeleton`；脚本均为 100755。