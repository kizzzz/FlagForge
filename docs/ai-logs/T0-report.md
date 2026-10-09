# T0 执行报告

执行日期：2026-10-09（Asia/Shanghai）
项目根：`C:/Users/77230/FlagForge`
状态：**骨架构建及前后端运行检查已通过；WSL 已成功安装，系统重启及 PostgreSQL 容器验收待完成。T0 尚未完成，波2尚未启动。**

## 环境探测结果

初次探测时 Docker 和 Go 命令缺失，按提示词停止并报告。用户随后明确授权：「由你安装所有缺失项」。以下为安装后在刷新 PATH 的 Git Bash 中重新探测的结果：

| 工具 / 检查 | 实际结果 | 状态 |
| --- | --- | --- |
| Docker Desktop | 4.94.0，安装器返回 0，安装日志 `Installation succeeded` | 已安装至 `%LOCALAPPDATA%/Programs/DockerDesktop` |
| `docker --version` | `Docker version 29.8.2, build 7fc2dff` | 通过 |
| `docker compose version` | `Docker Compose version v5.5.1` | CLI 通过，容器引擎待启动 |
| `go version` | `go version go1.27.0 windows/amd64` | ≥1.22，安装成功 |
| `node --version` | `v24.13.0` | ≥18，原已安装 |
| `npm --version` | `11.6.2` | 原已安装 |
| Git / Git Bash | `C:/Program Files/Git/bin/bash.exe` 可运行，Git 可用 | 通过 |
| golang-migrate CLI | 已通过 `go install -tags postgres ...@v4.17.1` 安装；`migrate -version` 输出 `dev`（该源码安装未注入发行版本号） | 可执行 |
| WSL 2 | 官方 x64 MSI 重试安装返回 0；`wsl --version` 确认 2.7.13.0，内核 6.18.33.2-2 | 已安装；WSL 2 运行时待系统重启 |
| VirtualMachinePlatform | Windows 管理员辅助脚本确认 `Enabled` | 已启用，首次启用报告需重启 |
| Microsoft-Windows-Subsystem-Linux | `Enabled`，`RestartNeeded: true` | 已启用，待重启 |
| 8787 / 5173 / 55432 | 初次及安装后探测均未发现占用；本次前后端验收临时使用 8787 / 5173 | 无原有端口冲突 |

系统为 Windows 11 家庭版中文版，64位，build 26200。Intel i5-8300H 的固件虚拟化及 SLAT 已启用；LanmanServer 运行且自动启动。当前 Codex 进程未以管理员身份运行，软件安装及 Windows 功能启用通过系统 UAC 提权；未自动重启电脑。

## 做了什么

- 动手前阅读 `docs/plan/00-README.md`、`01-design.md` §2/§3、`02-edge-cases.md` 组10，并按 T0 提示词执行。
- 将原始提示词完整归档至 `docs/ai-logs/T0-prompt.md`，含日期与来源。
- `backend/go.mod` / `go.sum`：module `flagforge`，`go 1.22`；依赖 Gin、GORM、PostgreSQL driver、golang-migrate、datatypes、testify。使用兼容 Go 1.22 的固定版本，未加入业务逻辑。
- `backend/cmd/server/main.go`：读取 `DATABASE_URL`，空时使用指定默认值；Gin 监听 `:8787`，仅注册 `GET /healthz`。T0 读取数据库配置，连接及业务 API 由后续任务接入。
- `backend/internal/{config,model,repo,service,eval,api,dto}/doc.go`：空包占位。
- `frontend/`：以 `create-vite@5.5.2` 的 React TypeScript 模板生成，保留 React 18；安装 antd@5、react-router-dom@6、dayjs、msw@2；首页仅显示 FlagForge。
- **API 选择原生 fetch**，遵循设计 §3。T0 不添加 API client 或业务页面，后续 T4a 实现。
- Vite 使用端口 5173、`strictPort: true`，`/api` 代理至 `http://localhost:8787`，浏览器标题为 FlagForge。
- `docker-compose.yml`：postgres:16，指定用户、密码、数据库，`55432:5432`，命名卷 pgdata。
- `Makefile` 与 `scripts/{dev,migrate-up,migrate-down,seed,test,smoke,down}.sh`，共享 `scripts/common.sh`。dev 等待 PostgreSQL 就绪，已有迁移/seed 时运行，然后后台启动后端及前端；T0 尚无迁移和 seed，按条件跳过。
- migrate 脚本查找 PATH 或 GOPATH/bin 下的 CLI；若后续提供 `backend/cmd/migrate/main.go` 则作为备选。migrate-down 回退最近一次迁移。
- `README.md`：前置依赖、Make/Git Bash 两种方式、端口、默认 DATABASE_URL、国内源及 T0/T1 的交付边界。
- `.gitignore` 覆盖 node_modules、dist、.env、日志、后端构建产物；`.gitattributes` 固定 Bash 脚本 LF；脚本设置可执行位。
- Git 已初始化于项目根，分支 `codex/scaffold`；首次提交 `b4d0310 chore: scaffold project skeleton` 已完成并核对 git log。

实际前端安装结果：React / React DOM 18.3.1、AntD 5.29.3、react-router-dom 6.30.6、dayjs 1.11.23、MSW 2.15.0；Vite 构建版本为 5.4.21。未实现业务表、业务路由或业务页面。

## 遇到的问题与解决

1. **缺少 Docker / Go**：首次探测按提示词停止；获得用户明确安装授权后继续。
2. **WinGet 下载停在 Delivery Optimization**：日志记录 Go 下载机制超时后回退 WinINet。Docker/WSL 的 HTTP 连通性及小范围请求成功，改用官方地址直接下载；WSL 直连速度较慢，按文档使用 `http://127.0.0.1:10808` 续传，未修改全局代理或 WinGet 设置。
3. **安装包完整性**：Docker 包 635493296 字节，SHA-256 `a9814e31049d66156477a86614e83365669677733014ec72f74229623ff3890a`，与 WinGet 清单一致，数字签名 Valid / Docker Inc。Go MSI 校验为 `728e9318c061b95082bcf167bab8b6c6e0d91a045f89f403b3285fa39f0c455b`，原始 MSI 日志最终记录安装状态 0。WSL MSI 258985984 字节，SHA-256 `a3505a50f4cc585551d11d9de824ba4375448d7a68f2e71d3fb315fa986fc754`，与 Microsoft/WSL 官方 release API 一致，数字签名 Valid / Microsoft Corporation。
4. **Windows 功能辅助脚本首次记录失败**：误读取 Enable-WindowsOptionalFeature 返回对象中不存在的 State 字段，功能实际已启用。改为重新查询 Get-WindowsOptionalFeature；再次运行返回 0，并写入 `docs/ai-logs/T0-wsl-features.json`。两次启用均使用 `-NoRestart`。
5. **Go 安装器并行状态**：WinGet 启动器长时间等待 UAC，直接 MSI 重试返回 1618（另一安装正在执行）。核对原始 MSI 日志确认原安装已成功，随后 Git Bash `go version` 通过；未反复重装。
6. **WSL UAC 取消及重试**：首次安装因 UAC 取消未开始；用户随后明确授权「重试并继续」。使用已校验 MSI 重试返回 0，安装日志确认成功；`wsl --version` 返回 0，版本为 2.7.13.0。
7. **Docker 引擎不可用**：实际 `docker compose up -d` 返回 1：`failed to connect to the docker API ... open //./pipe/docker_engine: The system cannot find the file specified`。WSL 安装及重启后再启动 Docker Desktop 并重验。

安装依据：[Docker Windows 官方说明](https://docs.docker.com/desktop/setup/install/windows-install/)、[Microsoft WSL 安装说明](https://learn.microsoft.com/en-us/windows/wsl/install)、[WSL 2.7.13 官方发布](https://github.com/microsoft/WSL/releases/tag/2.7.13)。WSL 已校验安装包缓存于 `%LOCALAPPDATA%/Temp/FlagForge-T0-install/wsl.2.7.13.0.x64.msi`，重试可直接使用，无须重新下载。

## 未尽事项 / 需要人工决策

- WSL 重试安装已成功，本次缺失软件已安装；无需再次确认安装授权。
- Windows 已明确报告功能启用需要重启。请保存工作后人工重启，重新打开 Codex/终端刷新 PATH，并启动 Docker Desktop。未执行自动重启。
- 重启后确认 `wsl --version`、Docker daemon；执行 `docker compose up -d` 和 `docker compose exec -T postgres pg_isready -U flagforge -d flagforge`。
- PostgreSQL 验收通过后，更新本报告并执行指定最终提交 `chore: T0 scaffold done`。
- **仅 T0 完整验收后，才同时启动波2的 T1、T2、T4a 三个 agent。当前尚未启动。**
- migrate/seed 的具体内容属于 T1；当前独立执行相应脚本会明确提示 T1 文件尚未交付，不能将其视为已验证的数据层。

## 验收自验清单

- [ ] `docker compose up -d` 后 pg_isready 或后端连接成功——Docker daemon 尚未可用；WSL 已安装，待系统重启使 Windows 可选功能生效后继续验收。
- [x] `cd backend && go build ./...`——`go build -mod=mod ./...` 返回 0，另 `go vet ./...`、`go test ./...` 返回 0；T0 各包均 `[no test files]`。
- [x] `cd backend && go run ./cmd/server` 后 healthz——Gin 监听 8787；HTTP 200，body 为 `{"status":"ok"}`。
- [x] `cd frontend && npm run dev` 起在 5173，浏览器见标题——Vite 输出 `http://localhost:5173/`；HTTP 200，Codex 浏览器实际 DOM 为 heading FlagForge；`npm run build` / `npm run lint` 均返回 0。
- [x] Make 或 scripts 目标齐全可执行——所有 Bash 文件 `bash -n` 返回 0；test、smoke 实跑；migrate/seed 对尚无 T1 文件返回预期提示；Compose 配置 `docker compose config --quiet` 返回 0。机器未探测到 make，使用脚本等价入口；Docker 相关运行验收仍受前述引擎限制。
- [x] git log 有首次 commit——`b4d0310 chore: scaffold project skeleton`，脚本 Git mode 全部为 100755。

仅以上已实际执行并通过的项目打勾；不能将骨架构建成功视为完整 T0 验收通过。

临时 healthz / Vite 验收服务及浏览器页面已关闭，释放 8787 / 5173；源码、依赖和已校验安装包保留，等待恢复验收。

## WSL 重试安装结果（2026-10-09）

- 用户指令：「重试并继续」。重新核对缓存 MSI 的 SHA-256 后，以 `/qn /norestart` 安装，MSI exit code 0，日志 `MainEngineThread is returning 0`。
- `wsl --version` 返回 0：WSL 2.7.13.0 / Linux kernel 6.18.33.2-2 / Windows 10.0.26200.9457。
- `wsl --status` 的命令退出码为 0，但正文明确提示「WSL2 无法启动，因为此计算机上未启用虚拟化」。不能仅凭退出码判定运行时已可用。
- `HKLM/SOFTWARE/Microsoft/Windows/CurrentVersion/Component Based Servicing/RebootPending` 存在；本次功能启用后系统尚未重启。`LastBootUpTime` 仍为 2026-10-01 12:09:29。
- 原始硬件探测显示固件虚拟化和 SLAT 已开启；Windows 可选功能已启用，但此前已明确报告 RestartNeeded。先人工重启，使新功能生效，再检查运行时；本次未修改 BIOS 或自动重启机器。
- 已尝试启动 Docker Desktop，daemon 探测仍返回 1（docker_engine 管道不存在）。T0 的 PostgreSQL 验收尚未通过，因此不会创建 `chore: T0 scaffold done` 提交或提前启动波2。
- 恢复入口：人工保存其他工作并重启 → 重新打开本聊天，告知「已重启，继续」→ 检查 WSL/Docker → 启动 postgres:16 并 pg_isready → 完成 T0 报告和提交 → 同时启动 T1 ∥ T2 ∥ T4a。