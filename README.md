# FlagForge

## 1. 简介与完成范围

面向小团队的 Feature Flag 与灰度规则平台：按环境管理开关，通过评估轨迹解释每个用户为什么命中规则或拿到默认值。

| 题面必做 | 状态 |
| --- | --- |
| React/TypeScript 前端、Go API、PostgreSQL 持久化 | 已实现，默认连接真实 Go 服务 |
| Flag 列表、搜索/筛选、分页、新建、编辑、详情、启停、删除 | 已实现；同环境 key 唯一、跨环境可复用；删除需确认 |
| 规则增删改、equals/in、确定性优先级 | 已实现；数字越小越优先，重复优先级返回 409 |
| 在线评估、结果/命中规则/默认值/停用与错误反馈 | 已实现；JSON 校验、加载态、空态、成功/失败反馈 |
| 操作历史及原子一致性 | 已实现；业务与历史同一 PostgreSQL 事务 |
| SQL 迁移、幂等 seed、核心自动化测试、AI 记录 | 已实现；逐项验证见 docs/ai-logs/T5-acceptance.md |

四个亮点均已实现：逐规则 **Trace**、最多 100 组的**批量模拟分布**、结构化历史 **Diff**、**环境复制**（复制后停用）。未实现的可选项与验证限制见第 8 节。

## 2. 快速开始

依赖：Go ≥1.22、Node 24.x + npm（本次验证 Node 24.13.0 / npm 11.6.2）、Git Bash（Windows）或 Bash、curl、Docker Desktop/Engine + Compose（Linux 容器）。计划的 Node 18 基线满足 smoke 自身，但当前锁定的前端传递依赖要求更高版本；前端请使用已验证的 Node 24.x。使用 PostgreSQL 16，可改用已有本地 PostgreSQL。GNU Make 可选。默认端口：前端 5173、Go 8787、PG 55432。Windows 安装后重新打开终端刷新 PATH；Docker Desktop 使用 WSL2 时首次启用虚拟机平台可能需要重启。

在仓库根目录 Git Bash/Bash 安装迁移工具与依赖：

```bash
cd backend
go mod download
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
cd ../frontend
npm ci
cd ..
```

`migrate` 需位于 PATH 或 `$(go env GOPATH)/bin`（脚本自动查找，Windows 包括 `.exe`）。前置安装后，一条主命令启动全套：

```bash
bash scripts/dev.sh
# 安装 GNU Make 时等价：make dev
```

脚本依次 Compose up → 等待 PG → migrate up → seed → build/run Go → Vite。打开 [localhost:5173](http://localhost:5173)，健康检查 [localhost:8787/healthz](http://localhost:8787/healthz)。Ctrl+C 停止前后端；`bash scripts/down.sh` 停止 PG 并保留卷。修改通过 Go 写入 PG，刷新或服务重启后仍保留。

分步骤备选（根目录执行，Go/Vite 分别占用独立终端）：

```bash
docker compose up -d
docker compose exec -T postgres pg_isready -U flagforge -d flagforge
bash scripts/migrate-up.sh
bash scripts/seed.sh
(cd backend && go run ./cmd/server)
# 另一个终端：
(cd frontend && npm run dev)
# 再开一个终端验收：
bash scripts/smoke.sh
```

PG 未就绪时重复 pg_isready，成功后再迁移。默认 DATABASE_URL：

```text
postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable
```

本地 PG 备选：使用有建库权限的管理员通过 psql 创建专用角色和库，然后跳过 Compose，只执行 migrate/seed/Go/Vite。`scripts/dev.sh` 始终启动 Compose，是 Docker 主路径。

```sql
CREATE ROLE flagforge LOGIN PASSWORD 'flagforge';
CREATE DATABASE flagforge OWNER flagforge;
```

```bash
export DATABASE_URL='postgres://flagforge:flagforge@localhost:5432/flagforge?sslmode=disable'
bash scripts/migrate-up.sh
bash scripts/seed.sh
# 保留 DATABASE_URL，在本终端起 Go；另一个终端起 Vite
(cd backend && go run ./cmd/server)
```

迁移创建 pgcrypto，数据库角色需有相应权限；受限实例可由管理员先安装该扩展。Go 可设 SERVER_ADDR，但 Vite `/api` 代理固定指向 `http://localhost:8787`，演示使用默认端口。大陆拉包失败可用 GOPROXY=https://goproxy.cn,direct 或 `npm ci --registry=https://registry.npmmirror.com`。

`frontend/.env.development` / `.env.production` 默认 VITE_USE_MSW=false。显式开发 mock：`(cd frontend && VITE_USE_MSW=true npm run dev)`。MSW 数据在内存中，刷新重置，不能证明 Go/PG 持久化。从旧 mock 切换时关闭旧页面并在新浏览器会话打开真实模式。

## 3. 数据模型

迁移在 backend/migrations/0001_init.up.sql / .down.sql，禁用 GORM AutoMigrate。

| 表 | 内容与关系 |
| --- | --- |
| flags | UUID、name、key、environment、enabled、default_value、description、created_at/updated_at |
| rules | UUID、flag_id 外键、attribute、operator、JSONB expected_value、roll_value、priority、时间戳 |
| histories | 自增 id、flag_id 外键、8 种 action、operator、JSONB summary、created_at；按 Flag/时间索引 |

`uq_flags_env_key UNIQUE(environment,key)` 拒绝同环境重复 key，允许跨环境复用。`uq_rules_flag_priority UNIQUE(flag_id,priority)` 拒绝同 Flag 重复优先级；DB 最终兜底并映射为 409。其他 CHECK 限定三环境、key 格式、名称/属性长度、equals/in、正整数 priority 与 history action。时间使用 TIMESTAMPTZ，API 输出 UTC，前端本地化显示。

Seed 首次初始化 4 Flag / 7 Rule / 4 flag_created 历史：结账 production/development、搜索 staging、停用 billing development。重跑不会覆盖已有 Flag、补回人工删掉的规则或重复写历史。

## 4. 规则评估语义

- **D1：数字越小越优先**。按 priority 升序，第一条命中立即返回 roll_value（可为 false）；后续规则不评估。全 miss 返回 default_value。
- **D2：重复优先级拒绝**。创建/编辑均 409 DUPLICATE_PRIORITY，UI 定位 priority 字段，避免隐藏二级排序改变结果。priority 为 1–2147483647 整数，可留空号。
- **D3：严格 JSON 类型**。equals 接受字符串/数字/布尔标量；in 接受非空标量数组，可混合类型，只比较同类型元素。`"1" ≠ 1`、`true ≠ "true"`；数字双侧归一 float64。期望值拒绝 null、对象、嵌套数组。
- **D4：停用先短路**。无论 default/规则为何，disabled 恒 false，reason=disabled，Trace 仅一条停用记录；停用时仍可维护规则。
- **D5：属性缺失跳过**。user_attributes 必须是 object，可含嵌套值；只读顶层键，不解析 a.b 路径。缺失记录 attribute_missing 后继续；对象/数组不能匹配标量。

Trace cause：matched、attribute_missing、type_mismatch、value_not_equal、value_not_in_list、short_circuited_by_disabled_flag。结果 reason：rule_hit/default/disabled。三个 seed 示例：

| flag_key / environment / user_attributes | 结果与解释 |
| --- | --- |
| checkout-redesign-2026 / production / `{"country":"CN","plan":"pro"}` | true，#2 命中；Trace 仅 #1 白名单 miss、#2 plan hit，#3 不评估 |
| checkout-redesign-2026 / production / `{"country":"CN","plan":"free","user_id":"9001"}` | false/default；#3 字符串9001不等于数字9001，三条全 miss |
| billing-portal / development / `{"plan":"pro"}` | false/disabled；default=true 且规则会 hit，仍仅一条短路 Trace |

development 同 key 结账 Flag 默认 true，但 country=CN 命中 #1 返回 false；评估必须同时指定 key/environment。

控制台在本次页面会话中保留状态：切到 Flag 列表或详情再返回，单次/批量草稿、已提交结果、错误和选中标签仍在，不会自动重新评估；刷新页面或关闭标签后重置。结果中的 Key、环境对应原请求；单次显示 API 评估时间，批量显示客户端模拟完成时间，编辑草稿不会改写已提交结果的标签。

单次和批量的 Flag Key 都可自由输入，也可从当前环境的已有 Flag 中选择（包含停用的 Flag，展示名称和状态）。候选加载完整分页列表；切换环境或重新聚焦输入框时刷新。环境变化不会改写输入的 Key；候选加载失败时仍可手动输入并评估，重新聚焦可重试。

## 5. API 摘要

GET /healthz；业务前缀 /api/v1。单资源成功直接返回资源 JSON 对象，分页 `{"items":[],"page":1,"page_size":20,"total":0}`，错误 `{"error":{"code":"...","message":"...","details":...}}`。

| 方法与路径（省略前缀） | 用途 |
| --- | --- |
| POST /flags；GET /flags | 新建；列表，environment/enabled/q/page/page_size，组合 AND |
| GET /flags/:id；PATCH /flags/:id | 详情与有序 rules；编辑 name/description/default_value，key/environment 不可变 |
| DELETE /flags/:id | 永久删除 Flag 及其规则/历史，成功返回空 204；非法 UUID 400、不存在或重复删除 404 FLAG_NOT_FOUND |
| PATCH /flags/:id/status | `{"enabled":false}` 启停 |
| POST /flags/:id/rules | 新建规则 |
| PATCH /flags/:id/rules/:rid；DELETE 同路径 | 部分更新；删除规则（204） |
| GET /flags/:id/history | created_at DESC/id DESC 分页，8 action 与 summary |
| POST /evaluate | flag_key + environment + user_attributes 单次评估 |
| POST /evaluate/batch | flag_key + environment + user_attributes_list（1–100 object） |
| POST /flags/:id/copy | `{"target_environment":"staging"}`；新 Flag 停用，保持 priority，源不变 |

```bash
curl -i -X POST http://localhost:8787/api/v1/flags \
  -H 'Content-Type: application/json' \
  -d '{"name":"Demo rollout","key":"demo-rollout","environment":"staging","enabled":true,"default_value":false}'
curl -sS -X POST http://localhost:8787/api/v1/evaluate \
  -H 'Content-Type: application/json' \
  -d '{"flag_key":"checkout-redesign-2026","environment":"production","user_attributes":{"country":"CN","plan":"pro"}}'
```

第二条返回 value=true、reason.type=rule_hit、reason.priority=2，Trace 两行。常见错误：非法 JSON 400 INVALID_JSON（位置）、非法输入400、资源不存在404 FLAG_NOT_FOUND/RULE_NOT_FOUND、重复 key/priority409。batch 任意非法组导致整体400，details.index 为0起；UI 转换为原 textarea 的1起行号，忽略空行，注释行视为非法 JSON。

## 6. 测试与验收

```bash
(cd backend && go test ./... -count=1)
(cd backend && go vet ./... && go build ./...)
(cd backend && go test ./internal/eval -count=1 -cover)
# 先启动 DB 并 migrate；显式设置连接，防止 integration 因未配置 Skip
export DATABASE_URL='postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable'
(cd backend && go test -tags=integration ./... -count=1 -v)
(cd frontend && npm test && npm run build && npm run lint)
# 已起 Go 后：
bash scripts/smoke.sh
# 自定义地址：BASE_URL=http://localhost:58025 bash scripts/smoke.sh
```

引擎单测覆盖全部15组语义与防御行为，statement coverage100%。真实 PG integration 覆盖约束、零值更新、分页、幂等 seed、全部 API、batch 查询次数/计数、精确历史diff、history/copy 失败事务回滚。API 用例随机隔离 schema 并清理；配置连接但连接失败会报错。

Smoke 使用 Bash/curl/已有 Node≥18 JSON断言，不需要 jq/Python。序列 health → 创建 → 同环境重复409 → 加规则 → hit/default → disable → false+disabled → history≥3 → batch.summary.total；失败非零。每次使用独立 key，**保留新建测试 Flag、规则和历史**，不删除已有数据。

前端新增 Node 内置测试与 jsdom，覆盖真实组件的导航状态、Key 候选及删除流程，使用网络替身隔离 HTTP；真实浏览器另外验证 Go/PG 联动。原始83条清单、干净重放、UI证据见 docs/ai-logs/T5-acceptance.md，命令汇总见 T5-report.md；本次三项修复证据见 docs/ai-logs/console-fixes-2026-10-10.md。`bash scripts/migrate-down.sh` 回退一版，随后 migrate-up；回退删除三张业务表/数据，保留 schema_migrations 和共享扩展。

## 7. 设计取舍

| 决策 | 原因 |
| --- | --- |
| D2 拒绝重复 priority | 单一可解释顺序，DB 处理并发撞号，UI 定位字段 |
| D3 严格类型 | 避免隐式转换误命中，Trace 解释失败 |
| D4 停用短路 | 总开关优先，false 可预测 |
| D6 key/environment 不可变 | 保护调用方锚点；显式400 IMMUTABLE_FIELD，另建或复制 |
| D7 Flag 永久删除 | 用户后续要求新增；确认后在事务中删除 Flag，外键级联删除规则与历史，无回收站/恢复或删除后历史查询 |
| D8 service 唯一事务边界 | 业务与历史同提交/回滚，真实 PG CHECK 失败验证 |
| D9 last-write-wins | 小团队低频编辑优先核心；无版本号/乐观锁 |
| D10 UTC | 统一写入/API口径，本地化只在显示层 |

纯函数引擎便于独立测试，SQL 迁移为 schema 事实源；MSW 契约解耦开发，最终演示走 Go/PG。顶层属性避免路径语言；batch 一次加载配置复用引擎，copy 复用事务/唯一约束。删除失败时整个事务回滚，不影响其他环境同 Key；删除成功后该环境可用同 Key 新建不同 UUID。删除既有 seed Flag 后，重新执行 seed 会重新创建该演示 Flag。

## 8. 已知限制

- 无登录/权限/真实用户审计身份，operator 固定 admin；默认密码为本地开发示例。
- 并发 last-write-wins，无乐观锁。仅布尔 Flag、equals/in、顶层属性；未做百分比灰度、嵌套路径、SDK、部署流水线等可选扩展。
- JSON数字 float64，超安全整数可能丢精度；大整数用户ID应使用字符串，并配置字符串规则。前端拒绝 Infinity 溢出。
- 前端自动化使用 Node/jsdom，不能替代完整跨浏览器端到端验证；build 存在 >500KB chunk提示，尚未分包优化。
- 本轮 `npm audit` 报告 4 项（3 moderate / 1 high），涉及锁定的 Vite 5.4.21 开发服务器、esbuild 与 react-router 6 依赖；包括 Windows 开发服务器路径绕过问题。修复建议涉及 major 升级，未在本次既定版本范围内升级；未声称 audit 零问题，当前验证范围为本地演示。
- 复制 curl 的生成/转义/点击反馈已有验证；实际操作系统剪贴板读回为空，尚未验证完整剪贴板写入，成功提示不能替代读回证据。
- 最终真实UI、重启持久化、PG删卷重建及新checkout按README启动已验证；83条功能清单证据见T5-acceptance。验证限于本报告环境，不表示所有平台或可选扩展均已覆盖。
- 48小时正式提交时限无法仅由Git证明；仓库交付不表示已向题目接收方正式提交。早期失败audit保留历史快照，闭环见T5报告。

## 9. AI 使用说明

AI 辅助设计、编码、检查与文档。session-main.md 保留原设计过程摘要；后续补齐由 Codex 主线调度 executor，每任务独立规格/质量审查，失败先复现再修复，复验后合并。归档是提示词、真实执行/审查摘要和命令证据，**不是完整逐字聊天导出**，不据摘要推断所有参与模型身份。

流程：主线需求/架构 → 8份任务提示词 → 依赖波次与并行 executor → 独立两阶段审查 → 合并 → 真实集成/归档。三层：完整任务提示词、执行报告、主线设计/调度过程。

```text
docs/ai-logs/
├── session-main.md                 # 原设计过程摘要
├── session-completion.md           # Codex后续补齐调度/集成
├── T0-prompt.md / T0-report.md
├── T0-wsl-features.json             # 安装环境检查记录
├── T1-prompt.md / T1-report.md
├── T2-prompt.md / T2-report.md
├── T3a-prompt.md / T3a-report.md
├── T3b-prompt.md / T3b-report.md
├── T4a-prompt.md / T4a-report.md
├── T4b-prompt.md / T4b-report.md
├── T5-prompt.md / T5-report.md
├── T5-acceptance.md                 # 全10组、实际83项
├── wave2-report.md                  # 早期波次汇总
└── requirements-validation-2026-10-10.md # 早期失败audit历史
```

docs/plan/ 保存统一设计、边界表和任务规范；frontend/evidence/ 保存报告注明来源的截图。历史报告中的待实现状态只反映当时基线；原始交付以 T5 为准，本次三项变更以 console-fixes-2026-10-10.md 为准。
