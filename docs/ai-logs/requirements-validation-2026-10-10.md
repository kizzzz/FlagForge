# 题目 A 完整要求验证报告

验证日期：2026-10-10（Asia/Shanghai）。

受测项目：`C:/Users/77230/FlagForge`；受测代码基线：`c5ef231`，分支 `codex/scaffold`。验证前工作区干净。本轮只增加验收报告与截图，没有补写业务实现。

## 结论

**当前项目不符合完整题目 A 的交付要求，不能按已完成的全栈应用提交。** 当前完成的是 T0 与波 2（T1/T2/T4a）。数据模型、迁移、seed、repo、纯函数规则引擎和前端部分页面已实现；Go 业务 API、规则管理、在线评估和操作历史展示尚未实现。

本轮重跑的构建、lint、引擎与真实 PostgreSQL 集成测试均通过，但这些通过结果只证明已实现模块；不证明缺失的全栈核心路径通过。此前 wave2-report 的完成结论仅适用于其明确的波次范围。

## 验证方法与环境

- 阅读 Go 入口、api/service、repo/model/eval/seed、SQL、前端路由/client/MSW/页面、启动脚本、README 和 AI 记录。
- 对当前 Go 服务与 Vite 代理发真实 HTTP 请求；合法、非法 JSON、同环境重复 key、资源查询、规则与评估请求均探测。
- 连接真实 PostgreSQL 查询迁移状态、数据、约束和历史；设置 DATABASE_URL 后跑全部 integration-tag 测试，防止因未配置数据库跳过。
- 重跑 Go build/vet、引擎覆盖率、前端 build/lint。
- 保留 5173 的现有演示环境，另在 5174 以进程级 `VITE_USE_MSW=false` 启动临时 Vite；新 origin 不沿用 5173 的 MSW worker。在真实浏览器查看列表、新建、详情和控制台。
- 临时浏览器和 5174 服务已结束；5173、8787 与 PostgreSQL 原环境保留。数据库测试使用回滚事务，本次 API 探测均未写入记录，最终数据仍为 4/7/4。

## 要求逐项判定

「部分」表示有底层或 mock 实现，但还没有满足题目的完整用户路径；不算通过。

| 题面要求 | 判定 | 证据与缺口 |
| --- | --- | --- |
| 前端 React、后端 Go、数据库 PostgreSQL | 通过 | React18/Vite、Gin/Go、实际 PG16.15 |
| 从零搭建、无提供的起始代码 | 有过程证据 | Git 从 scaffold 初始提交开始，T0 有安装/搭建记录；代码来源全过程无法仅凭仓库独立证明 |
| 48 小时内一次性提交 | 无法验证 | 仓库含提交日期，但缺完整题目接收与正式提交时间；本轮没有正式提交作业 |
| 推荐投入 6～8 小时 | 不作为通过门槛 | 是建议，不是强制要求；未据此判失败 |
| 必须 AI 辅助并附记录 | 通过 | docs/ai-logs 含主线设计记录、完整任务提示词、安装/执行报告和审查修复过程；题面允许相关信息，不强制逐字聊天导出 |
| 可运行的完整源码 | 部分 | 工程可编译启动，但完整业务应用缺失 |
| SQL 迁移或建表脚本 | 通过 | 0001_init up/down；运行库 version1、dirty=false |
| 必要核心业务自动化测试 | 通过（引擎） | eval 测试覆盖优先级、第一命中、默认值、停用、equals/in、数字归一和严格类型；覆盖率100% |
| 前端调用自己实现的 Go API | **未通过** | 默认 MSW 拦截业务请求；关闭后列表和创建真实请求404，Go仅实现healthz |
| 核心业务数据持久化到 PG | 部分 | PG有演示数据和可工作的repo；前端业务变更仍写入MSW内存，无法通过Go持久化 |
| Flag 包含题面全部字段 | 通过（模型） | flags包含name/key/environment/enabled/default_value/created_at/updated_at，另有description |
| 同环境 key 唯一、跨环境可复用 | 通过（数据库） | UNIQUE(environment,key)，PG中 checkout-redesign-2026 同时存在production/development，真实唯一冲突测试通过 |
| 列表、搜索/筛选、新建、编辑、启停 | 部分 | 前端和repo已实现，mock模式可演示；真实业务API全部404 |
| 查看 Flag 详情 | **未通过** | /flags/:id 只显示「Flag 详情 / T4b 实现」 |
| 规则属性名/equals或in/期望值/返回值/优先级 | 部分 | rules模型、repo、seed和引擎有这些字段；没有可用规则管理界面与API |
| 第一命中即停，无命中返回默认值，停用始终false | 通过（纯函数）/未完成在线路径 | 引擎实测通过；没有数据库加载→评估API→页面的全链路 |
| 明确优先级方向并在 UI 与 README 写清 | 部分 | 引擎/DB规则排序为数字越小越优先，设计记录说明；规则界面和README缺说明 |
| 重复优先级确定性策略、前后端README一致 | 部分 | DB拒绝同flag重复priority，repo映射DUPLICATE_PRIORITY；没有后端409响应、规则UI和README声明 |
| 在线输入Key/环境/JSON并调用后端评估 | **未通过** | /console占位；POST /api/v1/evaluate 404 |
| 展示结果/是否命中/哪条规则或默认值 | **未通过** | 引擎Result存在，但API/UI展示没有实现 |
| JSON错误、Flag不存在等清晰反馈 | 部分 | 前端基础校验与错误反馈有；Go对合法、非法JSON、不存在和业务冲突均返回通用text/plain404，无业务错误信封 |
| 记录创建、修改、规则增删改、启停历史 | 部分 | 表/repo支持，seed只有4条flag_created；没有业务请求自动记录这些操作 |
| 历史展示类型、时间、必要摘要 | **未通过** | history API404，详情没有历史展示 |
| 同请求业务变更与历史原子一致 | **未通过（尚未实现）** | seed Transaction及其回滚测试通过；service仍doc.go占位，无业务请求事务边界，不能用seed事务替代此要求 |
| 四类最低页面与清晰状态/校验反馈 | 部分 | 列表及新建/编辑有loading/空态/校验/反馈；详情规则页、评估页缺失 |
| 适量演示数据 | 通过（PG） | 实际4 flags/7 rules/4 histories；MSW的26条不计入PG演示持久化证明 |
| README启动、已完成/未完成、已知限制 | 通过当前阶段说明 | 启动命令和mock/后续范围如实说明；生产chunk等限制主要在执行报告 |
| README数据模型、规则优先级/匹配语义、设计取舍 | **部分/待补全** | README没有完整章节；设计文档有内容，但题面要求在README说明，不能仅靠计划文档替代 |
| 根据README本地完整复现 | 部分 | 本机依赖/容器与health可运行，脚本有迁移和seed；关闭mock真实核心路径失败，未在完全空环境重建验证 |
| 可选扩展 | 不作为门槛 | 未实现可选项不扣为必做失败；本轮不要求复制、批量、登录、部署或前端自动化测试 |

## 本轮执行命令与实测结果

### 构建与自动化测试

在 backend 设置真实 DATABASE_URL 后依次执行：

```powershell
$env:DATABASE_URL='postgres://flagforge:flagforge@localhost:55432/flagforge?sslmode=disable'
go build ./...
go vet ./...
go test -tags=integration ./... -count=1
go test ./internal/eval/ -count=1 -cover
```

全部 exit0，输出摘要：

```text
? flagforge/cmd/server       [no test files]
? flagforge/internal/api     [no test files]
? flagforge/internal/config  [no test files]
? flagforge/internal/dto     [no test files]
ok flagforge/internal/eval   0.174s
? flagforge/internal/model   [no test files]
ok flagforge/internal/repo   0.644s
? flagforge/internal/service [no test files]
ok flagforge/seed            0.596s
ok flagforge/internal/eval   0.176s coverage: 100.0% of statements
```

repo集成测试覆盖零值CRUD、组合筛选、分页/排序、真实PG唯一冲突；seed测试覆盖两次幂等、不覆盖已有数据、历史写入失败整批回滚。测试源码读取确认数据库连接错误会Fatal；本轮不是未配置DATABASE_URL后的Skip。api和service没有测试也没有业务实现。

frontend执行 `npm run build`、`npm run lint`：均exit0，1475 modules，13.43s；bundle1,070.91kB、gzip340.81kB。存在>500kB chunk警告，作为优化项记录，未将其当作题面阻断。

### 真实 HTTP 验证

下表全部为直接网络请求，未经过浏览器MSW。资源路径使用已知示例ID；无论该ID是否真实存在，Go源码只有healthz路由，返回的是路由缺失，而不是题面要求的资源不存在错误。

| 请求 | 本轮响应 | 验收意义 |
| --- | --- | --- |
| GET :8787/healthz | 200 application/json，{"status":"ok"} | 基础服务运行 |
| GET /api/v1/flags | 404 text/plain，404 page not found | 列表未实现 |
| GET /api/v1/flags?environment=production&q=checkout | 同上404 | 筛选未实现 |
| POST /api/v1/flags，合法audit-probe | 同上404 | 创建未实现 |
| POST /api/v1/flags，非法JSON {"name":} | 同上404 | 语法错误400未实现 |
| POST /api/v1/flags，已有production checkout key | 同上404 | 业务冲突409未实现 |
| GET /api/v1/flags/:id | 同上404 | 详情未实现 |
| PATCH /api/v1/flags/:id | 同上404 | 编辑未实现 |
| PATCH /api/v1/flags/:id/status | 同上404 | 启停未实现 |
| GET /api/v1/flags/:id/history | 同上404 | 历史未实现 |
| POST /api/v1/flags/:id/rules | 同上404 | 规则创建未实现 |
| PATCH /api/v1/flags/:id/rules/:rid | 同上404 | 规则编辑未实现 |
| DELETE /api/v1/flags/:id/rules/:rid | 同上404 | 规则删除未实现 |
| POST /api/v1/evaluate，有效key与plan=pro | 同上404 | 正常评估未实现 |
| POST /api/v1/evaluate，missing-flag | 同上404 | 无业务FLAG_NOT_FOUND反馈 |
| GET :5173/api/v1/flags（Vite代理） | 同上404 | 代理确实通向Go，目标业务路由缺失 |

源码佐证：[main.go](../../backend/cmd/server/main.go) 只注册healthz且将databaseURL赋值给 `_`；[api/doc.go](../../backend/internal/api/doc.go)、[service/doc.go](../../backend/internal/service/doc.go) 均仅占位。

### PostgreSQL 验证

```text
PostgreSQL 16.15
schema_migrations: 1|f
flags|rules|histories counts: 4|7|4
uq_flags_env_key: UNIQUE (environment, key)
uq_rules_flag_priority: UNIQUE (flag_id, priority)
development|billing-portal|f|t
development|checkout-redesign-2026|t|t
production|checkout-redesign-2026|t|f
staging|new-search-rank|t|f
history actions: flag_created|4
audit-probe / requirements-audit-probe persisted flags: 0
```

没有删除卷、没有重置seed，没有把当前PG中的四条创建历史误报为全部业务操作历史完成。

### 本轮实际浏览器验证

临时服务启动：PowerShell `$env:VITE_USE_MSW='false'; npm run dev -- --port 5174`。使用新临时IAB tab，实际操作结果：

- 列表显示「列表加载失败」「暂时无法获取 Flag」，错误message为HTTP404。
- 新建页填写Bad-Key并提交，空名称提示「请输入名称」、key提示格式错误。
- 改为合法名称与requirements-audit-probe提交，显示HTTP404，保留表单输入且仍在/flags/new。没有创建PG数据。
- /console只显示「评估控制台 / T4b 实现」，没有Key、环境、JSON输入和结果区。
- /flags/:id只显示「Flag 详情 / T4b 实现」，没有规则管理与历史。

截图：

- [关闭mock的真实列表失败](../../frontend/evidence/Compliance-real-api-failure.png)
- [控制台占位](../../frontend/evidence/Compliance-console-placeholder.png)
- [详情占位](../../frontend/evidence/Compliance-detail-placeholder.png)
- [真实创建失败后表单保留](../../frontend/evidence/Compliance-create-real-api-failure.png)

创建截图在toast消失后捕获，仅证明表单保留；404错误由当场DOM snapshot和HTTP探测确认。没有以截图证明其中不可见的信息。

## 提交前必须补齐的工作

1. T3a：实现Flag列表/新建/详情/编辑/启停/历史Go API，统一400/404/409反馈，业务变更与历史同一PG事务。
2. T3b必做部分：规则增删改、重复priority409、单次评估、合法JSON及用户属性校验、按key+环境加载PG配置。可选批量/复制不替代这些必做项。
3. T4b必做部分：详情、规则管理、历史与评估控制台；显式展示优先级方向和重复策略。
4. 默认前端接真实Go API；MSW保留为显式开发选项。验证创建/编辑/启停/规则变更在刷新、服务重启后仍持久化，并验证历史与业务一致。
5. T5：补充实际API集成/原子性失败用例和端到端smoke，按README验证完整启动链路；README写清数据模型、全部规则语义、取舍和最终完成/未完成项，保留AI日志。

本次是要求核验，未开始实现上述缺口；报告记录的是当前实际状态。
